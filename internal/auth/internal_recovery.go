package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

// InternalRecoveryService admits password-authenticated recovery-code use for
// trusted server-side callers. It does not create sessions, reset MFA, or
// expose an HTTP route. Public recovery still requires a separately reviewed
// caller, proxy/source policy, key custody and recovery ceremony.
type InternalRecoveryService struct {
	db       *sql.DB
	recovery *SQLiteRecoveryStore
	limiter  *SQLiteLoginLimiter
	budget   *PasswordVerifyBudget
	dummyPHC string
	now      func() time.Time
}

// NewInternalRecoveryService requires all components to share the same SQLite
// handle so limiter, credential and recovery state cannot diverge.
func NewInternalRecoveryService(db *sql.DB, recovery *SQLiteRecoveryStore, limiter *SQLiteLoginLimiter, budget *PasswordVerifyBudget) (*InternalRecoveryService, error) {
	if db == nil || recovery == nil || recovery.db != db ||
		limiter == nil || limiter.db != db || budget == nil ||
		budget.slots == nil || budget.verify == nil {
		return nil, ErrRecoveryDenied
	}
	dummy, err := HashPassword([]byte("TongPanel internal nonexistent-account dummy password"))
	if err != nil {
		return nil, err
	}
	return &InternalRecoveryService{
		db: db, recovery: recovery, limiter: limiter, budget: budget,
		dummyPHC: dummy, now: time.Now,
	}, nil
}

// RecoverTrusted checks a password before admitting one recovery code. Source
// must be the trusted socket peer, and tenant must come from server-side
// context. Admission and Argon2 happen before BEGIN IMMEDIATE. The write
// transaction rechecks the exact password PHC, role and MFA enrollment snapshot
// before consuming the code, revoking all sessions and writing the audit.
// Never expose this method directly to an HTTP handler or developer-token API.
func (s *InternalRecoveryService) RecoverTrusted(ctx context.Context, tenant, username, trustedPeerIP string, password []byte, code string) error {
	if s == nil || s.db == nil || s.recovery == nil || s.limiter == nil || s.budget == nil {
		return ErrRecoveryDenied
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	allowed, err := s.limiter.Admit(ctx, tenant, username, trustedPeerIP)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if !allowed {
		return ErrRecoveryDenied
	}

	var identity, role, tenantStatus string
	var storedPHC, enrollment sql.NullString
	var disabled int
	err = s.db.QueryRowContext(ctx, `SELECT i.id,m.role,i.password_phc,i.disabled,t.status,e.secret_envelope
		FROM identities i
		JOIN memberships m ON m.identity_id=i.id
		JOIN tenants t ON t.id=m.tenant_id
		LEFT JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active'
		WHERE i.username=? AND m.tenant_id=?`, username, tenant).
		Scan(&identity, &role, &storedPHC, &disabled, &tenantStatus, &enrollment)
	exists := err == nil && disabled == 0 && tenantStatus == "active" && validRole(rbac.Role(role)) && storedPHC.Valid
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	verifyPHC := storedPHC.String
	if !exists {
		verifyPHC = s.dummyPHC
	} else if _, valid := parseArgon2PHC(storedPHC.String); !valid {
		// Corrupt stored hashes still perform a bounded dummy verification.
		verifyPHC = s.dummyPHC
		exists = false
	}
	verified, err := s.budget.Verify(password, verifyPHC)
	if err != nil || !verified || !exists || !enrollment.Valid {
		return ErrRecoveryDenied
	}

	accepted := false
	err = s.recovery.transaction(ctx, func(conn *sql.Conn) error {
		var currentRole string
		var currentPHC, currentEnrollment string
		err := conn.QueryRowContext(ctx, `SELECT m.role,i.password_phc,e.secret_envelope
			FROM identities i
			JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
			JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
			JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active'
			WHERE i.id=? AND i.username=? AND i.disabled=0 AND i.password_phc=?`,
			tenant, identity, username, storedPHC.String).
			Scan(&currentRole, &currentPHC, &currentEnrollment)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecoveryDenied
		}
		if err != nil {
			return err
		}
		if currentRole != role || currentPHC != storedPHC.String || currentEnrollment != enrollment.String {
			return ErrRecoveryDenied
		}
		accepted, err = s.recovery.consumeInTransaction(ctx, conn, tenant, identity, code, currentEnrollment, s.now().UTC())
		return err
	})
	if err != nil {
		return err
	}
	if !accepted {
		return ErrRecoveryDenied
	}
	return nil
}
