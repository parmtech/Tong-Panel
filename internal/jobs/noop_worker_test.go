package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestDevelopmentNoopWorkerPersistsLifecycleAndAudit(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-lifecycle-0001")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.processDevelopmentNoop(); err != nil {
		t.Fatal(err)
	}
	otherDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer otherDB.Close()
	other, _ := NewSQLiteStore(otherDB)
	current, err := other.Get("tenant-a", job.ID)
	if err != nil || current.State != Done {
		t.Fatalf("persisted lifecycle: state=%s error=%v", current.State, err)
	}
	var attempts, completed int
	if err := otherDB.QueryRow("SELECT count(*),sum(status='completed') FROM job_attempts WHERE job_id=?", job.ID).Scan(&attempts, &completed); err != nil || attempts != 1 || completed != 1 {
		t.Fatal("no single completed attempt")
	}
	for _, action := range []string{"job.leased", "job.started", "job.completed"} {
		checkActor(t, other, job.ID, action, "worker", developmentNoopWorkerID, "")
		var count int
		if err := otherDB.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action=?", job.ID, action).Scan(&count); err != nil || count != 1 {
			t.Fatal("worker event is not exactly once")
		}
	}
	if err := other.processDevelopmentNoop(); !errors.Is(err, ErrNoJobs) {
		t.Fatal("completed job was eligible again")
	}
}

func TestDevelopmentNoopWorkerSkipsRoleIneligibleJobs(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	blocked, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-blocked-0001")
	if err != nil {
		t.Fatal(err)
	}
	setRequesterRole(t, store, rbac.Auditor)
	eligible, err := store.CreateFor("tenant-b", "user-b", KindNoop, "worker-eligible-0001")
	if err != nil {
		t.Fatal(err)
	}
	before := queueRoleSnapshotForJob(t, store, blocked.ID)
	if err := store.processDevelopmentNoop(); err != nil {
		t.Fatal(err)
	}
	current, err := store.Get("tenant-b", eligible.ID)
	if err != nil || current.State != Done {
		t.Fatal("eligible tenant job did not complete")
	}
	if !equalWorkerSnapshot(before, queueRoleSnapshotForJob(t, store, blocked.ID)) {
		t.Fatal("ineligible job changed")
	}
}

func equalWorkerSnapshot(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func TestDevelopmentNoopIndependentWorkersHaveOneCompletion(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-concurrent-0001")
	if err != nil {
		t.Fatal(err)
	}
	otherDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer otherDB.Close()
	other, _ := NewSQLiteStore(otherDB)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, worker := range []*SQLiteStore{store, other} {
		wg.Add(1)
		go func(worker *SQLiteStore) { defer wg.Done(); results <- worker.processDevelopmentNoop() }(worker)
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrNoJobs) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatal("concurrent workers did not have exactly one completion")
	}
	var count int
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.completed'", job.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate or missing completion audit")
	}
}

func TestDevelopmentNoopWorkerRecoversExpiredAttempt(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 11, 10, 1, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-expired-0001")
	if err != nil {
		t.Fatal(err)
	}
	old, err := store.ClaimNext("interrupted-worker", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(old); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if err := store.processDevelopmentNoop(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Complete(old); !errors.Is(err, ErrLeaseLost) {
		t.Fatal("stale attempt retained completion authority")
	}
	var attempts, completed, expired int
	if err := store.db.QueryRow("SELECT count(*),sum(status='completed'),sum(error_code='lease_expired') FROM job_attempts WHERE job_id=?", job.ID).Scan(&attempts, &completed, &expired); err != nil || attempts != 2 || completed != 1 || expired != 1 {
		t.Fatal("expired attempt recovery did not preserve the attempt ledger")
	}
}

func TestDevelopmentNoopWorkerAuditFailureDoesNotComplete(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-audit-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("CREATE TRIGGER reject_worker_complete BEFORE INSERT ON audit_events WHEN NEW.action='job.completed' BEGIN SELECT RAISE(ABORT,'sensitive-audit-canary'); END"); err != nil {
		t.Fatal(err)
	}
	if err := store.RunDevelopmentNoopWorker(context.Background()); !errors.Is(err, ErrDevelopmentNoopWorker) {
		t.Fatal("audit error did not terminate the worker generically")
	}
	current, err := store.Get("tenant-a", job.ID)
	if err != nil || current.State != Running {
		t.Fatal("failed completion audit committed completed state")
	}
	var status string
	if err := store.db.QueryRow("SELECT status FROM job_attempts WHERE job_id=?", job.ID).Scan(&status); err != nil || status != "started" {
		t.Fatal("failed completion audit finished the attempt")
	}
}

func TestDevelopmentNoopWorkerCancellationDoesNotPoisonAPIStore(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-cancelled-0001")
	if err != nil {
		t.Fatal(err)
	}
	before := queueRoleSnapshot(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.RunDevelopmentNoopWorker(ctx); err != nil {
		t.Fatal(err)
	}
	requireQueueRoleSnapshot(t, store, before)
	if _, err := store.CancelFor("tenant-a", "user-a", job.ID); err != nil {
		t.Fatal("worker cancellation changed the API store context")
	}
	idle, stop := context.WithCancel(context.Background())
	started := make(chan struct{}, 1)
	store.now = func() time.Time {
		select {
		case started <- struct{}{}:
		default:
		}
		return time.Now()
	}
	done := make(chan error, 1)
	go func() { done <- store.RunDevelopmentNoopWorker(idle) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		stop()
		t.Fatal("idle worker never attempted queue selection")
	}
	stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("idle worker did not stop cooperatively")
	}
}

func TestDevelopmentNoopWorkerRefusesFutureKind(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "worker-kind-0001")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a future schema supporting more kinds; never execute the payload.
	if _, err := store.db.Exec("PRAGMA ignore_check_constraints=ON; UPDATE jobs SET kind='future.host.mutation' WHERE id=?", job.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.RunDevelopmentNoopWorker(context.Background()); !errors.Is(err, ErrDevelopmentNoopWorker) {
		t.Fatal("unknown kind was not refused")
	}
	current, err := store.Get("tenant-a", job.ID)
	if err != nil || current.State != Leased {
		t.Fatal("unsupported kind started or completed")
	}
}
