package jobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func openTestSQLiteStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B');
		INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner');`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestSQLiteCreateIdempotencyScopeAndTransactions(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	created, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0001")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0001")
	if err != nil || retry.ID != created.ID {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", "different.work", "request-0001"); err == nil {
		t.Fatal("unsupported work kind accepted")
	}
	if _, err := store.CreateFor("tenant-a", "user-b", KindNoop, "request-0001"); err != ErrConflict {
		t.Fatalf("different requester should conflict: %v", err)
	}
	now = now.Add(time.Second)
	if _, err := store.CreateFor("tenant-b", "user-b", KindNoop, "request-0001"); err != nil {
		t.Fatalf("idempotency key should be tenant-scoped: %v", err)
	}
	if _, err := store.CreateFor("tenant-a", "missing-user", KindNoop, "request-0002"); err == nil {
		t.Fatal("missing membership accepted")
	}
	if _, err := store.db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0003"); err == nil {
		t.Fatal("job created for suspended tenant")
	}
	if _, err := store.db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0004"); err == nil {
		t.Fatal("job created by disabled identity")
	}
	if _, err := store.db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow("SELECT count(*) FROM jobs WHERE tenant_id='tenant-a'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed insertion left a row: count=%d err=%v", count, err)
	}
	if _, err := store.Get("tenant-b", created.ID); err != ErrNotFound {
		t.Fatalf("cross-tenant lookup was not hidden: %v", err)
	}
	other, err := store.List("tenant-b")
	if err != nil || len(other) != 1 {
		t.Fatalf("tenant-b list=%v err=%v", other, err)
	}
	missing, err := store.List("missing")
	if err != nil || len(missing) != 0 {
		t.Fatal("tenant-filtered list mismatch")
	}
	claim, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil || claim.Job.ID != created.ID {
		t.Fatalf("claim=%+v err=%v", claim, err)
	}
	started, err := store.Start(claim)
	if err != nil || started.State != Running {
		t.Fatal(err)
	}
	completed, err := store.Complete(claim)
	if err != nil || completed.State != Done {
		t.Fatal(err)
	}
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, _ := NewSQLiteStore(db)
	got, err := reopened.Get("tenant-a", created.ID)
	if err != nil || got.State != Done {
		t.Fatalf("restart lost committed state: %+v err=%v", got, err)
	}
}

func TestSQLiteConcurrentIdempotentCreate(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	const workers = 24
	var wg sync.WaitGroup
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-race-1")
			if err != nil {
				errs <- err
				return
			}
			ids <- j.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	want := ""
	for id := range ids {
		if want == "" {
			want = id
		}
		if id != want {
			t.Fatalf("concurrent retries produced multiple jobs: %s != %s", id, want)
		}
	}
	list, err := store.List("tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(list); got != 1 {
		t.Fatalf("stored %d jobs, want exactly one", got)
	}
}

func TestSQLiteLeaseFenceRetryRecoveryAndAudit(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "lease-job-0001")
	if err != nil {
		t.Fatal(err)
	}
	var auditCount int
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.created'", job.ID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("create audit count=%d err=%v", auditCount, err)
	}
	first, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil || first.Job.ID != job.ID || first.Token == "" {
		t.Fatalf("claim=%+v err=%v", first, err)
	}
	decoded, err := hex.DecodeString(first.Token)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256(decoded)
	var storedHash []byte
	if err := store.db.QueryRow("SELECT lease_token_sha256 FROM jobs WHERE id=?", job.ID).Scan(&storedHash); err != nil || !bytes.Equal(storedHash, wantHash[:]) {
		t.Fatalf("lease token hash mismatch: %x err=%v", storedHash, err)
	}
	if _, err := store.Start(first); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimNext("worker-b", time.Minute); err != ErrNoJobs {
		t.Fatalf("active lease was claimed twice: %v", err)
	}
	forged := first
	forged.Token = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	if _, err := store.Complete(forged); err != ErrLeaseLost {
		t.Fatalf("forged lease token accepted: %v", err)
	}
	completed, err := store.Complete(first)
	if err != nil || completed.State != Done {
		t.Fatalf("complete=%+v err=%v", completed, err)
	}
	if _, err := store.Complete(first); err != ErrLeaseLost {
		t.Fatalf("completed lease reused: %v", err)
	}
	var attempts, auditEvents int
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=? AND status='completed'", job.ID).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("completed attempt count=%d err=%v", attempts, err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=?", job.ID).Scan(&auditEvents); err != nil || auditEvents != 4 {
		t.Fatalf("audit count=%d err=%v; want create, lease, start, complete", auditEvents, err)
	}
}

func TestSQLiteLeaseRenewalExtendsOnlyCurrentCapability(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "renew-job-0001")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil || claim.Job.ID != job.ID {
		t.Fatalf("claim=%+v err=%v", claim, err)
	}
	if _, err := store.Renew(claim, maxLeaseDuration+time.Second); err == nil {
		t.Fatal("oversized lease renewal accepted")
	}
	now = now.Add(59 * time.Second)
	renewed, err := store.Renew(claim, time.Minute)
	if err != nil || !renewed.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("renewed lease=%+v err=%v", renewed, err)
	}
	// The original expiry has passed, but the renewed database lease remains
	// usable with the same unexpired capability.
	now = now.Add(2 * time.Second)
	if _, err := store.Start(claim); err != nil {
		t.Fatalf("renewal did not extend the lease: %v", err)
	}
	if _, err := store.Complete(renewed); err != nil {
		t.Fatal(err)
	}
	var renewEvents int
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.lease_renewed'", job.ID).Scan(&renewEvents); err != nil || renewEvents != 1 {
		t.Fatalf("renewal audit count=%d err=%v", renewEvents, err)
	}
}

