package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func TestTOTPReplayGuardRejectsReplaysAndAdjacentWindowReuse(t *testing.T) {
	_, db, _ := openSessionDB(t)
	guard, err := NewTOTPReplayGuard(db)
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("12345678901234567890")
	at := time.Unix(1_800_000_000, 0)
	code, err := TOTPCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}
	one, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, code, at)
	if err != nil || !one {
		t.Fatalf("valid first TOTP not consumed: ok=%v err=%v", one, err)
	}
	two, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, code, at)
	if err != nil || two {
		t.Fatalf("TOTP replay accepted: ok=%v err=%v", two, err)
	}
	// Old code remains within +/-1 window but can NEVER be reused.
	replay, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, code, at.Add(30*time.Second))
	if err != nil || replay {
		t.Fatalf("accepted older step after time tick: ok=%v err=%v", replay, err)
	}
	next, _ := TOTPCode(secret, at.Add(30*time.Second))
	ok, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, next, at.Add(30*time.Second))
	if err != nil || !ok {
		t.Fatalf("new step refused: ok=%v err=%v", ok, err)
	}
	var count, lastStep int64
	if err := db.QueryRow("SELECT count(*),max(last_accepted_step) FROM totp_replay_state WHERE identity_id='user-a'").Scan(&count, &lastStep); err != nil {
		t.Fatal(err)
	}
	if count != 1 || lastStep != at.Unix()/30+1 {
		t.Fatalf("replay ledger wrong: count=%d step=%d", count, lastStep)
	}
}

func TestTOTPReplayGuardIsIdentityWideAndRevalidatesAuthorization(t *testing.T) {
	_, db, _ := openSessionDB(t)
	if _, err := db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-b','user-a','customer')"); err != nil {
		t.Fatal(err)
	}
	guard, _ := NewTOTPReplayGuard(db)
	secret := []byte("12345678901234567890")
	at := time.Unix(1_800_000_000, 0)
	code, _ := TOTPCode(secret, at)
	if ok, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, code, at); !ok || err != nil {
		t.Fatalf("initial TOTP: %v %v", ok, err)
	}
	if ok, err := guard.VerifyAndConsume(context.Background(), "tenant-b", "user-a", secret, code, at); ok || err != nil {
		t.Fatalf("cross-tenant replay: %v %v", ok, err)
	}
	next := at.Add(30 * time.Second)
	newCode, _ := TOTPCode(secret, next)
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := guard.VerifyAndConsume(context.Background(), "tenant-b", "user-a", secret, newCode, next); ok || err != nil {
		t.Fatalf("disabled identity consumed code: %v %v", ok, err)
	}
	if _, err := db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'; UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, newCode, next); ok || err != nil {
		t.Fatalf("suspended tenant consumed code: %v %v", ok, err)
	}
	if _, err := db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-b' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := guard.VerifyAndConsume(context.Background(), "tenant-b", "user-a", secret, newCode, next); ok || err != nil {
		t.Fatalf("missing membership consumed code: %v %v", ok, err)
	}
}

func TestTOTPReplayGuardSerializesAcrossDatabaseHandles(t *testing.T) {
	path, db, _ := openSessionDB(t)
	otherDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer otherDB.Close()
	one, _ := NewTOTPReplayGuard(db)
	two, _ := NewTOTPReplayGuard(otherDB)
	secret := []byte("12345678901234567890")
	at := time.Unix(1_800_000_000, 0)
	code, _ := TOTPCode(secret, at)
	const attempts = 10
	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, attempts)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		guard := one
		if i%2 != 0 {
			guard = two
		}
		go func(g *TOTPReplayGuard) {
			defer wg.Done()
			<-start
			ok, err := g.VerifyAndConsume(context.Background(), "tenant-a", "user-a", secret, code, at)
			results <- result{ok, err}
		}(guard)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for r := range results {
		if r.err != nil {
			t.Fatalf("unexpected contention error: %v", r.err)
		}
		if r.ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("multiple concurrent consumers won: %d of %d", winners, attempts)
	}
}

func TestTOTPReplayGuardInvalidCodeDoesNotWriteState(t *testing.T) {
	_, db, _ := openSessionDB(t)
	guard, _ := NewTOTPReplayGuard(db)
	secret := []byte("12345678901234567890")
	at := time.Unix(1_800_000_000, 0)
	for _, tc := range []struct {
		sec  []byte
		code string
	}{
		{nil, "123456"}, {secret, "short"}, {secret, "12a456"},
		{secret, "000000"}, {secret, ""},
	} {
		ok, err := guard.VerifyAndConsume(context.Background(), "tenant-a", "user-a", tc.sec, tc.code, at)
		if err != nil || ok {
			t.Fatalf("invalid TOTP consumed: %#v, ok=%v err=%v", tc, ok, err)
		}
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid codes wrote %d replay rows", count)
	}
	if _, err := NewTOTPReplayGuard(nil); err == nil {
		t.Fatal("nil DB guard accepted")
	}
}
