package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func TestSQLiteCancelOwnQueuedJobIsIdempotentAndAudited(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "cancel-0001")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.CancelFor("tenant-a", "user-a", job.ID)
	if err != nil || got.State != Cancelled {
		t.Fatalf("cancel: job=%+v error=%v", got, err)
	}
	again, err := s.CancelFor("tenant-a", "user-a", job.ID)
	if err != nil || again.State != Cancelled || !again.UpdatedAt.Equal(got.UpdatedAt) {
		t.Fatalf("cancel retry: job=%+v error=%v", again, err)
	}
	current, err := s.Get("tenant-a", job.ID)
	if err != nil || current.State != Cancelled {
		t.Fatalf("persisted job=%+v error=%v", current, err)
	}
	var count int
	if err := s.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.cancelled'", job.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("cancellation audit count=%d error=%v", count, err)
	}
	if _, err := s.ClaimNext("worker-a", time.Minute); !errors.Is(err, ErrNoJobs) {
		t.Fatalf("cancelled job became leaseable: %v", err)
	}
}

func TestSQLiteCancelTenantAndRequesterBoundaries(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "cancel-0002")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ tenant, actor, id string }{
		{"tenant-b", "user-b", job.ID},
		{"tenant-a", "user-b", job.ID},
		{"tenant-a", "user-a", "invalid"},
		{"tenant-a", "missing", job.ID},
	} {
		if _, err := s.CancelFor(tc.tenant, tc.actor, tc.id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unexpected cancel access for %+v: %v", tc, err)
		}
	}
	if _, err := s.db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelFor("tenant-a", "user-a", job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("suspended tenant cancellation accepted: %v", err)
	}
	if _, err := s.db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelFor("tenant-a", "user-a", job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled identity cancellation accepted: %v", err)
	}
	current, err := s.Get("tenant-a", job.ID)
	if err != nil || current.State != Queued {
		t.Fatalf("denied operations mutated job: %+v %v", current, err)
	}
}

func TestSQLiteCancelRejectsLeasedRunningAndCompleted(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "cancel-0003")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimNext("worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{Leased, Running, Done} {
		if _, err := s.CancelFor("tenant-a", "user-a", job.ID); !errors.Is(err, ErrJobNotCancelable) {
			t.Fatalf("cancellation unexpectedly allowed for %s: %v", state, err)
		}
		if state == Leased {
			if _, err := s.Start(claim); err != nil {
				t.Fatal(err)
			}
		} else if state == Running {
			if _, err := s.Complete(claim); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestSQLiteCancelRollsBackIfAuditInsertFails(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "cancel-0004")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("DROP TABLE audit_events"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelFor("tenant-a", "user-a", job.ID); err == nil {
		t.Fatal("cancellation succeeded without an audit record")
	}
	current, err := s.Get("tenant-a", job.ID)
	if err != nil || current.State != Queued {
		t.Fatalf("failed audit did not rollback: job=%+v err=%v", current, err)
	}
}

func TestSQLiteCancelAndClaimCannotBothWin(t *testing.T) {
	s, path := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "cancel-0005")
	if err != nil {
		t.Fatal(err)
	}
	otherDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer otherDB.Close()
	otherStore, err := NewSQLiteStore(otherDB)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var cancelErr, claimErr error
	var claim Claim
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, cancelErr = s.CancelFor("tenant-a", "user-a", job.ID)
	}()
	go func() {
		defer wg.Done()
		<-start
		claim, claimErr = otherStore.ClaimNext("worker-race", time.Minute)
	}()
	close(start)
	wg.Wait()

	cancelWon := cancelErr == nil
	claimWon := claimErr == nil
	if cancelWon == claimWon {
		t.Fatalf("invalid race outcome cancel=%v claim=%v", cancelErr, claimErr)
	}
	if cancelWon && !errors.Is(claimErr, ErrNoJobs) {
		t.Fatalf("cancellation won but worker saw %v", claimErr)
	}
	if claimWon && (claim.Job.ID != job.ID || !errors.Is(cancelErr, ErrJobNotCancelable)) {
		t.Fatalf("claim won but cancellation outcome was %+v %v", claim, cancelErr)
	}
	current, err := s.Get("tenant-a", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if (cancelWon && current.State != Cancelled) || (claimWon && current.State != Leased) {
		t.Fatalf("invalid persisted state after contention: %q", current.State)
	}
}