func TestSQLiteLeaseExpiryReclaimsAndFencesStaleWorker(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "expire-job-0001")
	if err != nil {
		t.Fatal(err)
	}
	oldClaim, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute + time.Second)
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err = NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	newClaim, err := store.ClaimNext("worker-b", time.Minute)
	if err != nil || newClaim.Job.ID != job.ID || newClaim.Token == oldClaim.Token {
		t.Fatalf("reclaim=%+v err=%v", newClaim, err)
	}
	if _, err := store.Start(oldClaim); err != ErrLeaseLost {
		t.Fatalf("stale lease started after recovery: %v", err)
	}
	if _, err := store.Start(newClaim); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Complete(newClaim); err != nil {
		t.Fatal(err)
	}
	var attempts, expired int
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=?", job.ID).Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("attempt count=%d err=%v", attempts, err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=? AND error_code='lease_expired'", job.ID).Scan(&expired); err != nil || expired != 1 {
		t.Fatalf("expired attempts=%d err=%v", expired, err)
	}
}

func TestSQLiteLeaseRetryDelayAndMaximumAttempts(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	store.maxAttempts = 2
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "retry-job-0001")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(first); err != nil {
		t.Fatal(err)
	}
	retried, err := store.Fail(first, "temporary_failure", true, 30*time.Second)
	if err != nil || retried.State != Queued {
		t.Fatalf("retry=%+v err=%v", retried, err)
	}
	if _, err := store.ClaimNext("worker-b", time.Minute); err != ErrNoJobs {
		t.Fatalf("retry delay ignored: %v", err)
	}
	now = now.Add(30 * time.Second)
	second, err := store.ClaimNext("worker-b", time.Minute)
	if err != nil || second.Job.ID != job.ID {
		t.Fatalf("second claim=%+v err=%v", second, err)
	}
	if _, err := store.Start(second); err != nil {
		t.Fatal(err)
	}
	failed, err := store.Fail(second, "permanent_failure", true, 0)
	if err != nil || failed.State != Failed {
		t.Fatalf("maximum attempts not terminal: job=%+v err=%v", failed, err)
	}
	if _, err := store.Fail(second, "message contains a secret", false, 0); err == nil {
		t.Fatal("unsafe failure details accepted")
	}
	var attempts int
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=?", job.ID).Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("attempt count=%d err=%v", attempts, err)
	}
}

func TestSQLiteClaimRechecksTenantAndIdentityStatus(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "actor-check-0001")
	if err != nil {
		t.Fatal(err)
	}
	for _, update := range []string{
		"UPDATE tenants SET status='suspended' WHERE id='tenant-a'",
		"UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='user-a'",
	} {
		if _, err := store.db.Exec(update); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ClaimNext("worker-a", time.Minute); err != ErrNoJobs {
			t.Fatalf("ineligible actor's job was claimable: %v", err)
		}
	}
	if _, err := store.db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	claim, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil || claim.Job.ID != job.ID {
		t.Fatalf("job did not become claimable after tenant and identity reactivation: claim=%+v err=%v", claim, err)
	}
}

func TestSQLiteClaimRollsBackWhenAuditWriteFails(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "audit-rollback-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("DROP TABLE audit_events"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimNext("worker-a", time.Minute); err == nil {
		t.Fatal("claim succeeded without its audit record")
	}
	got, err := store.Get("tenant-a", job.ID)
	if err != nil || got.State != Queued {
		t.Fatalf("job claim was not rolled back with audit failure: state=%s err=%v", got.State, err)
	}
	var attempts int
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=?", job.ID).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatalf("attempt insert was not rolled back: count=%d err=%v", attempts, err)
	}
}

func TestSQLiteLeaseLimitFailureClosesExpiredJob(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	store.maxAttempts = 1
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "lease-limit-0001")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.ClaimNext("worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute + time.Second)
	if _, err := store.ClaimNext("worker-b", time.Minute); err != ErrNoJobs {
		t.Fatalf("exhausted expired lease was claimable: %v", err)
	}
	got, err := store.Get("tenant-a", job.ID)
	if err != nil || got.State != Failed {
		t.Fatalf("exhausted job state=%q err=%v", got.State, err)
	}
	if _, err := store.Start(claim); err != ErrLeaseLost {
		t.Fatalf("expired exhausted lease still authorized worker: %v", err)
	}
	if _, err := store.db.Exec("UPDATE jobs SET state='leased' WHERE id=?", job.ID); err == nil {
		t.Fatal("schema accepted leased job without lease capability fields")
	}
}

func TestSQLiteConcurrentWorkersClaimDifferentJobs(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	peerDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	peer, err := NewSQLiteStore(peerDB)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, fmt.Sprintf("claim-race-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	const workers = 12
	var wg sync.WaitGroup
	claims := make(chan Claim, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			queue := store
			if i%2 != 0 {
				queue = peer
			}
			claim, err := queue.ClaimNext(fmt.Sprintf("worker-%02d", i), time.Minute)
			if err != nil {
				errs <- err
				return
			}
			claims <- claim
		}(i)
	}
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for claim := range claims {
		if seen[claim.Job.ID] {
			t.Fatalf("job leased twice: %s", claim.Job.ID)
		}
		seen[claim.Job.ID] = true
	}
	if len(seen) != workers {
		t.Fatalf("claimed %d unique jobs, want %d", len(seen), workers)
	}
}
