package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var ErrRecoveryDenied = errors.New("MFA recovery unavailable")

const recoveryCodeCount = 8

// SQLiteRecoveryStore is a trusted-caller-only recovery primitive. Callers must
// verify recent password/MFA authentication before issuing replacement codes,
// and password plus source admission before consumption. It exposes no HTTP
// route, resets no MFA enrollment, and issues no authenticated session. A
// successful consumption is only proof for a separately reviewed recovery flow.
// The independent 32-byte pepper must remain outside DB/backups; losing it
// requires reissuing codes through a trusted authenticated procedure.
type SQLiteRecoveryStore struct {
	db  *sql.DB
	key []byte
	now func() time.Time
}

func NewSQLiteRecoveryStore(db *sql.DB, pepper []byte) (*SQLiteRecoveryStore, error) {
	if db == nil || len(pepper) != 32 {
		return nil, ErrRecoveryDenied
	}
	return &SQLiteRecoveryStore{db: db, key: append([]byte(nil), pepper...), now: time.Now}, nil
}

func (s *SQLiteRecoveryStore) digest(tenant, identity, code string) []byte {
	h := hmac.New(sha256.New, s.key)
	_, _ = h.Write([]byte("tong-mfa-recovery-v1\x00" + tenant + "\x00" + identity + "\x00" + code))
	return h.Sum(nil)
}

// IssueTrusted replaces the entire prior set atomically and returns plaintext
// codes exactly once. Codes contain 256 random bits. Neither codes nor hashes
// belong in logs. Issuance requires active membership, tenant, identity and MFA.
func (s *SQLiteRecoveryStore) IssueTrusted(ctx context.Context, tenant, identity string) ([]string, error) {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return nil, ErrRecoveryDenied
	}
	codes := make([]string, recoveryCodeCount)
	for i := range codes {
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return nil, err
		}
		codes[i] = hex.EncodeToString(raw[:])
		clear(raw[:])
	}
	err := s.transaction(ctx, func(conn *sql.Conn) error {
		enrollment, err := recoveryEnrollment(ctx, conn, tenant, identity)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "DELETE FROM mfa_recovery_sets WHERE tenant_id=? AND identity_id=?", tenant, identity); err != nil {
			return err
		}
		hash := sha256.Sum256([]byte(enrollment))
		if _, err := conn.ExecContext(ctx, "INSERT INTO mfa_recovery_sets(tenant_id,identity_id,enrollment_sha256,window_started_unix) VALUES(?,?,?,?)", tenant, identity, hash[:], s.now().UTC().Unix()); err != nil {
			return err
		}
		for _, code := range codes {
			if _, err := conn.ExecContext(ctx, "INSERT INTO mfa_recovery_codes(tenant_id,identity_id,code_sha256) VALUES(?,?,?)", tenant, identity, s.digest(tenant, identity, code)); err != nil {
				return err
			}
		}
		return recoveryAudit(ctx, conn, tenant, identity, "auth.recovery_codes_issued")
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// ConsumeTrusted consumes one code and revokes ALL sessions for the identity,
// across tenants, in the same transaction as the audit. It does not bypass
// password verification or create a new session. Five guesses per set/account
// per 15 minutes are admitted durably. Clock rollback fails closed. Enrollment
// changes invalidate the complete set even when MFA remains active.
func (s *SQLiteRecoveryStore) ConsumeTrusted(ctx context.Context, tenant, identity, code string) error {
	if s == nil || !validMFAIdentity(tenant) || !validMFAIdentity(identity) {
		return ErrRecoveryDenied
	}
	accepted := false
	err := s.transaction(ctx, func(conn *sql.Conn) error {
		enrollment, err := recoveryEnrollment(ctx, conn, tenant, identity)
		if err != nil {
			return err
		}
		accepted, err = s.consumeInTransaction(ctx, conn, tenant, identity, code, enrollment, s.now().UTC())
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

// consumeInTransaction consumes one recovery code using a connection already
// protected by the caller's SQLite writer transaction. It deliberately does
// not begin or commit a transaction so credential rechecks, consumption,
// session revocation and audit can be made atomic by a trusted coordinator.
func (s *SQLiteRecoveryStore) consumeInTransaction(ctx context.Context, conn *sql.Conn, tenant, identity, code, enrollment string, nowTime time.Time) (bool, error) {
	hash := sha256.Sum256([]byte(enrollment))
	var start int64
	var attempts int
	err := conn.QueryRowContext(ctx, "SELECT window_started_unix,attempts FROM mfa_recovery_sets WHERE tenant_id=? AND identity_id=? AND enrollment_sha256=?", tenant, identity, hash[:]).Scan(&start, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrRecoveryDenied
	}
	if err != nil {
		return false, err
	}
	now := nowTime.UTC().Unix()
	if now < start {
		return false, ErrRecoveryDenied
	}
	if now-start >= 15*60 {
		start, attempts = now, 0
	}
	if attempts >= 5 {
		return false, ErrRecoveryDenied
	}
	if _, err := conn.ExecContext(ctx, "UPDATE mfa_recovery_sets SET window_started_unix=?,attempts=? WHERE tenant_id=? AND identity_id=?", start, attempts+1, tenant, identity); err != nil {
		return false, err
	}
	// Reject noncanonical inputs while still charging the admission budget.
	var raw []byte
	decodeErr := ErrRecoveryDenied
	if len(code) == 64 {
		raw, decodeErr = hex.DecodeString(code)
	}
	accepted := false
	if decodeErr == nil && len(raw) == 32 && hex.EncodeToString(raw) == code {
		result, err := conn.ExecContext(ctx, "DELETE FROM mfa_recovery_codes WHERE tenant_id=? AND identity_id=? AND code_sha256=?", tenant, identity, s.digest(tenant, identity, code))
		if err != nil {
			clear(raw)
			return false, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			clear(raw)
			return false, err
		}
		accepted = n == 1
	}
	clear(raw)
	if !accepted {
		return false, recoveryAudit(ctx, conn, tenant, identity, "auth.recovery_denied")
	}
	if _, err := conn.ExecContext(ctx, "UPDATE sessions SET revoked_at=? WHERE identity_id=? AND revoked_at IS NULL", nowTime.UTC().Format(time.RFC3339Nano), identity); err != nil {
		return false, err
	}
	return true, recoveryAudit(ctx, conn, tenant, identity, "auth.recovery_consumed")
}

func recoveryEnrollment(ctx context.Context, conn *sql.Conn, tenant, identity string) (string, error) {
	var envelope string
	err := conn.QueryRowContext(ctx, `SELECT e.secret_envelope FROM memberships m
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active'
		WHERE m.tenant_id=? AND m.identity_id=?`, tenant, identity).Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRecoveryDenied
	}
	return envelope, err
}

func recoveryAudit(ctx context.Context, conn *sql.Conn, tenant, identity, action string) error {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return err
	}
	id := hex.EncodeToString(raw[:])
	if action == "auth.recovery_denied" {
		// An unsuccessful attempt must not claim that the account owner acted.
		_, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_kind,actor_ref,action,subject_id,request_id)
			VALUES(?,?,'system','auth.recovery',?,?,?)`, id, tenant, action, identity, id)
		return err
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id)
		VALUES(?,?,?,'user',?,?,?)`, id, tenant, identity, action, identity, id)
	return err
}

func (s *SQLiteRecoveryStore) transaction(ctx context.Context, fn func(*sql.Conn) error) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	if err := fn(conn); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}
