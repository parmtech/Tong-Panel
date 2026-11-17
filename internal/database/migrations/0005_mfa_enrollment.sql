-- P1 prototype: encrypted TOTP seed pending/active state.
-- NO KEY MATERIAL or plaintext secret is held in SQLite.
-- Public enrollment/login is intentionally absent pending ADR-009 security review.
CREATE TABLE mfa_totp_enrollments (
  identity_id TEXT PRIMARY KEY NOT NULL REFERENCES identities(id) ON DELETE RESTRICT,
  secret_envelope TEXT NOT NULL CHECK(
    length(secret_envelope) BETWEEN 35 AND 256 AND
    secret_envelope LIKE 'v1:%'
  ),
  status TEXT NOT NULL CHECK(status IN ('pending','active')),
  pending_expires_at TEXT NOT NULL,
  activated_at TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  CHECK((status='pending' AND activated_at IS NULL)
     OR (status='active' AND activated_at IS NOT NULL))
) STRICT;
