package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestSQLiteSessionIdleExpiryAndAbsoluteLimit(t *testing.T) {
	_, _, store := openSessionDB(t)
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	p := rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner}
	idleToken, _, absolute, err := store.Issue(p, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(20 * time.Minute)
	if _, ok := store.Check(idleToken); !ok {
		t.Fatal("legitimate use before idle cutoff was refused")
	}
	now = now.Add(29 * time.Minute)
	if _, ok := store.Check(idleToken); !ok {
		t.Fatal("monotonically refreshed idle deadline not honored")
	}
	now = now.Add(31 * time.Minute)
	if _, ok := store.Check(idleToken); ok {
		t.Fatal("idle session survived over 30 minutes of inactivity")
	}
	now = now.Add(-10 * time.Minute)
	if _, ok := store.Check(idleToken); ok {
		t.Fatal("already-idle-expired session was reactivated by a clock rollback")
	}
	// An active session may be refreshed, but its independent absolute
	// lifetime always wins. A separate test avoids the idle-expired token.
	now = absolute.Add(-time.Minute)
	fresh, _, exp, err := store.Issue(p, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = exp
	if _, ok := store.Check(fresh); ok {
		t.Fatal("absolute expiry accepted at boundary")
	}
}

func TestSQLiteSessionRotateRevokesOldAndPreservesAbsoluteExpiry(t *testing.T) {
	_, db, store := openSessionDB(t)
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	p := rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner}
	old, oldCSRF, absolute, err := store.Issue(p, 90*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Minute)
	next, csrf, exp, err := store.Rotate(context.Background(), old)
	if err != nil || exp != absolute || next == old || csrf == oldCSRF {
		t.Fatalf("rotation failed or prolonged absolute deadline: %v", err)
	}
	if _, ok := store.Check(old); ok || store.CheckCSRF(old, oldCSRF) {
		t.Fatal("old bearer was still accepted after rotation")
	}
	if _, ok := store.Check(next); !ok || !store.CheckCSRF(next, csrf) {
		t.Fatal("rotated bearer and CSRF not accepted")
	}
	if _, _, _, err := store.Rotate(context.Background(), old); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("revoked bearer rotated twice: %v", err)
	}
	now = now.Add(10 * time.Minute)
	third, thirdCSRF, thirdExpiry, err := store.Rotate(context.Background(), next)
	if err != nil || thirdExpiry != absolute {
		t.Fatalf("second rotation extended expiry: %s %v", thirdExpiry, err)
	}
	if _, ok := store.Check(next); ok {
		t.Fatal("second rotation did not revoke previous bearer")
	}
	if _, ok := store.Check(third); !ok || !store.CheckCSRF(third, thirdCSRF) {
		t.Fatal("second rotation session invalid")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.session_rotated'").Scan(&count); err != nil || count != 2 {
		t.Fatalf("rotations were not audited: %d %v", count, err)
	}
	now = absolute
	if _, ok := store.Check(third); ok {
		t.Fatal("rotation extended absolute lifetime")
	}
}

func TestSQLiteSessionLogoutAuditsAndRollsBackOnAuditFailure(t *testing.T) {
	_, db, store := openSessionDB(t)
	token, csrf, _, err := store.Issue(
		rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER block_logout BEFORE INSERT ON audit_events "+
		"WHEN NEW.action='auth.logout' BEGIN SELECT RAISE(ABORT,'audit failed'); END"); err != nil {
		t.Fatal(err)
	}
	if err := store.Logout(context.Background(), token); err == nil {
		t.Fatal("logout committed without mandatory audit")
	}
	if _, ok := store.Check(token); !ok {
		t.Fatal("audit failure incorrectly revoked the user session")
	}
	if _, err := db.Exec("DROP TRIGGER block_logout"); err != nil {
		t.Fatal(err)
	}
	if err := store.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Check(token); ok || store.CheckCSRF(token, csrf) {
		t.Fatal("logged-out session remained authorized")
	}
	if err := store.Logout(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("second logout was accepted: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.logout'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("logout audit count=%d err=%v", count, err)
	}
}

func TestSQLiteSessionConcurrentReadsDoNotInvalidateEachOther(t *testing.T) {
	_, db, store := openSessionDB(t)
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	token, _, _, err := store.Issue(
		rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(20 * time.Minute)
	const readers = 16
	var wg sync.WaitGroup
	failures := make(chan struct{}, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := store.Check(token); !ok {
				failures <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(failures)
	if len(failures) != 0 {
		t.Fatalf("%d concurrent valid session readers were denied", len(failures))
	}
	var deadline int64
	if err := db.QueryRow("SELECT idle_deadline_unix FROM sessions WHERE revoked_at IS NULL").
		Scan(&deadline); err != nil || deadline != now.Add(sqliteSessionIdle).Unix() {
		t.Fatalf("idle deadline was not monotonic: %d %v", deadline, err)
	}
}

func TestSQLiteSessionRotationHasOnlyOneWinner(t *testing.T) {
	_, db, store := openSessionDB(t)
	token, _, _, err := store.Issue(
		rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	const count = 4
	results := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, err := store.Rotate(context.Background(), token)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("unexpected rotate error: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("session rotation race produced %d winners", winners)
	}
	var open int
	if err := db.QueryRow("SELECT count(*) FROM sessions WHERE revoked_at IS NULL").Scan(&open); err != nil || open != 1 {
		t.Fatalf("unexpected active rotated sessions: %d %v", open, err)
	}
}

func TestSQLiteSessionMigrationV7RejectsMissingIdleDeadline(t *testing.T) {
	_, db, store := openSessionDB(t)
	_, err := db.Exec(`INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at)
		VALUES('legacy','tenant-a','user-a',zeroblob(32),zeroblob(32),'2099-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	var deadline int64
	if err := db.QueryRow("SELECT idle_deadline_unix FROM sessions WHERE id='legacy'").Scan(&deadline); err != nil || deadline != 0 {
		t.Fatalf("legacy session row accidentally received active idle deadline: %d %v", deadline, err)
	}
	if _, _, _, err := store.Rotate(context.Background(), "not-a-token"); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("unparseable session rotated: %v", err)
	}
}

