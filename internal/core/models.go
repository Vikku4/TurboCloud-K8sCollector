package core

import "time"

type Cluster struct {
	ID             string
	TenantID       string
	CloudAccountID *string
	Provider       string
	ExternalID     string
	Name           string
	Region         string
}

type Namespace struct {
	ID        string
	TenantID  string
	ClusterID string
	Name      string
}

type Workload struct {
	ID          string
	TenantID    string
	ClusterID   string
	NamespaceID string
	Name        string
	Kind        string
	Labels      map[string]string
}

type MetricsSample struct {
	TenantID         string
	WorkloadID       string
	Timestamp        time.Time
	CPUMcores        int64
	CPURequestMcores int64
	MemBytes         int64
	MemRequestBytes  int64
}

// Summary query

type SummaryParams struct {
	WindowMinutes int
	ClusterID     string
	Namespace     string
}

type SummaryRow struct {
	ClusterID    string `json:"cluster_id"`
	ClusterName  string `json:"cluster_name"`
	Namespace    string `json:"namespace"`
	WorkloadName string `json:"workload_name"`
	Kind         string `json:"kind"`

	AvgCPUMcores float64 `json:"avg_cpu_mcores"`
	MaxCPUMcores int64   `json:"max_cpu_mcores"`
	AvgMemBytes  float64 `json:"avg_mem_bytes"`
	MaxMemBytes  int64   `json:"max_mem_bytes"`
	SampleCount  int64   `json:"sample_count"`
}
