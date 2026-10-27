package jobs

import (
	"database/sql"
	"errors"
)

// GetFor denies cross-tenant and same-tenant cross-user reads. It rechecks
// active identity, membership and tenant status on the read itself. Use this
// method for a user's HTTP API; the older Get method is for internal tooling.
func (s *SQLiteStore) GetFor(tenant, requester, id string) (Job, error) {
	if s == nil || s.db == nil || tenant == "" || requester == "" || id == "" {
		return Job{}, ErrNotFound
	}
	j, err := scanJob(s.db.QueryRowContext(s.ctx, `SELECT j.id,j.tenant_id,j.kind,j.state,j.idempotency_key,j.created_at,j.updated_at,j.requested_by
		FROM jobs j
		JOIN tenants t ON t.id=j.tenant_id AND t.status='active'
		JOIN memberships m ON m.tenant_id=j.tenant_id AND m.identity_id=j.requested_by
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE j.tenant_id=? AND j.requested_by=? AND j.id=?`, tenant, requester, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return j, err
}

// ListFor shows only the active requester's own jobs. An owner/admin role is
// not automatically an exception: wider tenant views need a separate audited
// permission, never a client-chosen user filter.
func (s *SQLiteStore) ListFor(tenant, requester string) ([]Job, error) {
	if s == nil || s.db == nil || tenant == "" || requester == "" {
		return nil, ErrNotFound
	}
	if err := activeRequester(s.ctx, s.db, tenant, requester); err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.db.QueryContext(s.ctx, `SELECT j.id,j.tenant_id,j.kind,j.state,j.idempotency_key,j.created_at,j.updated_at,j.requested_by
		FROM jobs j
		JOIN tenants t ON t.id=j.tenant_id AND t.status='active'
		JOIN memberships m ON m.tenant_id=j.tenant_id AND m.identity_id=j.requested_by
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE j.tenant_id=? AND j.requested_by=?
		ORDER BY j.created_at,j.id`, tenant, requester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Job{}
	for rows.Next() {
		var item Job
		if _, err := scanJobInto(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// RoleFor rechecks the current database membership for a token-bound identity.
// The database role is authoritative over the caller's development-token role.
// Fail closed for suspended tenants, disabled identities, and deleted memberships.
func (s *SQLiteStore) RoleFor(tenant, requester string) (string, error) {
	if s == nil || s.db == nil || tenant == "" || requester == "" {
		return "", ErrNotFound
	}
	var role string
	err := s.db.QueryRowContext(s.ctx, `SELECT m.role FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=?`, tenant, requester).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}
