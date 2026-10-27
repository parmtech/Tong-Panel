package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func setupTrustedLogin(t *testing.T) (*InternalLoginService, *sql.DB, *SQLiteSessionStore, []byte, *time.Time, []byte) {
	t.Helper()
	_, db, sessions := openSessionDB(t)
	now := time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)
	clock := &now
	password := []byte("long independent owner password for login")
	phc, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", phc); err != nil {
		t.Fatal(err)
	}
	box := newTestMFABox(t)
	enrollment, err := NewMFAEnrollmentStore(db, box)
	if err != nil {
		t.Fatal(err)
	}
	enrollment.now = func() time.Time { return *clock }
	seed, err := enrollment.BeginTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	code, err := TOTPCode(seed, now)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := enrollment.ConfirmTrusted(context.Background(), "tenant-a", "user-a", code); err != nil || !ok {
		t.Fatalf("trusted MFA enrollment: %v %v", ok, err)
	}
	*clock = clock.Add(30 * time.Second)
	limit, err := NewSQLiteLoginLimiter(db, bytes.Repeat([]byte{0x44}, 32))
	if err != nil {
		t.Fatal(err)
	}
	limit.now = func() time.Time { return *clock }
	budget, err := NewPasswordVerifyBudget(2)
	if err != nil {
		t.Fatal(err)
	}
	login, err := NewInternalLoginService(db, box, limit, budget)
	if err != nil {
		t.Fatal(err)
	}
	login.now = func() time.Time { return *clock }
	sessions.now = func() time.Time { return *clock }
	return login, db, sessions, password, clock, seed
}

func TestInternalLoginIssuesRevocableSessionWithAtomicAudit(t *testing.T) {
	service, db, sessions, password, clock, seed := setupTrustedLogin(t)
	code, _ := TOTPCode(seed, *clock)
	result, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code)
	if err != nil {
		t.Fatalf("first MFA/password login refused: %v", err)
	}
	if result.Token == "" || result.CSRF == "" || result.Token == result.CSRF ||
		result.Principal.Role != rbac.Owner || result.Principal.UserID != "user-a" ||
		result.Expires != clock.Add(30*time.Minute) {
		t.Fatalf("invalid issued login session metadata: %+v", result.Principal)
	}
	if p, ok := sessions.Check(result.Token); !ok || p.UserID != "user-a" ||
		!sessions.CheckCSRF(result.Token, result.CSRF) {
		t.Fatal("login did not create a verifiable session/CSRF pair")
	}
	var auditCount int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.login' AND "+
		"actor_kind='user' AND actor_identity_id='user-a'").Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("successful authentication audit missing: %d %v", auditCount, err)
	}
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.11", password, code); !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("same OTP minted second session: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("replayed OTP committed a session: %d %v", auditCount, err)
	}
	if err := sessions.Revoke(result.Token); err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions.Check(result.Token); ok {
		t.Fatal("revoked login session accepted")
	}
}

func TestInternalLoginAuditFailureDoesNotBurnOTPOrIssueSession(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	code, _ := TOTPCode(seed, *clock)
	if _, err := db.Exec("CREATE TRIGGER block_login_audit BEFORE INSERT ON audit_events "+
		"WHEN NEW.action='auth.login' BEGIN SELECT RAISE(ABORT,'audit unavailable'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err == nil {
		t.Fatal("authentication succeeded without durable audit")
	}
	var sessionsCount, previousStep int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessionsCount); err != nil || sessionsCount != 0 {
		t.Fatalf("failed audit persisted session: %d %v", sessionsCount, err)
	}
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&previousStep); err != nil {
		t.Fatal(err)
	}
	if previousStep != int(clock.Add(-30*time.Second).Unix()/30) {
		t.Fatalf("failed audit burned MFA step: %d", previousStep)
	}
	if _, err := db.Exec("DROP TRIGGER block_login_audit"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err != nil {
		t.Fatalf("retry after restored audit failed: %v", err)
	}
}

func TestInternalLoginDeniesUnknownWrongPasswordOrUnenrolledMFA(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	code, _ := TOTPCode(seed, *clock)
	for _, tc := range []struct {
		user, tenant, source string
		password []byte
		code string
	}{
		{"unknown", "tenant-a", "192.0.2.10", password, code},
		{"alice", "tenant-a", "192.0.2.11", []byte("incorrect-password"), code},
		{"alice", "tenant-b", "192.0.2.12", password, code},
		{"alice", "tenant-a", "192.0.2.13", password, "000000"},
	} {
		if _, err := service.AuthenticateTrusted(context.Background(), tc.tenant, tc.user, tc.source, tc.password, tc.code); !errors.Is(err, ErrLoginDenied) {
			t.Fatalf("untrusted login for %s/%s produced %v", tc.tenant, tc.user, err)
		}
	}
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.15", password, code); !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("disabled identity logged in: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid login wrote sessions: %d %v", count, err)
	}
}

func TestInternalLoginConcurrentOTPOnlyOneWinner(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	code, _ := TOTPCode(seed, *clock)
	var wg sync.WaitGroup
	var successes int
	var mu sync.Mutex
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice",
				"198.51.100."+string(rune('1'+i)), password, code)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else if !errors.Is(err, ErrLoginDenied) {
				t.Errorf("unexpected concurrent login error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("same OTP authorized %d sessions", successes)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&count); err != nil || count != 1 {
		t.Fatalf("concurrent MFA session count=%d err=%v", count, err)
	}
}

func TestInternalLoginRejectsMissingDependenciesAndLimitsAttempts(t *testing.T) {
	if _, err := NewInternalLoginService(nil, nil, nil, nil); !errors.Is(err, ErrLoginDenied) {
		t.Fatal("invalid constructor accepted")
	}
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	code, _ := TOTPCode(seed, *clock)
	for i := 0; i < maxAccountAttempts; i++ {
		_, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice",
			"203.0.113.40", []byte("wrong password"), code)
		if !errors.Is(err, ErrLoginDenied) {
			t.Fatalf("incorrect password attempt did not deny: %v", err)
		}
	}
	_, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice",
		"203.0.113.40", password, code)
	if !errors.Is(err, ErrLoginDenied) {
		t.Fatal("login bypassed persistent account attempt budget")
	}
	var sessionsCount int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessionsCount); err != nil || sessionsCount != 0 {
		t.Fatalf("throttled login wrote session: %d %v", sessionsCount, err)
	}
}
