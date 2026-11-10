package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

// SQLiteSessionStore persists revocable token hashes with absolute and idle
// expiry. It does not expose HTTP login, cookie issuing, or password verification.
const sqliteSessionIdle = 30 * time.Minute
const sqliteSessionLookupTimeout = 5 * time.Second

type SQLiteSessionStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLiteSessionStore(db *sql.DB) (*SQLiteSessionStore, error) {
	if db == nil {
		return nil, errors.New("missing SQLite database")
	}
	return &SQLiteSessionStore{db: db, now: time.Now}, nil
}

// Issue makes independent 256-bit bearer and CSRF secrets. Only their hashes
// are stored; the identity must be an active member of the requested tenant.
func (s *SQLiteSessionStore) Issue(p rbac.Principal, lifetime time.Duration) (token, csrf string, expires time.Time, err error) {
	if s == nil || p.UserID == "" || p.TenantID == "" || !validRole(p.Role) || lifetime <= 0 || lifetime > MaxSessionLifetime {
		return "", "", time.Time{}, ErrInvalidSession
	}
	var tokenRaw, csrfRaw, idRaw [32]byte
	if _, err = rand.Read(tokenRaw[:]); err != nil {
		return "", "", time.Time{}, err
	}
	if _, err = rand.Read(csrfRaw[:]); err != nil {
		return "", "", time.Time{}, err
	}
	if _, err = rand.Read(idRaw[:16]); err != nil {
		return "", "", time.Time{}, err
	}
	tokenHash := sha256.Sum256(tokenRaw[:])
	csrfHash := sha256.Sum256(csrfRaw[:])
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		return "", "", time.Time{}, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		return "", "", time.Time{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	// Compute the lifetime only after the writer reservation, so connection
	// and lock wait cannot consume the requested session lifetime.
	now := s.now().UTC()
	expires = now.Add(lifetime)
	idleUntil := now.Add(sqliteSessionIdle)
	if expires.Before(idleUntil) {
		idleUntil = expires
	}
	var role string
	err = conn.QueryRowContext(context.Background(), `SELECT m.role FROM memberships m
		JOIN identities i ON i.id=m.identity_id
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE m.tenant_id=? AND m.identity_id=? AND i.disabled=0`, p.TenantID, p.UserID).Scan(&role)
	if err != nil || rbac.Role(role) != p.Role {
		return "", "", time.Time{}, ErrInvalidSession
	}
	_, err = conn.ExecContext(context.Background(), `INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at,idle_deadline_unix)
		VALUES(?,?,?,?,?,?,?)`, hex.EncodeToString(idRaw[:16]), p.TenantID, p.UserID, tokenHash[:], csrfHash[:],
		expires.Format(time.RFC3339Nano), idleUntil.Unix())
	if err != nil {
		return "", "", time.Time{}, err
	}
	if _, err = conn.ExecContext(context.Background(), "COMMIT"); err != nil {
		return "", "", time.Time{}, err
	}
	committed = true
	return hex.EncodeToString(tokenRaw[:]), hex.EncodeToString(csrfRaw[:]), expires, nil
}

func (s *SQLiteSessionStore) lookup(token string) (rbac.Principal, [32]byte, bool) {
	p, csrf, ok, _ := s.lookupContext(context.Background(), token)
	return p, csrf, ok
}

func (s *SQLiteSessionStore) lookupContext(ctx context.Context, token string) (rbac.Principal, [32]byte, bool, error) {
	return s.lookupWithTimeout(ctx, token, sqliteSessionLookupTimeout)
}

func (s *SQLiteSessionStore) lookupWithTimeout(ctx context.Context, token string, timeout time.Duration) (principal rbac.Principal, csrfHash [32]byte, valid bool, lookupErr error) {
	var empty [32]byte
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer func() {
		if !valid && lookupErr == nil {
			lookupErr = ctx.Err()
		}
	}()
	if s == nil || s.db == nil {
		return rbac.Principal{}, empty, false, nil
	}
	hash, ok := sessionKey(token)
	if !ok {
		return rbac.Principal{}, empty, false, nil
	}
	conn, err := s.db.Conn(queryCtx)
	if err != nil {
		return rbac.Principal{}, empty, false, nil
	}
	defer conn.Close()
	restoreBusyTimeout, err := beginSessionWrite(queryCtx, conn)
	if err != nil {
		return rbac.Principal{}, empty, false, nil
	}
	defer restoreBusyTimeout()
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	// Acquire the SQLite writer reservation before sampling time or reading the
	// session. Otherwise the idle refresh can wait past expiry and revive it.
	now := s.now().UTC()
	var p rbac.Principal
	var csrf []byte
	var expires string
	var idleDeadline int64
	err = conn.QueryRowContext(queryCtx, `SELECT m.identity_id,m.tenant_id,m.role,s.csrf_sha256,s.expires_at,s.idle_deadline_unix
		FROM sessions s
		JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
		JOIN identities i ON i.id=m.identity_id
		JOIN tenants t ON t.id=s.tenant_id AND t.status='active'
		WHERE s.token_sha256=? AND s.revoked_at IS NULL AND i.disabled=0`, hash[:]).
		Scan(&p.UserID, &p.TenantID, &p.Role, &csrf, &expires, &idleDeadline)
	if err != nil || len(csrf) != len(empty) {
		return rbac.Principal{}, empty, false, nil
	}
	parsedExpiry, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil || !validRole(p.Role) {
		return rbac.Principal{}, empty, false, nil
	}
	if !now.Before(parsedExpiry) {
		// Absolute expiry never becomes valid again after a clock rollback.
		if !s.revokeExpiredLookup(queryCtx, conn, now, hash[:]) {
			return rbac.Principal{}, empty, false, nil
		}
		committed = true
		return rbac.Principal{}, empty, false, nil
	}
	if idleDeadline <= now.Unix() {
		// An observed idle timeout is permanently revoked. A competing
		// request that already extended the deadline must not be revoked.
		if !s.revokeExpiredLookup(queryCtx, conn, now, hash[:]) {
			return rbac.Principal{}, empty, false, nil
		}
		committed = true
		return rbac.Principal{}, empty, false, nil
	}
	newIdle := now.Add(sqliteSessionIdle)
	if parsedExpiry.Before(newIdle) {
		newIdle = parsedExpiry
	}
	// A conditional write rechecks revocation and current membership. MAX
	// makes simultaneous valid reads monotonic, so concurrent requests do not
	// falsely expire one another after one refreshes the idle deadline.
	result, err := conn.ExecContext(queryCtx, `UPDATE sessions SET idle_deadline_unix=MAX(idle_deadline_unix,?)
		WHERE token_sha256=? AND revoked_at IS NULL AND idle_deadline_unix>?
		AND EXISTS(SELECT 1 FROM memberships m
			JOIN identities i ON i.id=m.identity_id AND i.disabled=0
			JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
			WHERE m.identity_id=sessions.identity_id AND m.tenant_id=sessions.tenant_id
			AND m.role=?)`, newIdle.Unix(), hash[:], now.Unix(), string(p.Role))
	if err != nil {
		return rbac.Principal{}, empty, false, nil
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return rbac.Principal{}, empty, false, nil
	}
	if _, err := conn.ExecContext(queryCtx, "COMMIT"); err != nil {
		return rbac.Principal{}, empty, false, nil
	}
	committed = true
	copy(empty[:], csrf)
	return p, empty, true, nil
}

// beginSessionWrite avoids SQLite's native busy wait, which cannot promptly
// observe request cancellation. The returned cleanup restores the connection's
// exact prior setting and must be deferred before transaction rollback.
func beginSessionWrite(ctx context.Context, conn *sql.Conn) (func(), error) {
	return database.BeginImmediate(ctx, conn, sqliteSessionLookupTimeout)
}

func (s *SQLiteSessionStore) revokeExpiredLookup(ctx context.Context, conn *sql.Conn, now time.Time, hash []byte) bool {
	result, err := conn.ExecContext(ctx,
		"UPDATE sessions SET revoked_at=? WHERE token_sha256=? AND revoked_at IS NULL",
		now.Format(time.RFC3339Nano), hash)
	if err != nil {
		return false
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return false
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return false
	}
	return true
}

// Check re-reads membership and disabled status so revoked or changed membership
// does not remain authorized until the browser cookie expires.
func (s *SQLiteSessionStore) Check(token string) (rbac.Principal, bool) {
	p, _, ok := s.lookup(token)
	return p, ok
}

// CheckContext revalidates the session while honoring request cancellation.
// Database and invalid-session failures remain indistinguishable; only caller
// cancellation is returned as an error.
func (s *SQLiteSessionStore) CheckContext(ctx context.Context, token string) (rbac.Principal, bool, error) {
	p, _, ok, err := s.lookupContext(ctx, token)
	return p, ok, err
}

func (s *SQLiteSessionStore) CheckCSRF(token, csrf string) bool {
	_, stored, ok := s.lookup(token)
	if !ok {
		return false
	}
	csrfKey, ok := sessionKey(csrf)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare(stored[:], csrfKey[:]) == 1
}

func (s *SQLiteSessionStore) CheckCSRFContext(ctx context.Context, token, csrf string) (bool, error) {
	_, stored, ok, err := s.lookupContext(ctx, token)
	if err != nil || !ok {
		return false, err
	}
	csrfKey, ok := sessionKey(csrf)
	if !ok {
		return false, nil
	}
	return subtle.ConstantTimeCompare(stored[:], csrfKey[:]) == 1, nil
}

func (s *SQLiteSessionStore) Revoke(token string) error {
	if s == nil {
		return ErrInvalidSession
	}
	hash, ok := sessionKey(token)
	if !ok {
		return nil
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at=? WHERE token_sha256=? AND revoked_at IS NULL`, now, hash[:])
	return err
}

func (s *SQLiteSessionStore) RevokeUser(userID string) error {
	if s == nil || userID == "" {
		return ErrInvalidSession
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at=? WHERE identity_id=? AND revoked_at IS NULL`, now, userID)
	return err
}
