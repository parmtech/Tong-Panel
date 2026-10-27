-- P1 SQLite session lifecycle: add a server-owned idle deadline.
-- Existing sessions from older drafts are intentionally invalidated because
-- their prior activity history is unknown. New rows must set a future deadline.
ALTER TABLE sessions ADD COLUMN idle_deadline_unix INTEGER NOT NULL DEFAULT 0
  CHECK(idle_deadline_unix >= 0);
UPDATE sessions SET revoked_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
  WHERE revoked_at IS NULL;
CREATE INDEX sessions_active_idle ON sessions(idle_deadline_unix)
  WHERE revoked_at IS NULL;
