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

// ErrLoginDenied intentionally covers wrong credentials, disabled tenants,
// missing MFA, exhausted attempt budgets and already-consumed TOTP codes.
var ErrLoginDenied = errors.New("authentication denied")

// InternalLoginService is for reviewed, trusted server-side callers ONLY.
// There is deliberately NO public HTTP login endpoint, cookie delivery,
// reverse-proxy trust or external key provisioning in this change.
type InternalLoginService struct {
	db       *sql.DB
	box      *MFASecretBox
	limiter  *SQLiteLoginLimiter
	budget   *PasswordVerifyBudget
	dummyPHC string
	now      func() time.Time
}

type InternalLoginResult struct {
	Token     string
	CSRF      string
	Expires   time.Time
	Principal rbac.Principal
}

// NewInternalLoginService initializes one generic Argon2id dummy verifier so
// a nonexistent or disabled account still pays the same bounded KDF cost.
// Only a trusted process should construct this; never construct per request.
func NewInternalLoginService(db *sql.DB, box *MFASecretBox, limiter *SQLiteLoginLimiter, budget *PasswordVerifyBudget) (*InternalLoginService, error) {
	if db == nil || box == nil || box.aead == nil ||
		limiter == nil || limiter.db != db || budget == nil ||
		budget.slots == nil || budget.verify == nil {
		return nil, ErrLoginDenied
	}
	dummy, err := HashPassword([]byte("TongPanel internal nonexistent-account dummy password"))
	if err != nil {
		return nil, err
	}
	return &InternalLoginService{
		db: db, box: box, limiter: limiter, budget: budget,
		dummyPHC: dummy, now: time.Now,
	}, nil
}

// AuthenticateTrusted accepts a trusted socket-derived peer IP and server-side
// tenant context, NOT client-controlled proxy headers or an arbitrary tenant.
// It applies the durable attempt gate before Argon2id, validates active
// membership and an activated AES-GCM-protected seed, then commits the TOTP
// replay step, session and attribution audit in ONE SQLite writer transaction.
// Never expose this method directly to a developer-token API or public HTTP.
func (s *InternalLoginService) AuthenticateTrusted(ctx context.Context, tenant, username, peerIP string, password []byte, code string) (InternalLoginResult, error) {
	var denied InternalLoginResult
	if s == nil || s.db == nil || s.box == nil || s.limiter == nil || s.budget == nil {
		return denied, ErrLoginDenied
	}
	if err := ctx.Err(); err != nil {
		return denied, err
	}
	allowed, err := s.limiter.Admit(ctx, tenant, username, peerIP)
	if err != nil {
		return denied, err
	}
	if !allowed {
		return denied, ErrLoginDenied
	}

	var identity, role, storedPHC string
	err = s.db.QueryRowContext(ctx,
		"SELECT i.id,m.role,i.password_phc FROM identities i "+
			"JOIN memberships m ON m.identity_id=i.id "+
			"JOIN tenants t ON t.id=m.tenant_id AND t.status='active' "+
			"WHERE i.username=? AND m.tenant_id=? AND i.disabled=0 "+
			"AND i.password_phc IS NOT NULL", username, tenant).
		Scan(&identity, &role, &storedPHC)
	exists := err == nil && validRole(rbac.Role(role))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return denied, err
	}
	verifyPHC := storedPHC
	if !exists {
		verifyPHC = s.dummyPHC
	} else if _, valid := parseArgon2PHC(storedPHC); !valid {
		// Corrupted legacy verifier still performs the generic KDF cost.
		verifyPHC = s.dummyPHC
		exists = false
	}
	ok, err := s.budget.Verify(password, verifyPHC)
	if err != nil {
		return denied, ErrLoginDenied
	}
	if !ok || !exists {
		return denied, ErrLoginDenied
	}

	var envelope string
	err = s.db.QueryRowContext(ctx,
		"SELECT secret_envelope FROM mfa_totp_enrollments "+
			"WHERE identity_id=? AND status='active'", identity).
		Scan(&envelope)
	if errors.Is(err, sql.ErrNoRows) {
		return denied, ErrLoginDenied
	}
	if err != nil {
		return denied, err
	}
	secret, err := s.box.Open(identity, envelope)
	if err != nil {
		return denied, ErrLoginDenied
	}
	defer clear(secret)
	now := s.now().UTC()
	step, ok := acceptedTOTPStep(secret, code, now)
	if !ok {
		return denied, ErrLoginDenied
	}
	var rawToken, rawCSRF [32]byte
	var rawID, rawAudit [16]byte
	for _, value := range [][]byte{rawToken[:], rawCSRF[:], rawID[:], rawAudit[:]} {
		if _, err := rand.Read(value); err != nil {
			return denied, err
		}
	}
	defer clear(rawToken[:])
	defer clear(rawCSRF[:])
	lifetime := 30 * time.Minute // interim short absolute lifetime; not an idle timer
	expires := now.Add(lifetime)
	sessionID := hex.EncodeToString(rawID[:])
	auditID := hex.EncodeToString(rawAudit[:])
	tokenHash := sha256.Sum256(rawToken[:])
	csrfHash := sha256.Sum256(rawCSRF[:])

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return denied, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return denied, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	// Protect against account disable, tenant/role change, password reset or
	// MFA secret rotation between the KDF and this writer transaction.
	var currentRole string
	err = conn.QueryRowContext(ctx,
		"SELECT m.role FROM identities i "+
			"JOIN memberships m ON m.identity_id=i.id AND m.tenant_id=? "+
			"JOIN tenants t ON t.id=m.tenant_id AND t.status='active' "+
			"JOIN mfa_totp_enrollments e ON e.identity_id=i.id AND e.status='active' "+
			"WHERE i.id=? AND i.username=? AND i.password_phc=? "+
			"AND i.disabled=0 AND e.secret_envelope=?",
		tenant, identity, username, storedPHC, envelope).Scan(&currentRole)
	if errors.Is(err, sql.ErrNoRows) || currentRole != role {
		return denied, ErrLoginDenied
	}
	if err != nil {
		return denied, err
	}
	r, err := conn.ExecContext(ctx,
		"INSERT INTO totp_replay_state(identity_id,last_accepted_step) VALUES(?,?) "+
			"ON CONFLICT(identity_id) DO UPDATE SET "+
			"last_accepted_step=excluded.last_accepted_step,"+
			"updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') "+
			"WHERE excluded.last_accepted_step>totp_replay_state.last_accepted_step",
		identity, step)
	if err != nil {
		return denied, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return denied, err
	}
	if n != 1 {
		return denied, ErrLoginDenied
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at) "+
			"VALUES(?,?,?,?,?,?)",
		sessionID, tenant, identity, tokenHash[:], csrfHash[:],
		expires.Format(time.RFC3339Nano)); err != nil {
		return denied, err
	}
	if _, err := conn.ExecContext(ctx,
		"INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,action,subject_id,request_id) "+
			"VALUES(?,?,?,'user','auth.login',?,?)",
		auditID, tenant, identity, sessionID, auditID); err != nil {
		return denied, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return denied, err
	}
	committed = true
	return InternalLoginResult{
		Token:     hex.EncodeToString(rawToken[:]),
		CSRF:      hex.EncodeToString(rawCSRF[:]),
		Expires:   expires,
		Principal: rbac.Principal{UserID: identity, TenantID: tenant, Role: rbac.Role(role)},
	}, nil
}
