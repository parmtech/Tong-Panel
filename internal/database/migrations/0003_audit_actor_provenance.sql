-- v3: make audit actor provenance explicit; legacy worker events had incorrectly
-- inherited the original requesting user's identity. That cannot be trusted.
-- Preserve those historical events but mark their provenance as unknown.
ALTER TABLE audit_events ADD COLUMN actor_kind TEXT NOT NULL DEFAULT 'user'
  CHECK(actor_kind IN ('user','worker','system','unknown'));
ALTER TABLE audit_events ADD COLUMN actor_ref TEXT
  CHECK(actor_ref IS NULL OR length(actor_ref) BETWEEN 1 AND 128);

UPDATE audit_events
   SET actor_identity_id=NULL, actor_kind='unknown', actor_ref=NULL
 WHERE action IN ('job.leased','job.started','job.lease_renewed',
                  'job.completed','job.retry_scheduled','job.failed',
                  'job.lease_expired');

-- Protect all newly written actor metadata even if a caller bypasses the Go API.
CREATE TRIGGER audit_actor_guard_insert BEFORE INSERT ON audit_events
WHEN (NEW.actor_kind='user' AND (NEW.actor_identity_id IS NULL OR NEW.actor_ref IS NOT NULL))
  OR (NEW.actor_kind='worker' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NULL))
  OR (NEW.actor_kind='system' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NULL))
  OR (NEW.actor_kind='unknown' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NOT NULL))
BEGIN
  SELECT RAISE(ABORT, 'audit actor provenance inconsistent');
END;

CREATE TRIGGER audit_actor_guard_update BEFORE UPDATE ON audit_events
WHEN (NEW.actor_kind='user' AND (NEW.actor_identity_id IS NULL OR NEW.actor_ref IS NOT NULL))
  OR (NEW.actor_kind='worker' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NULL))
  OR (NEW.actor_kind='system' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NULL))
  OR (NEW.actor_kind='unknown' AND (NEW.actor_identity_id IS NOT NULL OR NEW.actor_ref IS NOT NULL))
BEGIN
  SELECT RAISE(ABORT, 'audit actor provenance inconsistent');
END;
