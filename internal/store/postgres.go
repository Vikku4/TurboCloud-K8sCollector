package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/turbocloud-io/k8s-collector/internal/core"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func OpenPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return pool, nil
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

var _ core.Store = (*PostgresStore)(nil)

// ---- cluster lookup by token ----

func (s *PostgresStore) GetClusterByToken(ctx context.Context, token string) (*core.Cluster, error) {
	const q = `
		SELECT id, tenant_id, cloud_account_id, provider, external_id, name, region
		FROM clusters
		WHERE cluster_token = $1
	`
	row := s.db.QueryRow(ctx, q, token)
	var c core.Cluster
	var cloudAccountID *string
	if err := row.Scan(
		&c.ID, &c.TenantID, &cloudAccountID,
		&c.Provider, &c.ExternalID, &c.Name, &c.Region,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrInvalidToken
		}
		return nil, err
	}
	c.CloudAccountID = cloudAccountID
	return &c, nil
}

// UpsertCluster updates metadata (name, region, provider, external_id).
func (s *PostgresStore) UpsertCluster(ctx context.Context, c core.Cluster) (*core.Cluster, error) {
	const q = `
		UPDATE clusters
		SET name = COALESCE(NULLIF($2, ''), name),
		    region = COALESCE(NULLIF($3, ''), region),
		    provider = COALESCE(NULLIF($4, ''), provider),
		    external_id = COALESCE(NULLIF($5, ''), external_id),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, tenant_id, cloud_account_id, provider, external_id, name, region
	`
	row := s.db.QueryRow(ctx, q, c.ID, c.Name, c.Region, c.Provider, c.ExternalID)
	var res core.Cluster
	var cloudAccountID *string
	if err := row.Scan(
		&res.ID, &res.TenantID, &cloudAccountID,
		&res.Provider, &res.ExternalID, &res.Name, &res.Region,
	); err != nil {
		return nil, err
	}
	res.CloudAccountID = cloudAccountID
	return &res, nil
}

// ---- namespace & workload upserts ----

func (s *PostgresStore) UpsertNamespace(ctx context.Context, tenantID, clusterID, name string) (*core.Namespace, error) {
	const q = `
		INSERT INTO namespaces (tenant_id, cluster_id, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (cluster_id, name) DO UPDATE
		  SET name = EXCLUDED.name
		RETURNING id, tenant_id, cluster_id, name
	`
	row := s.db.QueryRow(ctx, q, tenantID, clusterID, name)
	var ns core.Namespace
	if err := row.Scan(&ns.ID, &ns.TenantID, &ns.ClusterID, &ns.Name); err != nil {
		return nil, err
	}
	return &ns, nil
}

func (s *PostgresStore) UpsertWorkload(ctx context.Context, w core.Workload) (*core.Workload, error) {
	const q = `
		INSERT INTO workloads (tenant_id, cluster_id, namespace_id, name, kind, labels)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		ON CONFLICT (cluster_id, namespace_id, name, kind) DO UPDATE
		  SET labels = EXCLUDED.labels
		RETURNING id, tenant_id, cluster_id, namespace_id, name, kind, labels
	`
	labelsJSON, err := json.Marshal(w.Labels)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRow(ctx, q,
		w.TenantID, w.ClusterID, w.NamespaceID,
		w.Name, w.Kind, string(labelsJSON),
	)
	var res core.Workload
	var labelsRaw []byte
	if err := row.Scan(&res.ID, &res.TenantID, &res.ClusterID,
		&res.NamespaceID, &res.Name, &res.Kind, &labelsRaw); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(labelsRaw, &res.Labels)
	return &res, nil
}

// ---- metrics insert ----

func (s *PostgresStore) InsertMetrics(ctx context.Context, samples []core.MetricsSample) error {
	const q = `
		INSERT INTO workload_metrics
		    (tenant_id, workload_id, ts,
		     cpu_mcores, cpu_request_mcores,
		     mem_bytes, mem_request_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	batch := &pgx.Batch{}
	for _, sm := range samples {
		batch.Queue(q,
			sm.TenantID, sm.WorkloadID, sm.Timestamp,
			sm.CPUMcores, sm.CPURequestMcores,
			sm.MemBytes, sm.MemRequestBytes,
		)
	}
	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for range samples {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// ---- summary query ----

func (s *PostgresStore) GetSummary(ctx context.Context, tenantID string, params core.SummaryParams) ([]core.SummaryRow, error) {
	windowStart := time.Now().Add(-time.Duration(params.WindowMinutes) * time.Minute)

	const q = `
		SELECT
		  c.id          AS cluster_id,
		  c.name        AS cluster_name,
		  ns.name       AS namespace,
		  w.name        AS workload_name,
		  w.kind        AS kind,
		  AVG(m.cpu_mcores)::float8 AS avg_cpu_mcores,
		  MAX(m.cpu_mcores)         AS max_cpu_mcores,
		  AVG(m.mem_bytes)::float8  AS avg_mem_bytes,
		  MAX(m.mem_bytes)          AS max_mem_bytes,
		  COUNT(*)                  AS sample_count
		FROM workload_metrics m
		JOIN workloads w   ON m.workload_id = w.id
		JOIN namespaces ns ON w.namespace_id = ns.id
		JOIN clusters c    ON w.cluster_id = c.id
		WHERE m.tenant_id = $1
		  AND m.ts >= $2
		  AND ($3 = '' OR c.id::text = $3)
		  AND ($4 = '' OR ns.name = $4)
		GROUP BY c.id, c.name, ns.name, w.name, w.kind
		ORDER BY c.name, ns.name, w.name
	`
	rows, err := s.db.Query(ctx, q, tenantID, windowStart, params.ClusterID, params.Namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []core.SummaryRow
	for rows.Next() {
		var r core.SummaryRow
		if err := rows.Scan(
			&r.ClusterID, &r.ClusterName, &r.Namespace, &r.WorkloadName, &r.Kind,
			&r.AvgCPUMcores, &r.MaxCPUMcores,
			&r.AvgMemBytes, &r.MaxMemBytes,
			&r.SampleCount,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
