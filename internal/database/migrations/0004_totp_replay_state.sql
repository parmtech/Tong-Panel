-- P1 MFA replay barrier (standalone primitive; no enrollment/login).
-- The secret is NOT stored in this table. Accepted time steps are global per
-- identity (not per tenant), preventing code reuse across tenant memberships.
CREATE TABLE totp_replay_state (
  identity_id TEXT PRIMARY KEY NOT NULL
    REFERENCES identities(id) ON DELETE RESTRICT,
  last_accepted_step INTEGER NOT NULL CHECK(last_accepted_step >= 0),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
) STRICT;
