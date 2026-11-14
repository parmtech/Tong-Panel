-- Internal recovery primitive only; pepper remains outside the database.
CREATE TABLE mfa_recovery_sets (
  tenant_id TEXT NOT NULL,
  identity_id TEXT NOT NULL,
  enrollment_sha256 BLOB NOT NULL CHECK(length(enrollment_sha256)=32),
  window_started_unix INTEGER NOT NULL CHECK(window_started_unix>=0),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5),
  PRIMARY KEY(tenant_id,identity_id),
  FOREIGN KEY(tenant_id,identity_id) REFERENCES memberships(tenant_id,identity_id) ON DELETE CASCADE
) STRICT;
CREATE TABLE mfa_recovery_codes (
  tenant_id TEXT NOT NULL,
  identity_id TEXT NOT NULL,
  code_sha256 BLOB NOT NULL CHECK(length(code_sha256)=32),
  PRIMARY KEY(tenant_id,identity_id,code_sha256),
  FOREIGN KEY(tenant_id,identity_id) REFERENCES mfa_recovery_sets(tenant_id,identity_id) ON DELETE CASCADE
) STRICT;
