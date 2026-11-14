package jobs

import (
	"context"
	"database/sql"
	"errors"

	"github.com/parmtech/tong-panel/internal/rbac"
)

// jobExecutionRolePredicate is shared by claim selection and every lease
// transition. Keep it in parity with the jobs:create RBAC policy. It contains
// only server-authored SQL; no role or action supplied by a caller is spliced
// into a query.
const jobExecutionRolePredicate = "m.role IN ('owner','admin','reseller','customer')"

// ErrRequesterDenied means current database state does not grant the requested
// job mutation. It deliberately does not distinguish membership, account,
// tenant or role failures.
var ErrRequesterDenied = errors.New("requester is not authorized for job mutation")

// authorizedRequester rechecks current mutation authority in the caller's
// transaction. activeRequester remains a membership-only check for own-job
// reads, which auditors are still permitted to perform.
func authorizedRequester(ctx context.Context, q sqliteSession, tenant, requester, action string) error {
	var role string
	err := q.QueryRowContext(ctx, `SELECT m.role FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=?`, tenant, requester).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRequesterDenied
	}
	if err != nil {
		return err
	}
	principal := rbac.Principal{TenantID: tenant, UserID: requester, Role: rbac.Role(role)}
	if !principal.Can(action, tenant) {
		return ErrRequesterDenied
	}
	return nil
}
