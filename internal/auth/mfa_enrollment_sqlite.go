package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"
)

// ErrMFAEnrollmentUnavailable intentionally does not distinguish already
// enrolled users from disabled identities or missing tenant membership.
var ErrMFAEnrollmentUnavailable = errors.New("MFA enrollment unavailable")

// MFAEnrollmentStore is an internal, trusted-caller-only prototype.
// NEVER connect BeginTrusted to an unauthenticated or developer-token-only
// HTTP route. An approved password/re-auth/bootstrap proof is required first.
type MFAEnrollmentStore struct {
	db  *sql.DB
	box *MFASecretBox
	now func() time.Time
}

func NewMFAEnrollmentStore(db *sql.DB, box *MFASecretBox) (*MFAEnrollmentStore, error) {
	if db == nil || box == nil || box.aead == nil {
		return nil, ErrMFAEnrollmentUnavailable
	}
	return &MFAEnrollmentStore{db: db, box: box, now: time.Now}, nil
}

// BeginTrusted starts enrollment or replaces only an expired pending attempt.
// A live pending or active enrollment must never be overwritten by this flow.
func (s *MFAEnrollmentStore) BeginTrusted(ctx context.Context, tenant, identity string) ([]byte, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return nil, ErrMFAEnrollmentUnavailable
	}
	secret := make([]byte, 20) // RFC 6238 baseline: at least 160 unpredictable bits
	delivered := false
	defer func() {
		if !delivered {
			clear(secret)
		}
	}()
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	envelope, err := s.box.Seal(identity, secret)
	if err != nil {
		return nil, err
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	// Start the pending-enrollment lifetime only after obtaining the writer
	// reservation, so a connection or SQLite wait cannot consume its window.
	now := s.now().UTC()
	expiry := now.Add(10 * time.Minute).Format(time.RFC3339Nano)
	res, err := conn.ExecContext(ctx, `INSERT INTO mfa_totp_enrollments(identity_id,secret_envelope,status,pending_expires_at)
		SELECT i.id,?,'pending',? FROM identities i
		JOIN memberships m ON m.identity_id=i.id
		JOIN tenants t ON t.id=m.tenant_id
		WHERE i.id=? AND m.tenant_id=? AND i.disabled=0 AND t.status='active'
		ON CONFLICT(identity_id) DO UPDATE SET
			secret_envelope=excluded.secret_envelope,
			pending_expires_at=excluded.pending_expires_at
		WHERE mfa_totp_enrollments.status='pending'
			AND julianday(mfa_totp_enrollments.pending_expires_at)<=julianday(?)`,
		envelope, expiry, identity, tenant, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, ErrMFAEnrollmentUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return nil, err
	}
	committed = true
	// Only trusted callers may show this once; the DB stores encrypted bytes.
	delivered = true
	return secret, nil
}

