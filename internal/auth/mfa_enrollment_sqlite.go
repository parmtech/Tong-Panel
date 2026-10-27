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
	db *sql.DB
	box *MFASecretBox
	now func() time.Time
}

func NewMFAEnrollmentStore(db *sql.DB, box *MFASecretBox) (*MFAEnrollmentStore, error) {
	if db == nil || box == nil || box.aead == nil {
		return nil, ErrMFAEnrollmentUnavailable
	}
	return &MFAEnrollmentStore{db: db, box: box, now: time.Now}, nil
}

func (s *MFAEnrollmentStore) BeginTrusted(ctx context.Context, tenant, identity string) ([]byte, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return nil, ErrMFAEnrollmentUnavailable
	}
	secret := make([]byte, 20) // RFC 6238 baseline: at least 160 unpredictable bits
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	envelope, err := s.box.Seal(identity, secret)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	expiry := now.Add(10 * time.Minute).Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `INSERT INTO mfa_totp_enrollments(identity_id,secret_envelope,status,pending_expires_at)
		SELECT i.id,?,'pending',? FROM identities i
		JOIN memberships m ON m.identity_id=i.id
		JOIN tenants t ON t.id=m.tenant_id
		WHERE i.id=? AND m.tenant_id=? AND i.disabled=0 AND t.status='active'
		ON CONFLICT(identity_id) DO NOTHING`, envelope, expiry, identity, tenant)
	if err != nil { return nil, err }
	n, err := res.RowsAffected()
	if err != nil { return nil, err }
	if n != 1 { return nil, ErrMFAEnrollmentUnavailable }
	// Only trusted callers may show this once; the DB stores encrypted bytes.
	return secret, nil
}

func (s *MFAEnrollmentStore) ConfirmTrusted(ctx context.Context, tenant, identity, code string) (bool, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return false, ErrMFAEnrollmentUnavailable
	}
	now := s.now().UTC()
	var envelope string
	err := s.db.QueryRowContext(ctx, `SELECT secret_envelope FROM mfa_totp_enrollments
		WHERE identity_id=? AND status='pending' AND julianday(pending_expires_at)>julianday(?)`,
		identity, now.Format(time.RFC3339Nano)).Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) { return false, nil }
	if err != nil { return false, err }
	secret, err := s.box.Open(identity, envelope)
	if err != nil { return false, err } // tampered ciphertext must fail closed
	defer clear(secret)
	step, ok := acceptedTOTPStep(secret, code, now)
	if !ok { return false, nil }
	// One transaction couples replay consumption to enrollment activation.
	// If the identity/tenant is revoked or another client won, both roll back.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return false, err }
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, `INSERT INTO totp_replay_state(identity_id,last_accepted_step)
		SELECT e.identity_id,? FROM mfa_totp_enrollments e
		JOIN identities i ON i.id=e.identity_id AND i.disabled=0
		JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=?
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE e.identity_id=? AND e.status='pending'
			AND julianday(e.pending_expires_at)>julianday(?)
		ON CONFLICT(identity_id) DO UPDATE SET
			last_accepted_step=excluded.last_accepted_step,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE excluded.last_accepted_step>totp_replay_state.last_accepted_step`,
		step, tenant, identity, now.Format(time.RFC3339Nano))
	if err != nil { return false, err }
	n, err := r.RowsAffected()
	if err != nil { return false, err }
	if n != 1 { return false, nil }
	r, err = tx.ExecContext(ctx, `UPDATE mfa_totp_enrollments
		SET status='active',activated_at=?
		WHERE identity_id=? AND status='pending'
			AND julianday(pending_expires_at)>julianday(?)`,
		now.Format(time.RFC3339Nano), identity, now.Format(time.RFC3339Nano))
	if err != nil { return false, err }
	n, err = r.RowsAffected()
	if err != nil { return false, err }
	if n != 1 { return false, nil }
	if err := tx.Commit(); err != nil { return false, err }
	return true, nil
}

// VerifyActive checks only an already-confirmed encrypted seed, then consumes
// the step in the identity-wide replay ledger. It does NOT create a session.
func (s *MFAEnrollmentStore) VerifyActive(ctx context.Context, tenant, identity, code string) (bool, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return false, ErrMFAEnrollmentUnavailable
	}
	var envelope string
	err := s.db.QueryRowContext(ctx, `SELECT secret_envelope FROM mfa_totp_enrollments WHERE identity_id=? AND status='active'`,
		identity).Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) { return false, nil }
	if err != nil { return false, err }
	secret, err := s.box.Open(identity, envelope)
	if err != nil { return false, err }
	defer clear(secret)
	guard, err := NewTOTPReplayGuard(s.db)
	if err != nil { return false, err }
	return guard.VerifyAndConsume(ctx, tenant, identity, secret, code, s.now().UTC())
}
