package jobs

import (
	"errors"
	"testing"
	"time"
)

func TestSQLiteExhaustedLeaseDoesNotStarveNextReadyJob(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	s.maxAttempts = 1

	exhausted, err := s.CreateFor("tenant-a", "user-a", KindNoop, "exhaust-0001")
	if err != nil {
		t.Fatal(err)
	}
	firstLease, err := s.ClaimNext("worker-first", time.Minute)
	if err != nil || firstLease.Job.ID != exhausted.ID {
		t.Fatalf("first claim job=%+v error=%v", firstLease.Job, err)
	}
	now = now.Add(61 * time.Second)

	ready, err := s.CreateFor("tenant-a", "user-a", KindNoop, "ready-0002")
	if err != nil {
		t.Fatal(err)
	}
	nextLease, err := s.ClaimNext("worker-next", time.Minute)
	if err != nil || nextLease.Job.ID != ready.ID {
		t.Fatalf("exhausted older lease starved ready job: claim=%+v error=%v", nextLease.Job, err)
	}
	// Exhausted jobs still need to be reconciled. When no runnable work remains,
	// the oldest stale exhausted lease is recorded failed, not re-leased.
	if _, err := s.ClaimNext("worker-reconciler", time.Minute); !errors.Is(err, ErrNoJobs) {
		t.Fatalf("expected cleanup without new claim: %v", err)
	}
	old, err := s.Get("tenant-a", exhausted.ID)
	if err != nil || old.State != Failed {
		t.Fatalf("exhausted job not finalized: %+v error=%v", old, err)
	}
	checkActor(t, s, exhausted.ID, "job.failed", "system", "lease-reconciler", "")
	if _, err := s.Start(firstLease); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale worker capability still valid: %v", err)
	}
	if _, err := s.Start(nextLease); err != nil {
		t.Fatalf("eligible job lost valid lease: %v", err)
	}
}
