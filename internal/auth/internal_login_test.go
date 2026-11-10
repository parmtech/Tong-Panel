package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
	"golang.org/x/crypto/argon2"
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

func waitForDBPoolWait(t *testing.T, db *sql.DB, previous int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for db.Stats().WaitCount == previous && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if db.Stats().WaitCount == previous {
		t.Fatal("operation did not wait for the held database connection")
	}
}

func rejectedTOTPLater(t *testing.T, seed []byte, code string, now time.Time) time.Time {
	t.Helper()
	later := now.Add(2 * time.Minute)
	for range 20 {
		if _, ok := acceptedTOTPStep(seed, code, later); !ok {
			return later
		}
		later = later.Add(30 * time.Second)
	}
	t.Fatal("could not find a rejected TOTP time for this code")
	return time.Time{}
}

func TestInternalLoginRevalidatesTOTPAfterConnectionWait(t *testing.T) {
	for _, expired := range []bool{true, false} {
		name := "within window"
		if expired {
			name = "expired during connection wait"
		}
		t.Run(name, func(t *testing.T) {
			service, db, _, password, clock, seed := setupTrustedLogin(t)
			var currentPHC string
			if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
				t.Fatal(err)
			}
			legacyPHC := strings.Replace(currentPHC, "p=4", "p=2", 1)
			if legacyPHC == currentPHC {
				t.Fatal("test verifier did not produce a rehash-eligible profile")
			}
			if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
				t.Fatal(err)
			}
			service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
			hashStarted := make(chan struct{})
			releaseHash := make(chan struct{})
			service.budget.hash = func([]byte) (string, error) {
				close(hashStarted)
				<-releaseHash
				return currentPHC, nil
			}
			code, err := TOTPCode(seed, *clock)
			if err != nil {
				t.Fatal(err)
			}
			beforeReplay, beforePHC := int64(0), legacyPHC
			if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&beforeReplay); err != nil {
				t.Fatal(err)
			}
			waitsBefore := db.Stats().WaitCount
			type loginResult struct {
				result InternalLoginResult
				err    error
			}
			result := make(chan loginResult, 1)
			go func() {
				got, authErr := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.80", password, code)
				result <- loginResult{result: got, err: authErr}
			}()
			select {
			case <-hashStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("login did not reach verifier upgrade")
			}
			blocker, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			close(releaseHash)
			waitForDBPoolWait(t, db, waitsBefore)
			if expired {
				*clock = rejectedTOTPLater(t, seed, code, *clock)
			} else {
				*clock = clock.Add(10 * time.Second)
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-result:
				if expired {
					if got.err != ErrLoginDenied || got.result.Token != "" {
						t.Fatalf("login accepted a code that expired during the connection wait: %v", got.err)
					}
				} else if got.err != nil || got.result.Token == "" {
					t.Fatalf("login rejected a code still in its accepted window: %v", got.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("login did not complete after the connection was released")
			}
			if expired {
				var sessions, audits int
				var afterPHC string
				var afterReplay int64
				if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.login'").Scan(&audits); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&afterPHC); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&afterReplay); err != nil {
					t.Fatal(err)
				}
				if sessions != 0 || audits != 0 || afterPHC != beforePHC || afterReplay != beforeReplay {
					t.Fatalf("expired login changed durable security state: sessions=%d audits=%d verifierUnchanged=%t replayUnchanged=%t", sessions, audits, afterPHC == beforePHC, afterReplay == beforeReplay)
				}
			}
		})
	}
}

func TestInternalLoginRehashesOutdatedPasswordInSuccessfulTransaction(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	var currentPHC string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
		t.Fatal(err)
	}
	legacyPHC := strings.Replace(currentPHC, "p=4", "p=2", 1)
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
		t.Fatal(err)
	}
	var hashes int
	service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
	service.budget.hash = func(got []byte) (string, error) {
		hashes++
		if !bytes.Equal(got, password) {
			t.Fatal("rehash received a different password")
		}
		return currentPHC, nil
	}
	code, _ := TOTPCode(seed, *clock)
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err != nil {
		t.Fatalf("successful login with outdated verifier failed: %v", err)
	}
	var stored string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != currentPHC || hashes != 1 {
		t.Fatalf("verifier upgrade stored=%t hash calls=%d", stored == currentPHC, hashes)
	}
	var sessions, audits int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.login'").Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if sessions != 1 || audits != 1 {
		t.Fatalf("successful rehash login session/audit counts=%d/%d", sessions, audits)
	}
}

