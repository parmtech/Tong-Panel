package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func recoveryFixture(t *testing.T) (string, *sql.DB, *SQLiteRecoveryStore) {
	t.Helper()
	path, db, _ := openSessionDB(t)
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
	return path, db, store
}

func TestRecoveryPersistenceReplayAndIdentityWideRevocation(t *testing.T) {
	path, db, store := recoveryFixture(t)
	ctx := context.Background()
	if _, err := db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-b','user-a','customer')"); err != nil {
		t.Fatal(err)
	}
	sessions, _ := NewSQLiteSessionStore(db)
	a, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-b", UserID: "user-a", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := store.IssueTrusted(ctx, "tenant-a", "user-a")
	if err != nil || len(codes) != recoveryCodeCount {
		t.Fatalf("issue: %v", err)
	}
	var stored []byte
	if err := db.QueryRow("SELECT code_sha256 FROM mfa_recovery_codes LIMIT 1").Scan(&stored); err != nil || len(stored) != 32 {
		t.Fatalf("stored hash: %v", err)
	}
	if _, err := db.Exec("SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	store, _ = NewSQLiteRecoveryStore(reopened, bytes.Repeat([]byte{0x93}, 32))
	if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatalf("replay: %v", err)
	}
	sessions, _ = NewSQLiteSessionStore(reopened)
	for _, token := range []string{a, b} {
		if _, ok := sessions.Check(token); ok {
			t.Fatal("recovery preserved a prior identity session")
		}
	}
}

func TestRecoveryConcurrentIndependentHandlesHaveOneWinner(t *testing.T) {
	path, db, store := recoveryFixture(t)
	ctx := context.Background()
	codes, err := store.IssueTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	otherDB, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer otherDB.Close()
	other, _ := NewSQLiteRecoveryStore(otherDB, bytes.Repeat([]byte{0x93}, 32))
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, candidate := range []*SQLiteRecoveryStore{store, other} {
		wg.Add(1)
		go func(s *SQLiteRecoveryStore) {
			defer wg.Done()
			results <- s.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0])
		}(candidate)
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrRecoveryDenied) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("winners=%d", winners)
	}
	var audits int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.recovery_consumed'").Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("successful audits=%d err=%v", audits, err)
	}
}

func TestRecoveryAuditFailureRollsBackCodeAndRevocation(t *testing.T) {
	_, db, store := recoveryFixture(t)
	ctx := context.Background()
	codes, err := store.IssueTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	sessions, _ := NewSQLiteSessionStore(db)
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_recovery BEFORE INSERT ON audit_events WHEN NEW.action='auth.recovery_consumed' BEGIN SELECT RAISE(ABORT,'test audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); err == nil {
		t.Fatal("recovery succeeded without audit")
	}
	if _, ok := sessions.Check(token); !ok {
		t.Fatal("failed recovery revoked session")
	}
	if _, err := db.Exec("DROP TRIGGER reject_recovery"); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); err != nil {
		t.Fatalf("code was consumed by rolled-back recovery: %v", err)
	}
}

func TestRecoveryActiveStateScopeAndEnrollmentBinding(t *testing.T) {
	for _, change := range []string{
		"UPDATE identities SET disabled=1 WHERE id='user-a'",
		"UPDATE tenants SET status='suspended' WHERE id='tenant-a'",
		// Audit history intentionally restricts membership removal. Remove the
		// fixture audit explicitly to exercise the recovery FK cascade itself.
		"DELETE FROM audit_events; DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'",
		"UPDATE mfa_totp_enrollments SET status='pending',activated_at=NULL WHERE identity_id='user-a'",
		"UPDATE mfa_totp_enrollments SET secret_envelope=secret_envelope||'x' WHERE identity_id='user-a'",
	} {
		t.Run(change, func(t *testing.T) {
			_, db, store := recoveryFixture(t)
			codes, err := store.IssueTrusted(context.Background(), "tenant-a", "user-a")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(change); err != nil {
				t.Fatal(err)
			}
			if err := store.ConsumeTrusted(context.Background(), "tenant-a", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
				t.Fatalf("ineligible account accepted: %v", err)
			}
		})
	}
	_, _, store := recoveryFixture(t)
	codes, _ := store.IssueTrusted(context.Background(), "tenant-a", "user-a")
	if err := store.ConsumeTrusted(context.Background(), "tenant-b", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatal("cross-tenant recovery accepted")
	}
}

func TestRecoveryDurableAdmissionRotationAndWrongPepper(t *testing.T) {
	_, db, store := recoveryFixture(t)
	ctx := context.Background()
	now := time.Unix(1900000000, 0)
	store.now = func() time.Time { return now }
	codes, err := store.IssueTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	wrong, _ := NewSQLiteRecoveryStore(db, bytes.Repeat([]byte{0x42}, 32))
	wrong.now = store.now
	if err := wrong.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatal("wrong pepper accepted")
	}
	for i := 0; i < 4; i++ {
		if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", strings.Repeat("z", 64)); !errors.Is(err, ErrRecoveryDenied) {
			t.Fatal("invalid code accepted")
		}
	}
	restarted, _ := NewSQLiteRecoveryStore(db, bytes.Repeat([]byte{0x93}, 32))
	restarted.now = store.now
	if err := restarted.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatal("attempt budget reset on service restart")
	}
	now = now.Add(-time.Second)
	if err := restarted.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatal("clock rollback accepted")
	}
	now = now.Add(16 * time.Minute)
	if err := restarted.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[0]); err != nil {
		t.Fatal(err)
	}
	newCodes, err := store.IssueTrusted(ctx, "tenant-a", "user-a")
	if err != nil || newCodes[0] == codes[0] {
		t.Fatalf("replacement: %v", err)
	}
	if err := store.ConsumeTrusted(ctx, "tenant-a", "user-a", codes[1]); !errors.Is(err, ErrRecoveryDenied) {
		t.Fatal("replacement preserved old recovery set")
	}
}
