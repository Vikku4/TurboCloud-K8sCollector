CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Optional: local cloud_accounts table for this service
CREATE TABLE cloud_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    provider TEXT NOT NULL,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE clusters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    cloud_account_id UUID REFERENCES cloud_accounts(id),
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    region TEXT,
    cluster_token TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE namespaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    cluster_id UUID NOT NULL REFERENCES clusters(id),
    name TEXT NOT NULL,
    UNIQUE (cluster_id, name)
);

CREATE TABLE workloads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    cluster_id UUID NOT NULL REFERENCES clusters(id),
    namespace_id UUID NOT NULL REFERENCES namespaces(id),
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (cluster_id, namespace_id, name, kind)
);

CREATE TABLE workload_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workload_id UUID NOT NULL REFERENCES workloads(id),
    ts TIMESTAMPTZ NOT NULL,
    cpu_mcores BIGINT NOT NULL,
    cpu_request_mcores BIGINT,
    mem_bytes BIGINT NOT NULL,
    mem_request_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workload_metrics_workload_ts ON workload_metrics (workload_id, ts DESC);
CREATE INDEX idx_workload_metrics_tenant_ts   ON workload_metrics (tenant_id, ts DESC);