func (s *MFAEnrollmentStore) ConfirmTrusted(ctx context.Context, tenant, identity, code string) (bool, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return false, ErrMFAEnrollmentUnavailable
	}
	preflightNow := s.now().UTC()
	var envelope, pendingExpiry string
	err := s.db.QueryRowContext(ctx, `SELECT secret_envelope,pending_expires_at FROM mfa_totp_enrollments
		WHERE identity_id=? AND status='pending' AND julianday(pending_expires_at)>julianday(?)`,
		identity, preflightNow.Format(time.RFC3339Nano)).Scan(&envelope, &pendingExpiry)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	secret, err := s.box.Open(identity, envelope)
	if err != nil {
		return false, err
	} // tampered ciphertext must fail closed
	defer clear(secret)
	if _, ok := acceptedTOTPStep(secret, code, preflightNow); !ok {
		return false, nil
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	// Pending expiry and the TOTP window must be checked after acquiring the
	// writer reservation; otherwise this confirmation can age out while waiting.
	now := s.now().UTC()
	var currentEnvelope, currentExpiry string
	err = conn.QueryRowContext(ctx, `SELECT secret_envelope,pending_expires_at FROM mfa_totp_enrollments
		WHERE identity_id=? AND status='pending' AND julianday(pending_expires_at)>julianday(?)`,
		identity, now.Format(time.RFC3339Nano)).Scan(&currentEnvelope, &currentExpiry)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if currentEnvelope != envelope || currentExpiry != pendingExpiry {
		return false, nil
	}
	step, ok := acceptedTOTPStep(secret, code, now)
	if !ok {
		return false, nil
	}
	// The same writer transaction couples replay consumption to activation.
	// If authority or enrollment state changed, both operations roll back.
	r, err := conn.ExecContext(ctx, `INSERT INTO totp_replay_state(identity_id,last_accepted_step)
		SELECT e.identity_id,? FROM mfa_totp_enrollments e
		JOIN identities i ON i.id=e.identity_id AND i.disabled=0
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE e.identity_id=? AND e.status='pending'
			AND e.secret_envelope=? AND e.pending_expires_at=?
			AND julianday(e.pending_expires_at)>julianday(?)
		ON CONFLICT(identity_id) DO UPDATE SET
			last_accepted_step=excluded.last_accepted_step,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE excluded.last_accepted_step>totp_replay_state.last_accepted_step`,
		step, tenant, identity, envelope, pendingExpiry, now.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	r, err = conn.ExecContext(ctx, `UPDATE mfa_totp_enrollments
		SET status='active',activated_at=?
		WHERE identity_id=? AND status='pending'
			AND secret_envelope=? AND pending_expires_at=?
			AND julianday(pending_expires_at)>julianday(?)`,
		now.Format(time.RFC3339Nano), identity, envelope, pendingExpiry, now.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	n, err = r.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return false, err
	}
	committed = true
	return true, nil
}

// VerifyActive checks only an already-confirmed encrypted seed, then consumes
// the step in the identity-wide replay ledger. It does NOT create a session.
func (s *MFAEnrollmentStore) VerifyActive(ctx context.Context, tenant, identity, code string) (bool, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return false, ErrMFAEnrollmentUnavailable
	}
	preflightNow := s.now().UTC()
	var envelope string
	err := s.db.QueryRowContext(ctx, `SELECT e.secret_envelope FROM mfa_totp_enrollments e
		JOIN identities i ON i.id=e.identity_id AND i.disabled=0
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE e.identity_id=? AND e.status='active'`, tenant, identity).Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	secret, err := s.box.Open(identity, envelope)
	if err != nil {
		return false, err
	}
	defer clear(secret)
	if _, ok := acceptedTOTPStep(secret, code, preflightNow); !ok {
		return false, nil
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	now := s.now().UTC()
	var currentEnvelope string
	err = conn.QueryRowContext(ctx, `SELECT e.secret_envelope FROM mfa_totp_enrollments e
		JOIN identities i ON i.id=e.identity_id AND i.disabled=0
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE e.identity_id=? AND e.status='active'`, tenant, identity).Scan(&currentEnvelope)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if currentEnvelope != envelope {
		return false, nil
	}
	step, ok := acceptedTOTPStep(secret, code, now)
	if !ok {
		return false, nil
	}
	r, err := conn.ExecContext(ctx, `INSERT INTO totp_replay_state(identity_id,last_accepted_step)
		SELECT i.id,? FROM identities i
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active' AND e.secret_envelope=?
		WHERE i.id=? AND i.disabled=0
		ON CONFLICT(identity_id) DO UPDATE SET
			last_accepted_step=excluded.last_accepted_step,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE excluded.last_accepted_step > totp_replay_state.last_accepted_step`,
		step, tenant, envelope, identity)
	if err != nil {
		return false, err
	}
	changed, err := r.RowsAffected()
	if err != nil || changed != 1 {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return false, err
	}
	committed = true
	return true, nil
}
