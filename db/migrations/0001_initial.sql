-- Tong Panel P1: metadata schema proposal, SQLite >= 3.37.
-- IMPORTANT: Set PRAGMA foreign_keys=ON on EVERY SQLite connection.
-- This schema is not wired to the live Go API or the development JSON job journal.
-- All rows and cross-tenant access still require RBAC in API and job workers.
PRAGMA foreign_keys = ON;
BEGIN IMMEDIATE;
CREATE TABLE IF NOT EXISTS tenants (
  id TEXT PRIMARY KEY NOT NULL CHECK(length(id) BETWEEN 1 AND 128),
  name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 160),
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','suspended')),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
) STRICT;
CREATE TABLE IF NOT EXISTS identities (
  id TEXT PRIMARY KEY NOT NULL CHECK(length(id) BETWEEN 1 AND 128),
  username TEXT NOT NULL COLLATE NOCASE UNIQUE CHECK(length(username) BETWEEN 3 AND 120),
  password_phc TEXT CHECK(password_phc IS NULL OR password_phc LIKE '$argon2id$%'),
  disabled INTEGER NOT NULL DEFAULT 0 CHECK(disabled IN (0,1)),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
) STRICT;
CREATE TABLE IF NOT EXISTS memberships (
  tenant_id TEXT NOT NULL,
  identity_id TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('owner','admin','reseller','customer','auditor')),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  PRIMARY KEY (tenant_id, identity_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
  FOREIGN KEY (identity_id) REFERENCES identities(id) ON DELETE RESTRICT
) STRICT;
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY NOT NULL,
  tenant_id TEXT NOT NULL,
  identity_id TEXT NOT NULL,
  token_sha256 BLOB NOT NULL UNIQUE CHECK(length(token_sha256) = 32),
  csrf_sha256 BLOB NOT NULL CHECK(length(csrf_sha256) = 32),
  expires_at TEXT NOT NULL,
  revoked_at TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (tenant_id, identity_id) REFERENCES memberships(tenant_id, identity_id) ON DELETE CASCADE
) STRICT;
CREATE INDEX IF NOT EXISTS sessions_tenant_identity ON sessions(tenant_id, identity_id, expires_at);
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY NOT NULL,
  tenant_id TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('validation.noop')),
  state TEXT NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','leased','running','completed','failed','cancelled')),
  idempotency_key TEXT NOT NULL CHECK(length(idempotency_key) BETWEEN 8 AND 128),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  lease_until TEXT,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count >= 0),
  FOREIGN KEY (tenant_id, requested_by) REFERENCES memberships(tenant_id, identity_id) ON DELETE RESTRICT,
  UNIQUE(tenant_id, idempotency_key)
) STRICT;
CREATE INDEX IF NOT EXISTS jobs_pending ON jobs(state, lease_until, created_at);
CREATE INDEX IF NOT EXISTS jobs_tenant_created ON jobs(tenant_id, created_at);
CREATE TABLE IF NOT EXISTS job_attempts (
  id TEXT PRIMARY KEY NOT NULL,
  job_id TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  finished_at TEXT,
  status TEXT NOT NULL CHECK(status IN ('started','completed','failed')),
  error_code TEXT,
  FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE RESTRICT
) STRICT;
CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY NOT NULL,
  tenant_id TEXT NOT NULL,
  actor_identity_id TEXT,
  action TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  request_id TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, actor_identity_id) REFERENCES memberships(tenant_id, identity_id) ON DELETE RESTRICT
) STRICT;
CREATE INDEX IF NOT EXISTS audit_tenant_date ON audit_events(tenant_id, created_at);
PRAGMA user_version = 1;
COMMIT;
