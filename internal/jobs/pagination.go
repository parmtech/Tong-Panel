package jobs

import (
	"context"
	"errors"
	"sort"
)

var ErrInvalidPage = errors.New("invalid jobs page")

func validPage(after string, limit int) bool {
	if limit < 1 || limit > 100 {
		return false
	}
	if after == "" {
		return true
	}
	if len(after) != 32 {
		return false
	}
	for _, char := range after {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

// ListPage retains the journal's tenant-wide read scope. Its cursor is an ID
// boundary, never authority. Concurrent inserts before it need a fresh listing.
func (s *Store) ListPage(ctx context.Context, tenant, after string, limit int) ([]Job, string, error) {
	if ctx == nil || !validPage(after, limit) {
		return nil, "", ErrInvalidPage
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if s == nil || tenant == "" {
		return nil, "", ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	items := make([]Job, 0, limit+1)
	for _, job := range s.items {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if job.TenantID != tenant || job.ID <= after {
			continue
		}
		// Retain only the smallest limit+1 IDs in a sorted bounded buffer.
		// The journal still requires a full scan of its already resident rows.
		index := sort.Search(len(items), func(i int) bool { return items[i].ID >= job.ID })
		if index == len(items) && len(items) == limit+1 {
			continue
		}
		if len(items) < limit+1 {
			items = append(items, Job{})
		}
		copy(items[index+1:], items[index:len(items)-1])
		items[index] = job
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].ID
	}
	return items, next, nil
}

// ListPageFor uses the request context, without changing the store context
// shared by other requests or the development worker. SQLite reads only the
// active requester's own jobs; auditors retain their existing read allowance.
func (s *SQLiteStore) ListPageFor(ctx context.Context, tenant, requester, after string, limit int) ([]Job, string, error) {
	if ctx == nil || !validPage(after, limit) {
		return nil, "", ErrInvalidPage
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if s == nil || s.db == nil || tenant == "" || requester == "" {
		return nil, "", ErrNotFound
	}
	if err := activeRequester(ctx, s.db, tenant, requester); err != nil {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		if errors.Is(err, errInactiveRequester) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT j.id,j.tenant_id,j.kind,j.state,j.idempotency_key,j.created_at,j.updated_at,j.requested_by
		FROM jobs j
		JOIN tenants t ON t.id=j.tenant_id AND t.status='active'
		JOIN memberships m ON m.tenant_id=j.tenant_id AND m.identity_id=j.requested_by
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE j.tenant_id=? AND j.requested_by=? AND j.id>? COLLATE BINARY
		ORDER BY j.id COLLATE BINARY LIMIT ?`, tenant, requester, after, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := []Job{}
	for rows.Next() {
		var job Job
		if _, err := scanJobInto(rows, &job); err != nil {
			return nil, "", err
		}
		items = append(items, job)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].ID
	}
	return items, next, nil
}
