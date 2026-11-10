package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

var ErrPasswordChangeUnavailable = errors.New("password change unavailable")

// ChangePasswordTrusted reauthenticates the identity with its password and
// active TOTP seed, replaces its verifier and revokes all its sessions across
// tenants. It issues no session. Only reviewed trusted server-side callers may
// invoke it; there is no public route, reset or administrative-user operation.
func (s *InternalLoginService) ChangePasswordTrusted(ctx context.Context, tenant, username, peerIP string, currentPassword, newPassword []byte, code string) (resultErr error) {
	if ctx == nil || s == nil || s.db == nil || s.box == nil || s.limiter == nil || s.budget == nil {
		return ErrLoginDenied
	}
	// Operational diagnostics must not expose database values, paths or secret
	// material. Cancellation keeps its standard identity for trusted callers.
	defer func() {
		if resultErr == nil {
			return
		}
		if ctx.Err() != nil {
			resultErr = ctx.Err()
			return
		}
		if !errors.Is(resultErr, ErrLoginDenied) && !errors.Is(resultErr, context.Canceled) && !errors.Is(resultErr, context.DeadlineExceeded) {
			resultErr = ErrPasswordChangeUnavailable
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	allowed, err := s.limiter.Admit(ctx, tenant, username, peerIP)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrLoginDenied
	}
	if len(newPassword) < 12 || len(newPassword) > maxPasswordBytes {
		return ErrLoginDenied
	}
	var identity, role, storedPHC string
	err = s.db.QueryRowContext(ctx, `SELECT i.id,m.role,i.password_phc FROM identities i
		JOIN memberships m ON m.identity_id=i.id
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE i.username=? AND m.tenant_id=? AND i.disabled=0 AND i.password_phc IS NOT NULL`, username, tenant).
		Scan(&identity, &role, &storedPHC)
	exists := err == nil && validRole(rbac.Role(role))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	verifyPHC := storedPHC
	if !exists {
		verifyPHC = s.dummyPHC
	} else if _, valid := parseArgon2PHC(storedPHC); !valid {
		verifyPHC = s.dummyPHC
		exists = false
	}
	verified, err := s.budget.Verify(currentPassword, verifyPHC)
	if err != nil {
		return ErrLoginDenied
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !verified || !exists {
		return ErrLoginDenied
	}
	var envelope string
	err = s.db.QueryRowContext(ctx, "SELECT secret_envelope FROM mfa_totp_enrollments WHERE identity_id=? AND status='active'", identity).Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoginDenied
	}
	if err != nil {
		return err
	}
	secret, err := s.box.Open(identity, envelope)
	if err != nil {
		return ErrLoginDenied
	}
	defer clear(secret)
	if _, valid := acceptedTOTPStep(secret, code, s.now().UTC()); !valid {
		return ErrLoginDenied
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Both the old-password verifier and new-password derivation run outside
	// the writer transaction under the same process-wide admission slots.
	newPHC, err := s.budget.Hash(newPassword)
	if err != nil {
		if errors.Is(err, ErrPasswordVerifierBusy) {
			return ErrLoginDenied
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, valid := acceptedTOTPStep(secret, code, s.now().UTC()); !valid {
		return ErrLoginDenied
	}
	auditID, err := bootstrapID()
	if err != nil {
		return err
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var currentRole string
	err = conn.QueryRowContext(ctx, `SELECT m.role FROM identities i
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active'
		WHERE i.id=? AND i.username=? AND i.password_phc=? AND i.disabled=0 AND e.secret_envelope=?`,
		tenant, identity, username, storedPHC, envelope).Scan(&currentRole)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoginDenied
	}
	if err != nil {
		return err
	}
	if currentRole != role {
		return ErrLoginDenied
	}
	// Recheck at writer-transaction time; password hashing and lock acquisition
	// can outlast the TOTP's accepted window.
	now := s.now().UTC()
	step, valid := acceptedTOTPStep(secret, code, now)
	if !valid {
		return ErrLoginDenied
	}
	r, err := conn.ExecContext(ctx, `INSERT INTO totp_replay_state(identity_id,last_accepted_step) VALUES(?,?)
		ON CONFLICT(identity_id) DO UPDATE SET last_accepted_step=excluded.last_accepted_step,
		updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE excluded.last_accepted_step>totp_replay_state.last_accepted_step`, identity, step)
	if err != nil {
		return err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrLoginDenied
	}
	r, err = conn.ExecContext(ctx, "UPDATE identities SET password_phc=? WHERE id=? AND password_phc=? AND disabled=0", newPHC, identity, storedPHC)
	if err != nil {
		return err
	}
	count, err = r.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrLoginDenied
	}
	if _, err := conn.ExecContext(ctx, "UPDATE sessions SET revoked_at=? WHERE identity_id=? AND revoked_at IS NULL", now.Format(time.RFC3339Nano), identity); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id)
		VALUES(?,?,?,'user','auth.password_changed',?,?)`, auditID, tenant, identity, identity, auditID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}