func TestInternalLoginDoesNotDowngradeStrongerSupportedVerifier(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	var currentPHC string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
		t.Fatal(err)
	}
	strongerPHC := strings.Replace(currentPHC, "m=65536,t=3,p=4", "m=131072,t=4,p=2", 1)
	if strongerPHC == currentPHC {
		t.Fatal("test PHC profile replacement did not match")
	}
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", strongerPHC); err != nil {
		t.Fatal(err)
	}
	service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
	hashes := 0
	service.budget.hash = func([]byte) (string, error) {
		hashes++
		return currentPHC, nil
	}
	code, _ := TOTPCode(seed, *clock)
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err != nil {
		t.Fatalf("login with supported stronger verifier failed: %v", err)
	}
	var stored string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != strongerPHC || hashes != 0 {
		t.Fatalf("stronger verifier was downgraded or rehash invoked: stored=%t hash calls=%d", stored == strongerPHC, hashes)
	}
}

func TestInternalLoginRehashFailureDoesNotConsumeMFAOrIssueSession(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	var currentPHC string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
		t.Fatal(err)
	}
	legacyPHC := strings.Replace(currentPHC, "p=4", "p=2", 1)
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
		t.Fatal(err)
	}
	service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
	service.budget.hash = func([]byte) (string, error) { return "", errors.New("synthetic rehash failure") }
	code, _ := TOTPCode(seed, *clock)
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err == nil {
		t.Fatal("login succeeded when verifier upgrade failed")
	}
	var sessions, audits int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.login'").Scan(&audits); err != nil {
		t.Fatal(err)
	}
	var step int
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&step); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || audits != 0 || step != int(clock.Add(-30*time.Second).Unix()/30) {
		t.Fatalf("rehash failure left session/audit/MFA state %d/%d/%d", sessions, audits, step)
	}
}

func TestInternalLoginDoesNotRehashBeforeValidMFA(t *testing.T) {
	service, db, _, password, _, _ := setupTrustedLogin(t)
	var currentPHC string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
		t.Fatal(err)
	}
	legacyPHC := strings.Replace(currentPHC, "p=4", "p=2", 1)
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
		t.Fatal(err)
	}
	service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
	hashes := 0
	service.budget.hash = func([]byte) (string, error) {
		hashes++
		return currentPHC, nil
	}
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, "000000"); !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("invalid MFA returned %v", err)
	}
	if hashes != 0 {
		t.Fatalf("password rehashed before MFA succeeded: calls=%d", hashes)
	}
	var stored string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != legacyPHC {
		t.Fatal("invalid MFA changed the stored password verifier")
	}
}

func TestInternalLoginPreservesValidLegacyShortPassword(t *testing.T) {
	service, db, _, _, clock, seed := setupTrustedLogin(t)
	legacyPassword := []byte("short")
	salt := []byte("0123456789abcdef")
	legacyTime := passwordHashTime
	legacyParallelism := uint8(2)
	hash := argon2.IDKey(legacyPassword, salt, uint32(legacyTime), passwordHashMemoryKiB, legacyParallelism, passwordHashLength)
	legacyPHC := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		passwordHashMemoryKiB, legacyTime, legacyParallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
		t.Fatal(err)
	}
	code, _ := TOTPCode(seed, *clock)
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", legacyPassword, code); err != nil {
		t.Fatalf("valid bounded legacy password was denied during rehash: %v", err)
	}
	var stored string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != legacyPHC {
		t.Fatal("short legacy verifier was unexpectedly replaced")
	}
}

func TestInternalLoginRehashRollsBackWhenAuditFails(t *testing.T) {
	service, db, _, password, clock, seed := setupTrustedLogin(t)
	var currentPHC string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&currentPHC); err != nil {
		t.Fatal(err)
	}
	legacyPHC := strings.Replace(currentPHC, "p=4", "p=2", 1)
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", legacyPHC); err != nil {
		t.Fatal(err)
	}
	service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, password) }
	service.budget.hash = func([]byte) (string, error) { return currentPHC, nil }
	if _, err := db.Exec("CREATE TRIGGER block_login_audit BEFORE INSERT ON audit_events WHEN NEW.action='auth.login' BEGIN SELECT RAISE(ABORT,'audit unavailable'); END"); err != nil {
		t.Fatal(err)
	}
	code, _ := TOTPCode(seed, *clock)
	if _, err := service.AuthenticateTrusted(context.Background(), "tenant-a", "alice", "192.0.2.10", password, code); err == nil {
		t.Fatal("login succeeded without durable audit")
	}
	var stored string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	var sessions, step int
	if err := db.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&step); err != nil {
		t.Fatal(err)
	}
	if stored != legacyPHC || sessions != 0 || step != int(clock.Add(-30*time.Second).Unix()/30) {
		t.Fatalf("audit failure committed rehash/session/MFA state: verifier=%t sessions=%d step=%d", stored == currentPHC, sessions, step)
	}
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
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.login' AND " +
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
	if _, err := db.Exec("CREATE TRIGGER block_login_audit BEFORE INSERT ON audit_events " +
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
		password             []byte
		code                 string
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
