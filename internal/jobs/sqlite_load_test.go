package jobs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

// This bounded stress test drains metadata-only work through two independent
// SQLite pools. It does not execute host effects or model power loss.
func TestSQLiteWorkersDrainQueueAcrossHandles(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	peerDB, err := database.Open(context.Background(), filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	peer, err := NewSQLiteStore(peerDB)
	if err != nil {
		t.Fatal(err)
	}

	const totalJobs = 128
	for i := 0; i < totalJobs; i++ {
		if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, fmt.Sprintf("drain-load-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store.ctx, peer.ctx = ctx, ctx
	const workers = 8
	start := make(chan struct{})
	completed := make(chan string, totalJobs)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		queue := store
		if i%2 != 0 {
			queue = peer
		}
		wg.Add(1)
		go func(index int, queue *SQLiteStore) {
			defer wg.Done()
			<-start
			workerID := fmt.Sprintf("load-worker-%02d", index)
			for {
				if err := ctx.Err(); err != nil {
					errs <- fmt.Errorf("worker %s deadline: %w", workerID, err)
					return
				}
				claim, err := queue.ClaimNext(workerID, time.Minute)
				if errors.Is(err, ErrNoJobs) {
					return
				}
				if err != nil {
					errs <- fmt.Errorf("worker %s claim: %w", workerID, err)
					return
				}
				if _, err = queue.Start(claim); err != nil {
					errs <- fmt.Errorf("worker %s start: %w", workerID, err)
					return
				}
				job, err := queue.Complete(claim)
				if err != nil {
					errs <- fmt.Errorf("worker %s complete: %w", workerID, err)
					return
				}
				select {
				case completed <- job.ID:
				case <-ctx.Done():
					errs <- fmt.Errorf("worker %s report completion: %w", workerID, ctx.Err())
					return
				}
			}
		}(i, queue)
	}
	close(start)
	wg.Wait()
	close(completed)
	close(errs)
	for err := range errs {
		t.Fatalf("worker queue drain failed: %v", err)
	}

	seen := make(map[string]bool, totalJobs)
	for id := range completed {
		if seen[id] {
			t.Fatalf("job completed more than once: %s", id)
		}
		seen[id] = true
	}
	if len(seen) != totalJobs {
		t.Fatalf("completed %d unique jobs, want %d", len(seen), totalJobs)
	}
	var completedJobs, attempts, completedAttempts, leaseAudits, startAudits, completeAudits int
	err = store.db.QueryRow(`SELECT
		(SELECT count(*) FROM jobs WHERE tenant_id='tenant-a' AND state='completed'),
		(SELECT count(*) FROM job_attempts a JOIN jobs j ON j.id=a.job_id WHERE j.tenant_id='tenant-a'),
		(SELECT count(*) FROM job_attempts a JOIN jobs j ON j.id=a.job_id WHERE j.tenant_id='tenant-a' AND a.status='completed'),
		(SELECT count(*) FROM audit_events WHERE tenant_id='tenant-a' AND action='job.leased'),
		(SELECT count(*) FROM audit_events WHERE tenant_id='tenant-a' AND action='job.started'),
		(SELECT count(*) FROM audit_events WHERE tenant_id='tenant-a' AND action='job.completed')`).
		Scan(&completedJobs, &attempts, &completedAttempts, &leaseAudits, &startAudits, &completeAudits)
	if err != nil {
		t.Fatal(err)
	}
	for label, count := range map[string]int{
		"completed jobs": completedJobs, "attempts": attempts, "completed attempts": completedAttempts,
		"lease audits": leaseAudits, "start audits": startAudits, "completion audits": completeAudits,
	} {
		if count != totalJobs {
			t.Errorf("%s=%d, want %d", label, count, totalJobs)
		}
	}
}

func TestSQLiteClaimRetriesTransientBeginBusyAcrossHandles(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	peerDB, err := database.Open(context.Background(), filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	peerConn, err := peerDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peerConn.Close()
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "begin-busy-retry"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := peerConn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer peerConn.ExecContext(context.Background(), "ROLLBACK")
	type claimResult struct {
		claim Claim
		err   error
	}
	result := make(chan claimResult, 1)
	go func() {
		claim, err := store.ClaimNext("busy-retry-worker", time.Minute)
		result <- claimResult{claim: claim, err: err}
	}()
	time.Sleep(100 * time.Millisecond)
	if _, err := peerConn.ExecContext(context.Background(), "COMMIT"); err != nil {
		t.Fatal(err)
	}
	select {
	case outcome := <-result:
		if outcome.err != nil {
			t.Fatalf("claim did not recover after writer released: %v", outcome.err)
		}
		if outcome.claim.Job.IdempotencyKey != "begin-busy-retry" {
			t.Fatalf("claimed unexpected job %q", outcome.claim.Job.IdempotencyKey)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("claim did not finish after writer lock was released")
	}
}

func TestSQLiteClaimBeginBusyRetryHonorsContextDeadline(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	peerDB, err := database.Open(context.Background(), filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	peerConn, err := peerDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peerConn.Close()
	if _, err := peerConn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer peerConn.ExecContext(context.Background(), "ROLLBACK")
	if _, err := store.db.Exec("PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	store.ctx = ctx
	started := time.Now()
	_, err = store.ClaimNext("busy-timeout-worker", time.Minute)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("claim under a held writer lock returned %v, want deadline", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("claim exceeded its bounded context: %s", elapsed)
	}
}

func TestSQLiteClaimLeaseStartsAfterConnectionWait(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	start := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	var current atomic.Int64
	current.Store(start.Unix())
	store.now = func() time.Time { return time.Unix(current.Load(), 0).UTC() }
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "lease-after-wait")
	if err != nil {
		t.Fatal(err)
	}
	store.db.SetMaxOpenConns(1)
	blocker, err := store.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	waitsBefore := store.db.Stats().WaitCount
	type outcome struct {
		claim Claim
		err   error
	}
	result := make(chan outcome, 1)
	go func() {
		claim, claimErr := store.ClaimNext("lease-wait-worker", time.Minute)
		result <- outcome{claim: claim, err: claimErr}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for store.db.Stats().WaitCount == waitsBefore && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if store.db.Stats().WaitCount == waitsBefore {
		t.Fatal("claim did not wait for the held database connection")
	}
	late := start.Add(2 * time.Minute)
	current.Store(late.Unix())
	if err := blocker.Close(); err != nil {
		t.Fatal(err)
	}
	var got outcome
	select {
	case got = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("claim did not finish after releasing the connection")
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	wantExpiry := late.Add(time.Minute)
	if got.claim.Job.ID != job.ID || !got.claim.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("claim job=%s expiry=%s, want job=%s expiry=%s", got.claim.Job.ID, got.claim.ExpiresAt, job.ID, wantExpiry)
	}
	var storedExpiry string
	if err := store.db.QueryRow("SELECT lease_until FROM jobs WHERE id=?", job.ID).Scan(&storedExpiry); err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, storedExpiry)
	if err != nil || !parsed.Equal(wantExpiry) {
		t.Fatalf("stored lease expiry=%q err=%v, want %s", storedExpiry, err, wantExpiry)
	}
}

func TestSQLiteMutationsHonorCallerCancellationDuringConnectionWait(t *testing.T) {
	for _, operation := range []string{"create", "cancel"} {
		t.Run(operation, func(t *testing.T) {
			store, _ := openTestSQLiteStore(t)
			var job Job
			var err error
			if operation == "cancel" {
				job, err = store.CreateFor("tenant-a", "user-a", KindNoop, "cancel-context-job")
				if err != nil {
					t.Fatal(err)
				}
			}
			store.db.SetMaxOpenConns(1)
			blocker, err := store.db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			waitsBefore := store.db.Stats().WaitCount
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				if operation == "create" {
					_, callErr := store.CreateForContext(ctx, "tenant-a", "user-a", KindNoop, "create-context-job")
					result <- callErr
					return
				}
				_, callErr := store.CancelForContext(ctx, "tenant-a", "user-a", job.ID)
				result <- callErr
			}()
			deadline := time.Now().Add(3 * time.Second)
			for store.db.Stats().WaitCount == waitsBefore && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if store.db.Stats().WaitCount == waitsBefore {
				t.Fatal("mutation did not wait for the held pooled connection")
			}
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("mutation error=%v, want context cancellation", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("mutation did not stop after caller cancellation")
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			if operation == "cancel" {
				var state string
				if err := store.db.QueryRow("SELECT state FROM jobs WHERE id=?", job.ID).Scan(&state); err != nil || state != Queued {
					t.Fatalf("canceled database wait changed job state=%q err=%v", state, err)
				}
			}
		})
	}
}

func TestSQLiteCancelPreservesCancellationDuringAuthorization(t *testing.T) {
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(want.Error(), func(t *testing.T) {
			if got := cancelAuthorizationError(want); !errors.Is(got, want) {
				t.Fatalf("authorization error=%v, want preserved %v", got, want)
			}
		})
	}
	if got := cancelAuthorizationError(ErrRequesterDenied); !errors.Is(got, ErrNotFound) {
		t.Fatalf("authorization error=%v, want concealed not-found", got)
	}
}

func TestSQLiteActorListPreservesCancellationDuringAuthorization(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		var ctx context.Context
		var cancel context.CancelFunc
		if errors.Is(want, context.Canceled) {
			ctx, cancel = context.WithCancel(context.Background())
			cancel()
		} else {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}
		_, err := store.ListForContext(ctx, "tenant-a", "user-a")
		cancel()
		if !errors.Is(err, want) {
			t.Fatalf("list error=%v, want %v", err, want)
		}
	}
	_, err := store.ListForContext(context.Background(), "tenant-a", "missing-user")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unauthorized list error=%v, want concealed not-found", err)
	}
}

func TestSQLiteActorReadsHonorCancellationDuringConnectionWait(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "actor-read-wait")
	if err != nil {
		t.Fatal(err)
	}
	store.db.SetMaxOpenConns(1)
	blocker, err := store.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	for _, operation := range []string{"list", "detail"} {
		t.Run(operation, func(t *testing.T) {
			waitsBefore := store.db.Stats().WaitCount
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan error, 1)
			go func() {
				if operation == "list" {
					_, callErr := store.ListForContext(ctx, "tenant-a", "user-a")
					result <- callErr
					return
				}
				_, callErr := store.GetForContext(ctx, "tenant-a", "user-a", job.ID)
				result <- callErr
			}()
			deadline := time.Now().Add(3 * time.Second)
			for store.db.Stats().WaitCount == waitsBefore && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if store.db.Stats().WaitCount == waitsBefore {
				cancel()
				t.Fatal("job read did not wait for held pooled connection")
			}
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("job read error=%v, want cancellation", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("job read did not stop after context cancellation")
			}
		})
	}
}
