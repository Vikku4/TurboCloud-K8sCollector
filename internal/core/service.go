package core

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidToken = errors.New("invalid cluster token")

// Store is implemented by the Postgres layer.
type Store interface {
	GetClusterByToken(ctx context.Context, token string) (*Cluster, error)
	UpsertCluster(ctx context.Context, c Cluster) (*Cluster, error)
	UpsertNamespace(ctx context.Context, tenantID, clusterID, name string) (*Namespace, error)
	UpsertWorkload(ctx context.Context, w Workload) (*Workload, error)
	InsertMetrics(ctx context.Context, samples []MetricsSample) error
	GetSummary(ctx context.Context, tenantID string, params SummaryParams) ([]SummaryRow, error)
}

// Service is where business rules live.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// ==== Ingest DTOs (from Agent) ====

type IngestRequest struct {
	Cluster   IngestCluster    `json:"cluster"`
	Workloads []IngestWorkload `json:"workloads"`
}

type IngestCluster struct {
	ExternalID             string `json:"external_id"`
	Name                   string `json:"name"`
	CloudProvider          string `json:"cloud_provider"`
	CloudAccountExternalID string `json:"cloud_account_external_id"`
	Region                 string `json:"region"`
}

type IngestWorkload struct {
	Namespace    string            `json:"namespace"`
	WorkloadName string            `json:"workload_name"`
	Kind         string            `json:"kind"`
	Labels       map[string]string `json:"labels"`
	Metrics      IngestMetrics     `json:"metrics"`
}

type IngestMetrics struct {
	Timestamp        string `json:"timestamp"` // RFC3339
	CPUMcores        int64  `json:"cpu_mcores"`
	CPURequestMcores int64  `json:"cpu_request_mcores"`
	MemBytes         int64  `json:"mem_bytes"`
	MemRequestBytes  int64  `json:"mem_request_bytes"`
}

// IngestMetricsBatch handles full batch from agent.
func (s *Service) IngestMetricsBatch(ctx context.Context, clusterToken string, req IngestRequest) error {
	cluster, err := s.store.GetClusterByToken(ctx, clusterToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			return ErrInvalidToken
		}
		return err
	}
	tenantID := cluster.TenantID

	// Keep cluster metadata up to date.
	_, err = s.store.UpsertCluster(ctx, Cluster{
		ID:         cluster.ID,
		TenantID:   tenantID,
		Provider:   req.Cluster.CloudProvider,
		ExternalID: req.Cluster.ExternalID,
		Name:       req.Cluster.Name,
		Region:     req.Cluster.Region,
	})
	if err != nil {
		return err
	}

	var samples []MetricsSample
	for _, wl := range req.Workloads {
		ns, err := s.store.UpsertNamespace(ctx, tenantID, cluster.ID, wl.Namespace)
		if err != nil {
			return err
		}
		w, err := s.store.UpsertWorkload(ctx, Workload{
			TenantID:    tenantID,
			ClusterID:   cluster.ID,
			NamespaceID: ns.ID,
			Name:        wl.WorkloadName,
			Kind:        wl.Kind,
			Labels:      wl.Labels,
		})
		if err != nil {
			return err
		}

		ts, err := time.Parse(time.RFC3339, wl.Metrics.Timestamp)
		if err != nil {
			return err
		}

		samples = append(samples, MetricsSample{
			TenantID:         tenantID,
			WorkloadID:       w.ID,
			Timestamp:        ts,
			CPUMcores:        wl.Metrics.CPUMcores,
			CPURequestMcores: wl.Metrics.CPURequestMcores,
			MemBytes:         wl.Metrics.MemBytes,
			MemRequestBytes:  wl.Metrics.MemRequestBytes,
		})
	}

	if len(samples) == 0 {
		return nil
	}
	return s.store.InsertMetrics(ctx, samples)
}

// GetUsageSummary wraps store with a safety default.
func (s *Service) GetUsageSummary(ctx context.Context, tenantID string, params SummaryParams) ([]SummaryRow, error) {
	if params.WindowMinutes <= 0 {
		params.WindowMinutes = 60
	}
	return s.store.GetSummary(ctx, tenantID, params)
}
