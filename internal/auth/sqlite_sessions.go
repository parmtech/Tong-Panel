package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

// SQLiteSessionStore persists revocable token hashes with absolute and idle
// expiry. It does not expose HTTP login, cookie issuing, or password verification.
const sqliteSessionIdle = 30 * time.Minute

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
	now := s.now().UTC()
	expires = now.Add(lifetime)
	idleUntil := now.Add(sqliteSessionIdle)
	if expires.Before(idleUntil) {
		idleUntil = expires
	}
	tokenHash := sha256.Sum256(tokenRaw[:])
	csrfHash := sha256.Sum256(csrfRaw[:])
	tx, err := s.db.Begin()
	if err != nil {
		return "", "", time.Time{}, err
	}
	defer tx.Rollback()
	var role string
	err = tx.QueryRow(`SELECT m.role FROM memberships m
		JOIN identities i ON i.id=m.identity_id
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE m.tenant_id=? AND m.identity_id=? AND i.disabled=0`, p.TenantID, p.UserID).Scan(&role)
	if err != nil || rbac.Role(role) != p.Role {
		return "", "", time.Time{}, ErrInvalidSession
	}
	_, err = tx.Exec(`INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at,idle_deadline_unix)
		VALUES(?,?,?,?,?,?,?)`, hex.EncodeToString(idRaw[:16]), p.TenantID, p.UserID, tokenHash[:], csrfHash[:],
		expires.Format(time.RFC3339Nano), idleUntil.Unix())
	if err != nil {
		return "", "", time.Time{}, err
	}
	if err = tx.Commit(); err != nil {
		return "", "", time.Time{}, err
	}
	return hex.EncodeToString(tokenRaw[:]), hex.EncodeToString(csrfRaw[:]), expires, nil
}

func (s *SQLiteSessionStore) lookup(token string) (rbac.Principal, [32]byte, bool) {
	var empty [32]byte
	if s == nil {
		return rbac.Principal{}, empty, false
	}
	hash, ok := sessionKey(token)
	if !ok {
		return rbac.Principal{}, empty, false
	}
	var p rbac.Principal
	var csrf []byte
	var expires string
	var idleDeadline int64
	err := s.db.QueryRow(`SELECT m.identity_id,m.tenant_id,m.role,s.csrf_sha256,s.expires_at,s.idle_deadline_unix
		FROM sessions s
		JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
		JOIN identities i ON i.id=m.identity_id
		JOIN tenants t ON t.id=s.tenant_id AND t.status='active'
		WHERE s.token_sha256=? AND s.revoked_at IS NULL AND i.disabled=0`, hash[:]).
		Scan(&p.UserID, &p.TenantID, &p.Role, &csrf, &expires, &idleDeadline)
	if err != nil || len(csrf) != len(empty) {
		return rbac.Principal{}, empty, false
	}
	now := s.now().UTC()
	parsedExpiry, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil || !validRole(p.Role) {
		return rbac.Principal{}, empty, false
	}
	if !now.Before(parsedExpiry) || idleDeadline <= now.Unix() {
		// Permanently invalidate sessions observed past an idle or absolute
		// deadline. A later clock rollback must not revive them. The WHERE
		// clause avoids revoking an independently refreshed idle deadline.
		_, _ = s.db.Exec(`UPDATE sessions SET revoked_at=?
			WHERE token_sha256=? AND revoked_at IS NULL
			AND (idle_deadline_unix<=? OR expires_at=?)`,
			now.Format(time.RFC3339Nano), hash[:], now.Unix(), expires)
		return rbac.Principal{}, empty, false
	}
	newIdle := now.Add(sqliteSessionIdle)
	if parsedExpiry.Before(newIdle) {
		newIdle = parsedExpiry
	}
	// A conditional write rechecks revocation and current membership. MAX
	// makes simultaneous valid reads monotonic, so concurrent requests do not
	// falsely expire one another after one refreshes the idle deadline.
	result, err := s.db.Exec(`UPDATE sessions SET idle_deadline_unix=MAX(idle_deadline_unix,?)
		WHERE token_sha256=? AND revoked_at IS NULL AND idle_deadline_unix>?
		AND EXISTS(SELECT 1 FROM memberships m
			JOIN identities i ON i.id=m.identity_id AND i.disabled=0
			JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
			WHERE m.identity_id=sessions.identity_id AND m.tenant_id=sessions.tenant_id
			AND m.role=?)`, newIdle.Unix(), hash[:], now.Unix(), string(p.Role))
	if err != nil {
		return rbac.Principal{}, empty, false
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return rbac.Principal{}, empty, false
	}
	copy(empty[:], csrf)
	return p, empty, true
}

// Check re-reads membership and disabled status so revoked or changed membership
// does not remain authorized until the browser cookie expires.
func (s *SQLiteSessionStore) Check(token string) (rbac.Principal, bool) {
	p, _, ok := s.lookup(token)
	return p, ok
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
