package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
)

// ErrSessionDenied deliberately conceals why a browser session no longer
// authorizes the requester's own-job read.
var ErrSessionDenied = errors.New("session is not authorized for job read")

var errSessionReadUnavailable = errors.New("job read session database unavailable")

const jobSessionReadAuthorization = `SELECT s.expires_at,s.idle_deadline_unix
	FROM sessions s
	JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
	JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
	JOIN identities i ON i.id=m.identity_id AND i.disabled=0
	WHERE s.token_sha256=? AND s.tenant_id=? AND s.identity_id=? AND s.revoked_at IS NULL
	AND m.role IN ('owner','admin','reseller','customer','auditor')`

const jobReadDetail = `SELECT id,tenant_id,kind,state,idempotency_key,created_at,updated_at,requested_by
	FROM jobs WHERE tenant_id=? AND requested_by=?`

// GetForSession rechecks browser-session and current membership state in the
// same statement snapshot as the owned job. A nil proof retains the existing
// trusted/developer-bearer read path; the proof is not authentication itself.
func (s *SQLiteStore) GetForSession(ctx context.Context, tenant, requester, id string, proof *auth.SessionReadProof) (Job, error) {
	if proof == nil {
		return s.GetForContext(ctx, tenant, requester, id)
	}
	if id == "" {
		return Job{}, ErrNotFound
	}
	items, err := s.readForSession(ctx, tenant, requester, proof, jobReadDetail+" AND id=?", "detail.id", tenant, requester, id)
	if err != nil {
		return Job{}, err
	}
	if len(items) == 0 {
		return Job{}, ErrNotFound
	}
	return items[0], nil
}

// ListForSession retains the requester's own-job scope and legacy list order.
func (s *SQLiteStore) ListForSession(ctx context.Context, tenant, requester string, proof *auth.SessionReadProof) ([]Job, error) {
	if proof == nil {
		return s.ListForContext(ctx, tenant, requester)
	}
	return s.readForSession(ctx, tenant, requester, proof, jobReadDetail, "detail.created_at,detail.id", tenant, requester)
}

// ListPageForSession applies the same bounded ID cursor as ListPageFor while
// rechecking the supplied browser session alongside the selected jobs.
func (s *SQLiteStore) ListPageForSession(ctx context.Context, tenant, requester, after string, limit int, proof *auth.SessionReadProof) ([]Job, string, error) {
	if proof == nil {
		return s.ListPageFor(ctx, tenant, requester, after, limit)
	}
	if ctx == nil || !validPage(after, limit) {
		return nil, "", ErrInvalidPage
	}
	items, err := s.readForSession(ctx, tenant, requester, proof,
		jobReadDetail+" AND id>? COLLATE BINARY ORDER BY id COLLATE BINARY LIMIT ?", "detail.id COLLATE BINARY",
		tenant, requester, after, limit+1)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].ID
	}
	return items, next, nil
}

// detailSQL and orderSQL are server-authored fragments. The left join retains
// authorization even for an empty list/page or a missing job; an empty result
// must not turn a withdrawn session into a successful read. Close the statement
// before the expiry helper takes a writer reservation on the same pool.
func (s *SQLiteStore) readForSession(ctx context.Context, tenant, requester string, proof *auth.SessionReadProof, detailSQL, orderSQL string, detailArgs ...any) ([]Job, error) {
	if ctx == nil || s == nil || s.db == nil || !proof.MatchesDatabase(s.db) {
		return nil, errSessionReadUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if tenant == "" || requester == "" {
		return nil, ErrSessionDenied
	}
	digest := proof.Digest()
	args := append([]any{digest[:], tenant, requester}, detailArgs...)
	rows, err := s.db.QueryContext(ctx, "WITH authorized AS ("+jobSessionReadAuthorization+"), detail AS ("+detailSQL+
		") SELECT authorized.expires_at,authorized.idle_deadline_unix,detail.* FROM authorized LEFT JOIN detail ON 1=1 ORDER BY "+orderSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("read session-authorized jobs: %w", err)
	}
	defer rows.Close()
	items := []Job{}
	var seen bool
	var expiry sql.NullString
	var idle int64
	for rows.Next() {
		var id, tenantID, kind, state, key, created, updated, actor sql.NullString
		if err := rows.Scan(&expiry, &idle, &id, &tenantID, &kind, &state, &key, &created, &updated, &actor); err != nil {
			return nil, fmt.Errorf("scan session-authorized job: %w", err)
		}
		seen = true
		if !id.Valid {
			continue
		}
		job := Job{ID: id.String, TenantID: tenantID.String, Kind: kind.String, State: state.String, IdempotencyKey: key.String, RequestedBy: actor.String}
		job.CreatedAt, err = time.Parse(time.RFC3339Nano, created.String)
		if err != nil {
			return nil, fmt.Errorf("invalid job created_at: %w", err)
		}
		job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated.String)
		if err != nil {
			return nil, fmt.Errorf("invalid job updated_at: %w", err)
		}
		items = append(items, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session-authorized jobs: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close session-authorized jobs: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !seen || !expiry.Valid {
		return nil, ErrSessionDenied
	}
	expires, err := time.Parse(time.RFC3339Nano, expiry.String)
	if err != nil {
		return nil, ErrSessionDenied
	}
	now := proof.Now()
	absolute := !now.Before(expires)
	if absolute || idle <= now.Unix() {
		if err := proof.RevokeObservedExpiry(ctx, expiry.String, now, absolute); err != nil {
			return nil, fmt.Errorf("record job read session expiry: %w", err)
		}
		return nil, ErrSessionDenied
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
