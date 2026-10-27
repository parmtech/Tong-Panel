-- Add capability-fenced leases, bounded retry scheduling and lease consistency.
ALTER TABLE jobs ADD COLUMN available_at TEXT;
ALTER TABLE jobs ADD COLUMN lease_owner TEXT CHECK(lease_owner IS NULL OR length(lease_owner) BETWEEN 1 AND 128);
ALTER TABLE jobs ADD COLUMN lease_token_sha256 BLOB CHECK(lease_token_sha256 IS NULL OR length(lease_token_sha256) = 32);

-- v1 could represent running without a worker attempt. That state was only a
-- manual prototype transition and never executed host work; requeue it safely.
UPDATE jobs SET state='queued', updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
  WHERE state IN ('leased','running');

CREATE INDEX jobs_schedulable ON jobs(state, available_at, lease_until, created_at);
CREATE TRIGGER jobs_lease_fields_insert
BEFORE INSERT ON jobs
WHEN (NEW.state IN ('leased','running') AND
      (NEW.lease_until IS NULL OR NEW.lease_owner IS NULL OR NEW.lease_token_sha256 IS NULL))
  OR (NEW.state NOT IN ('leased','running') AND
      (NEW.lease_until IS NOT NULL OR NEW.lease_owner IS NOT NULL OR NEW.lease_token_sha256 IS NOT NULL))
BEGIN
  SELECT RAISE(ABORT, 'job lease fields do not match state');
END;
CREATE TRIGGER jobs_lease_fields_update
BEFORE UPDATE OF state, lease_until, lease_owner, lease_token_sha256 ON jobs
WHEN (NEW.state IN ('leased','running') AND
      (NEW.lease_until IS NULL OR NEW.lease_owner IS NULL OR NEW.lease_token_sha256 IS NULL))
  OR (NEW.state NOT IN ('leased','running') AND
      (NEW.lease_until IS NOT NULL OR NEW.lease_owner IS NOT NULL OR NEW.lease_token_sha256 IS NOT NULL))
BEGIN
  SELECT RAISE(ABORT, 'job lease fields do not match state');
END;
