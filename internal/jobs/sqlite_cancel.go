package jobs

import (
	"database/sql"
	"errors"
)

// Cancelled denotes a queued job withdrawn before any worker lease.
// In-flight jobs cannot be cancelled safely without host-side fencing.
const Cancelled = "cancelled"

// ErrJobNotCancelable means the job has already been leased or finalized.
var ErrJobNotCancelable = errors.New("job is not safely cancellable")

// CancelFor atomically withdraws only the active requester's own queued job.
// It uses the same BEGIN IMMEDIATE lock as ClaimNext, preventing cancel/claim
// from both succeeding. It never sends a signal to workers or the Linux host.
// A repeat cancel is idempotent without a duplicate audit record.
func (s *SQLiteStore) CancelFor(tenant, requester, id string) (Job, error) {
	if s == nil || s.db == nil || tenant == "" || requester == "" ||
		len(tenant) > 128 || len(requester) > 128 || len(id) != 32 {
		return Job{}, ErrNotFound
	}
	var cancelled Job
	err := withImmediate(s.ctx, s.db, func(q sqliteSession) error {
		// Recheck current access, not just the identity that created the job.
		// Never reveal another tenant's or another requester's job IDs.
		if err := activeRequester(s.ctx, q, tenant, requester); err != nil {
			return ErrNotFound
		}
		var err error
		cancelled, err = scanJob(q.QueryRowContext(s.ctx,
			"SELECT id,tenant_id,kind,state,idempotency_key,created_at,updated_at,requested_by "+
				"FROM jobs WHERE tenant_id=? AND requested_by=? AND id=?",
			tenant, requester, id))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		switch cancelled.State {
		case Cancelled:
			return nil // repeat request: no duplicate audit event
		case Queued:
			// Only an unclaimed job may transition to cancelled.
		default:
			return ErrJobNotCancelable
		}

		now := s.now().UTC()
		result, err := q.ExecContext(s.ctx,
			"UPDATE jobs SET state='cancelled',available_at=NULL,updated_at=? "+
				"WHERE tenant_id=? AND requested_by=? AND id=? AND state='queued'",
			dbTime(now), tenant, requester, id)
		if err != nil {
			return err
		}
		if !oneRow(result) {
			return ErrJobNotCancelable
		}
		cancelled.State, cancelled.UpdatedAt = Cancelled, now
		// Fail closed and roll back the cancellation if audit insert fails.
		return appendAudit(s.ctx, q, cancelled, "job.cancelled", now)
	})
	return cancelled, err
}
