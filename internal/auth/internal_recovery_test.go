package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

const admissionTestPHC = "$argon2id$v=19$m=65536,t=3,p=4$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"

func setupInternalRecovery(t *testing.T) (*InternalRecoveryService, *SQLiteRecoveryStore, *sql.DB, *SQLiteSessionStore, []string, []byte, time.Time) {
	t.Helper()
	_, db, sessions, store := recoveryAdmissionFixture(t)
	password := []byte("verified recovery password")
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", admissionTestPHC); err != nil {
		t.Fatal(err)
	}
	limiter, err := NewSQLiteLoginLimiter(db, bytes.Repeat([]byte{0x44}, 32))
	if err != nil {
		t.Fatal(err)
	}
	budget, err := NewPasswordVerifyBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewInternalRecoveryService(db, store, limiter, budget)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 27, 2, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	store.now = func() time.Time { return now }
	service.now = func() time.Time { return now }
	codes, err := store.IssueTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	return service, store, db, sessions, codes, password, now
}

// recoveryAdmissionFixture reuses the same MFA schema setup as the recovery
// store tests, while keeping its recovery set issuance in each test explicit.
func recoveryAdmissionFixture(t *testing.T) (string, *sql.DB, *SQLiteSessionStore, *SQLiteRecoveryStore) {
	t.Helper()
	path, db, sessions := openSessionDB(t)
	box := newTestMFABox(t)
	envelope, err := box.Seal("user-a", bytes.Repeat([]byte{7}, 20))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO mfa_totp_enrollments(identity_id,secret_envelope,status,pending_expires_at,activated_at)
		VALUES('user-a',?,'active','2026-10-27T00:00:00Z','2026-10-27T00:00:00Z')`, envelope)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteRecoveryStore(db, bytes.Repeat([]byte{0x93}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return path, db, sessions, store
}

func TestInternalRecoveryRequiresPasswordAndRevokesSessionsAcrossTenants(t *testing.T) {
	service, _, db, sessions, codes, password, now := setupInternalRecovery(t)
	service.budget.verify = func(got []byte, phc string) bool {
		return bytes.Equal(got, password) && phc == admissionTestPHC
	}
	sessions.now = func() time.Time { return now }
	if _, err := db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-b','user-a','customer')"); err != nil {
		t.Fatal(err)
	}
	first, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	second, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-b", UserID: "user-a", Role: rbac.Customer}, time.Hour)
	if err != nil {
		var actual string
		_ = db.QueryRow("SELECT role FROM memberships WHERE tenant_id='tenant-b' AND identity_id='user-a'").Scan(&actual)
		t.Fatalf("issue tenant-b session role=%q err=%v", actual, err)
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.21", password, codes[0]); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{first, second} {
		if _, ok := sessions.Check(token); ok {
			t.Fatal("successful recovery left an identity session active")
		}
	}
	var rows, consumed int
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&rows); err != nil || rows != recoveryCodeCount-1 {
		t.Fatalf("recovery code rows=%d err=%v", rows, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.recovery_consumed'").Scan(&consumed); err != nil || consumed != 1 {
		t.Fatalf("successful recovery audit=%d err=%v", consumed, err)
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.21", password, codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("replayed recovery accepted: %v", err)
	}
	var sessionsCreated int
	if err := db.QueryRow("SELECT count(*) FROM sessions WHERE identity_id='user-a'").Scan(&sessionsCreated); err != nil || sessionsCreated != 2 {
		t.Fatalf("recovery unexpectedly issued a session: count=%d err=%v", sessionsCreated, err)
	}
}

func TestInternalRecoveryWrongPasswordDoesNotConsumeOrRevoke(t *testing.T) {
	service, _, db, sessions, codes, _, now := setupInternalRecovery(t)
	service.budget.verify = func(got []byte, phc string) bool {
		return bytes.Equal(got, []byte("verified recovery password")) && phc == admissionTestPHC
	}
	sessions.now = func() time.Time { return now }
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.22", []byte("wrong password"), codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("wrong password error=%v", err)
	}
	if _, ok := sessions.Check(token); !ok {
		t.Fatal("wrong password revoked the session")
	}
	var rows, consumed int
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&rows); err != nil || rows != recoveryCodeCount {
		t.Fatalf("wrong password changed recovery-code rows=%d err=%v", rows, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.recovery_consumed'").Scan(&consumed); err != nil || consumed != 0 {
		t.Fatalf("wrong password wrote success audit=%d err=%v", consumed, err)
	}
}

func TestInternalRecoveryLimiterRunsBeforePasswordVerification(t *testing.T) {
	service, _, _, _, codes, password, _ := setupInternalRecovery(t)
	verified := 0
	service.budget.verify = func([]byte, string) bool { verified++; return true }
	for i := 0; i < maxAccountAttempts; i++ {
		allowed, err := service.limiter.Admit(context.Background(), "tenant-a", "alice", "192.0.2.23")
		if err != nil || !allowed {
			t.Fatalf("pre-admit %d: allowed=%t err=%v", i, allowed, err)
		}
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.23", password, codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("rate-limited recovery error=%v", err)
	}
	if verified != 0 {
		t.Fatalf("password KDF ran %d times after limiter denial", verified)
	}
}

func TestInternalRecoveryUsesDummyVerifierForAbsentDisabledAndCorruptAccounts(t *testing.T) {
	service, _, db, _, codes, password, _ := setupInternalRecovery(t)
	var verified []string
	service.budget.verify = func(_ []byte, phc string) bool {
		verified = append(verified, phc)
		return false
	}
	for _, tc := range []struct {
		name string
		user string
		prep string
	}{
		{name: "absent", user: "nobody"},
		{name: "disabled", user: "alice", prep: "UPDATE identities SET disabled=1 WHERE id='user-a'"},
		{name: "corrupt", user: "alice", prep: "UPDATE identities SET disabled=0,password_phc='$argon2id$corrupt' WHERE id='user-a'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.prep != "" {
				if _, err := db.Exec(tc.prep); err != nil {
					t.Fatal(err)
				}
			}
			before := len(verified)
			if err := service.RecoverTrusted(context.Background(), "tenant-a", tc.user, "192.0.2.24", password, codes[0]); !errors.Is(err, ErrRecoveryDenied) {
				t.Fatalf("generic denial error=%v", err)
			}
			if len(verified) != before+1 || verified[before] != service.dummyPHC {
				t.Fatalf("expected one dummy verification, observed %q", verified[before:])
			}
		})
	}
}

func TestInternalRecoveryRechecksCredentialRoleAndEnrollmentSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change string
	}{
		{name: "password PHC", change: "UPDATE identities SET password_phc=substr(password_phc,1,length(password_phc)-1)||'A' WHERE id='user-a'"},
		{name: "membership role", change: "UPDATE memberships SET role='admin' WHERE tenant_id='tenant-a' AND identity_id='user-a'"},
		{name: "MFA enrollment envelope", change: "UPDATE mfa_totp_enrollments SET secret_envelope=secret_envelope||'x' WHERE identity_id='user-a'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, _, db, sessions, codes, password, now := setupInternalRecovery(t)
			sessions.now = func() time.Time { return now }
			token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			service.budget.verify = func(got []byte, _ string) bool {
				if !bytes.Equal(got, password) {
					t.Fatal("unexpected password passed to verifier")
				}
				if _, err := db.Exec(tc.change); err != nil {
					t.Fatal(err)
				}
				return true
			}
			if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.25", password, codes[0]); !errors.Is(err, ErrRecoveryDenied) {
				t.Fatalf("TOCTOU change accepted: %v", err)
			}
			if _, ok := sessions.Check(token); !ok {
				t.Fatal("TOCTOU denial revoked a session")
			}
			var count int
			if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&count); err != nil || count != recoveryCodeCount {
				t.Fatalf("TOCTOU denial consumed a code: count=%d err=%v", count, err)
			}
		})
	}
}

func TestInternalRecoveryAuditFailureRollsBackConsumptionAndRevocation(t *testing.T) {
	service, _, db, sessions, codes, password, now := setupInternalRecovery(t)
	service.budget.verify = func(got []byte, phc string) bool {
		return bytes.Equal(got, password) && phc == admissionTestPHC
	}
	sessions.now = func() time.Time { return now }
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_internal_recovery BEFORE INSERT ON audit_events
		WHEN NEW.action='auth.recovery_consumed' BEGIN SELECT RAISE(ABORT,'test audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.26", password, codes[0]); err == nil {
		t.Fatal("recovery succeeded without audit")
	}
	if _, ok := sessions.Check(token); !ok {
		t.Fatal("failed recovery revoked an existing session")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&count); err != nil || count != recoveryCodeCount {
		t.Fatalf("failed audit did not roll back code consumption: count=%d err=%v", count, err)
	}
}

func TestInternalRecoveryRunsRealArgon2ForWrongAndCorrectPasswords(t *testing.T) {
	service, _, db, _, codes, password, _ := setupInternalRecovery(t)
	phc, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", phc); err != nil {
		t.Fatal(err)
	}
	service.budget.verify = VerifyPassword
	ctx := context.Background()
	if err := service.RecoverTrusted(ctx, "tenant-a", "alice", "192.0.2.28", []byte("wrong but long enough password"), codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("wrong password error=%v", err)
	}
	var remaining int
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&remaining); err != nil || remaining != recoveryCodeCount {
		t.Fatalf("real KDF wrong password consumed code: %d %v", remaining, err)
	}
	if err := service.RecoverTrusted(ctx, "tenant-a", "alice", "192.0.2.28", password, codes[0]); err != nil {
		t.Fatalf("real Argon2 password denied recovery: %v", err)
	}
}

func TestNewInternalRecoveryServiceRequiresSharedDatabase(t *testing.T) {
	_, db, _, store := recoveryAdmissionFixture(t)
	_, otherDB, _, _ := recoveryAdmissionFixture(t)
	limiter, err := NewSQLiteLoginLimiter(db, bytes.Repeat([]byte{0x44}, 32))
	if err != nil {
		t.Fatal(err)
	}
	budget, err := NewPasswordVerifyBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewInternalRecoveryService(db, store, limiter, budget); err != nil {
		t.Fatal(err)
	}
	otherStore, err := NewSQLiteRecoveryStore(otherDB, bytes.Repeat([]byte{0x93}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewInternalRecoveryService(db, otherStore, limiter, budget); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("separate DB handle accepted: %v", err)
	}
}

func TestInternalRecoveryWrongCodeAttemptsRemainDurable(t *testing.T) {
	service, _, db, _, codes, password, _ := setupInternalRecovery(t)
	service.budget.verify = func(got []byte, phc string) bool {
		return bytes.Equal(got, password) && phc == admissionTestPHC
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.27", password, fmt.Sprintf("%064x", 0)); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("wrong recovery code error=%v", err)
	}
	var attempts, denied int
	if err := db.QueryRow("SELECT attempts FROM mfa_recovery_sets WHERE tenant_id='tenant-a' AND identity_id='user-a'").Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("durable recovery attempts=%d err=%v", attempts, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.recovery_denied'").Scan(&denied); err != nil || denied != 1 {
		t.Fatalf("wrong-code denial audit=%d err=%v", denied, err)
	}
	if err := service.RecoverTrusted(context.Background(), "tenant-a", "alice", "192.0.2.27", password, codes[0]); err != nil {
		t.Fatalf("valid recovery after wrong code failed: %v", err)
	}
}
