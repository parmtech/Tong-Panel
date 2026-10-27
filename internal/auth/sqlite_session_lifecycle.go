package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

// Rotate replaces an already authenticated SQLite session and its CSRF
// secret atomically. It preserves the original absolute expiry, so repeated
// rotation cannot extend the session's maximum lifetime. No cookie is set.
func (s *SQLiteSessionStore) Rotate(ctx context.Context, oldToken string) (string, string, time.Time, error) {
	if s == nil || s.db == nil {
		return "", "", time.Time{}, ErrInvalidSession
	}
	oldHash, ok := sessionKey(oldToken)
	if !ok {
		return "", "", time.Time{}, ErrInvalidSession
	}
	if err := ctx.Err(); err != nil {
		return "", "", time.Time{}, err
	}
	var newToken, newCSRF [32]byte
	var rawID, rawAudit [16]byte
	for _, item := range [][]byte{newToken[:], newCSRF[:], rawID[:], rawAudit[:]} {
		if _, err := rand.Read(item); err != nil {
			return "", "", time.Time{}, err
		}
	}
	defer clear(newToken[:])
	defer clear(newCSRF[:])
	tokenHash := sha256.Sum256(newToken[:])
	csrfHash := sha256.Sum256(newCSRF[:])
	now := s.now().UTC()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return "", "", time.Time{}, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return "", "", time.Time{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var p rbac.Principal
	var oldID, rawExpiry string
	var idleUntil int64
	err = conn.QueryRowContext(ctx, `SELECT s.id,s.tenant_id,s.identity_id,m.role,s.expires_at,s.idle_deadline_unix
		FROM sessions s
		JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE s.token_sha256=? AND s.revoked_at IS NULL`, oldHash[:]).
		Scan(&oldID, &p.TenantID, &p.UserID, &p.Role, &rawExpiry, &idleUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", time.Time{}, ErrInvalidSession
	}
	if err != nil {
		return "", "", time.Time{}, err
	}
	expires, err := time.Parse(time.RFC3339Nano, rawExpiry)
	if err != nil || !now.Before(expires) || idleUntil <= now.Unix() || !validRole(p.Role) {
		return "", "", time.Time{}, ErrInvalidSession
	}
	deadline := now.Add(sqliteSessionIdle)
	if expires.Before(deadline) {
		deadline = expires
	}
	r, err := conn.ExecContext(ctx,
		"UPDATE sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL",
		now.Format(time.RFC3339Nano), oldID)
	if err != nil {
		return "", "", time.Time{}, err
	}
	rows, err := r.RowsAffected()
	if err != nil || rows != 1 {
		return "", "", time.Time{}, ErrInvalidSession
	}
	newID := hex.EncodeToString(rawID[:])
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at,idle_deadline_unix) VALUES(?,?,?,?,?,?,?)",
		newID, p.TenantID, p.UserID, tokenHash[:], csrfHash[:], rawExpiry, deadline.Unix()); err != nil {
		return "", "", time.Time{}, err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id) "+
			"VALUES(?,?,?,'user','auth.session_rotated',?,?)",
		hex.EncodeToString(rawAudit[:]), p.TenantID, p.UserID, newID, hex.EncodeToString(rawAudit[:])); err != nil {
		return "", "", time.Time{}, err
	}
	if err := ctx.Err(); err != nil {
		return "", "", time.Time{}, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return "", "", time.Time{}, err
	}
	committed = true
	return hex.EncodeToString(newToken[:]), hex.EncodeToString(newCSRF[:]), expires, nil
}

// Logout revokes a current session and records its user-attributed audit in
// one transaction. It does not set/clear HTTP cookies; a separately reviewed
// TLS handler must send auth.ExpiredSessionCookie after successful logout.
func (s *SQLiteSessionStore) Logout(ctx context.Context, token string) error {
	if s == nil || s.db == nil {
		return ErrInvalidSession
	}
	hash, ok := sessionKey(token)
	if !ok {
		return ErrInvalidSession
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
	var sessionID string
	var p rbac.Principal
	err = conn.QueryRowContext(ctx, `SELECT s.id,s.tenant_id,s.identity_id,m.role FROM sessions s
		JOIN memberships m ON m.tenant_id=s.tenant_id AND m.identity_id=s.identity_id
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		WHERE s.token_sha256=? AND s.revoked_at IS NULL`, hash[:]).
		Scan(&sessionID, &p.TenantID, &p.UserID, &p.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidSession
	}
	if err != nil {
		return err
	}
	if !validRole(p.Role) {
		return ErrInvalidSession
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	if _, err := conn.ExecContext(ctx,
		"UPDATE sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL", now, sessionID); err != nil {
		return err
	}
	auditID, err := bootstrapID()
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id) "+
			"VALUES(?,?,?,'user','auth.logout',?,?)",
		auditID, p.TenantID, p.UserID, sessionID, auditID); err != nil {
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
