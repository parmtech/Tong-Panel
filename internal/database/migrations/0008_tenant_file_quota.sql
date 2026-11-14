-- Persistent per-tenant byte reservations for the opt-in Files API.
-- Quotas stay unconfigured until a trusted reconciler supplies existing usage.
CREATE TABLE tenant_file_quotas (
    tenant_id TEXT PRIMARY KEY REFERENCES tenants(id) ON DELETE RESTRICT,
    limit_bytes INTEGER NOT NULL CHECK (limit_bytes >= 0),
    used_bytes INTEGER NOT NULL CHECK (used_bytes >= 0),
    reserved_bytes INTEGER NOT NULL DEFAULT 0 CHECK (reserved_bytes >= 0),
    reconciled_at INTEGER NOT NULL CHECK (reconciled_at > 0),
    CHECK (used_bytes <= limit_bytes),
    CHECK (reserved_bytes <= limit_bytes - used_bytes)
);

CREATE TABLE tenant_file_quota_reservations (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenant_file_quotas(tenant_id) ON DELETE RESTRICT,
    reserved_bytes INTEGER NOT NULL CHECK (reserved_bytes >= 0),
    created_at INTEGER NOT NULL CHECK (created_at > 0)
);

CREATE INDEX tenant_file_quota_reservations_tenant
    ON tenant_file_quota_reservations(tenant_id);
