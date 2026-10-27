package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrBootstrapUnavailable covers an already initialized panel, invalid
// parameters or missing security preconditions. No public route uses it.
var ErrBootstrapUnavailable = errors.New("local owner bootstrap unavailable")

type BootstrapResult struct {
	TenantID   string
	IdentityID string
	Username   string
}

func bootstrapUsername(username string) bool {
	if len(username) < 3 || len(username) > 80 {
		return false
	}
	for i, c := range username {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			continue
		}
		if i > 0 && (c == '-' || c == '_' || c == '.') {
			continue
		}
		return false
	}
	return true
}

func bootstrapTenantName(value string) bool {
	if len(value) == 0 || len(value) > 160 || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func bootstrapID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

// BootstrapOwner creates the first local owner in an otherwise empty v5
// metadata database. Hashing precedes locking. BEGIN IMMEDIATE serializes
// competing bootstraps. Account, role and audit commit or roll back together.
// The local CLI MUST verify secure file ownership and mode before calling it.
// Never expose this method over HTTP, background jobs or agent commands.
func BootstrapOwner(ctx context.Context, db *sql.DB, username, tenantName string, password []byte) (BootstrapResult, error) {
	var empty BootstrapResult
	if db == nil || !bootstrapUsername(username) || !bootstrapTenantName(tenantName) {
		return empty, ErrBootstrapUnavailable
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	phc, err := HashPassword(password)
	if err != nil {
		return empty, err
	}
	userID, err := bootstrapID()
	if err != nil {
		return empty, err
	}
	tenantID, err := bootstrapID()
	if err != nil {
		return empty, err
	}
	auditID, err := bootstrapID()
	if err != nil {
		return empty, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return empty, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return empty, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	var version int
	if err := conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return empty, err
	}
	if version != 5 {
		return empty, ErrBootstrapUnavailable
	}
	var populated int
	query := "SELECT (SELECT COUNT(*) FROM tenants) +" +
		" (SELECT COUNT(*) FROM identities) +" +
		" (SELECT COUNT(*) FROM memberships) +" +
		" (SELECT COUNT(*) FROM sessions) +" +
		" (SELECT COUNT(*) FROM jobs) +" +
		" (SELECT COUNT(*) FROM audit_events) +" +
		" (SELECT COUNT(*) FROM mfa_totp_enrollments) +" +
		" (SELECT COUNT(*) FROM totp_replay_state)"
	if err := conn.QueryRowContext(ctx, query).Scan(&populated); err != nil {
		return empty, err
	}
	if populated != 0 {
		return empty, ErrBootstrapUnavailable
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO tenants(id,name) VALUES(?,?)", tenantID, tenantName); err != nil {
		return empty, err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO identities(id,username,password_phc) VALUES(?,?,?)",
		userID, username, phc); err != nil {
		return empty, err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO memberships(tenant_id,identity_id,role) VALUES(?,?,'owner')",
		tenantID, userID); err != nil {
		return empty, err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id) "+
			"VALUES(?,?,?,'user','owner.bootstrap',?,?)",
		auditID, tenantID, userID, userID, auditID); err != nil {
		return empty, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return empty, err
	}
	committed = true
	return BootstrapResult{TenantID: tenantID, IdentityID: userID, Username: username}, nil
}
