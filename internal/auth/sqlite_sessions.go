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

// SQLiteSessionStore persists revocable session hashes but does not implement
// login, password verification, MFA, HTTP cookies, CSRF middleware, or rate limits.
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
	tokenHash := sha256.Sum256(tokenRaw[:])
	csrfHash := sha256.Sum256(csrfRaw[:])
	tx, err := s.db.Begin()
	if err != nil {
		return "", "", time.Time{}, err
	}
	defer tx.Rollback()
	var role string
	err = tx.QueryRow(`SELECT m.role FROM memberships m JOIN identities i ON i.id=m.identity_id
		WHERE m.tenant_id=? AND m.identity_id=? AND i.disabled=0`, p.TenantID, p.UserID).Scan(&role)
	if err != nil || rbac.Role(role) != p.Role {
		return "", "", time.Time{}, ErrInvalidSession
	}
	_, err = tx.Exec(`INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at)
		VALUES(?,?,?,?,?,?)`, hex.EncodeToString(idRaw[:16]), p.TenantID, p.UserID, tokenHash[:], csrfHash[:], expires.Format(time.RFC3339Nano))
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
	err := s.db.QueryRow(`SELECT m.identity_id,m.tenant_id,m.role,s.csrf_sha256,s.expires_at
		FROM sessions s
		JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
		JOIN identities i ON i.id=m.identity_id
		WHERE s.token_sha256=? AND s.revoked_at IS NULL AND i.disabled=0`, hash[:]).
		Scan(&p.UserID, &p.TenantID, &p.Role, &csrf, &expires)
	if err != nil || len(csrf) != len(empty) {
		return rbac.Principal{}, empty, false
	}
	parsedExpiry, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil || !s.now().Before(parsedExpiry) || !validRole(p.Role) {
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
