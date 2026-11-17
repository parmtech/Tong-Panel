package jobs

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func checkActor(t *testing.T, s *SQLiteStore, jobID, action, kind, actorRef, identity string) {
	t.Helper()
	var gotKind string
	var gotRef, gotIdentity sql.NullString
	err := s.db.QueryRow("SELECT actor_kind,actor_ref,actor_identity_id FROM audit_events WHERE subject_id=? AND action=? ORDER BY rowid DESC LIMIT 1", jobID, action).
		Scan(&gotKind, &gotRef, &gotIdentity)
	if err != nil {
		t.Fatalf("audit event %s: %v", action, err)
	}
	if gotKind != kind || gotRef.String != actorRef || gotIdentity.String != identity {
		t.Fatalf("audit %s actor got kind=%q ref=%q user=%q, want kind=%q ref=%q user=%q",
			action, gotKind, gotRef.String, gotIdentity.String, kind, actorRef, identity)
	}
}

func TestSQLiteWorkerEventsAreNotAttributedToRequester(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "worker-audit-0001")
	if err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.created", "user", "", "user-a")
	claim, err := s.ClaimNext("worker-123", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.leased", "worker", "worker-123", "")
	if _, err := s.Start(claim); err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.started", "worker", "worker-123", "")
	if _, err := s.Renew(claim, time.Minute); err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.lease_renewed", "worker", "worker-123", "")
	if _, err := s.Complete(claim); err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.completed", "worker", "worker-123", "")
	// The actor reference must not contain a token or hash.
	var leaked int
	if err := s.db.QueryRow("SELECT count(*) FROM audit_events WHERE actor_ref LIKE ? OR request_id LIKE ?",
		"%"+claim.Token+"%", "%"+claim.Token+"%").Scan(&leaked); err != nil || leaked != 0 {
		t.Fatalf("lease secret found in audit: leaked=%d err=%v", leaked, err)
	}
}

func TestSQLiteReconcilerEventsHaveSystemActor(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	s.maxAttempts = 1
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "audit-expired-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimNext("worker-old", time.Minute); err != nil {
		t.Fatal(err)
	}
	now = now.Add(61 * time.Second)
	if _, err := s.ClaimNext("worker-reclaimer", time.Minute); !errors.Is(err, ErrNoJobs) {
		t.Fatalf("exhausted expired lease not terminal: %v", err)
	}
	checkActor(t, s, job.ID, "job.lease_expired", "system", "lease-reconciler", "")
	checkActor(t, s, job.ID, "job.failed", "system", "lease-reconciler", "")
}

func TestSQLiteExplicitWorkerFailHasWorkerActor(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	job, err := s.CreateFor("tenant-a", "user-a", KindNoop, "audit-fail-0001")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimNext("worker-7", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(claim); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Fail(claim, "validation.failed", false, 0); err != nil {
		t.Fatal(err)
	}
	checkActor(t, s, job.ID, "job.failed", "worker", "worker-7", "")
}

func TestSQLiteRejectsWorkerIDAuditInjection(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	if _, err := s.CreateFor("tenant-a", "user-a", KindNoop, "audit-inject-0001"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "worker\nadmin", "worker/secret", "worker\radmin", "worker with spaces", "@worker"} {
		if _, err := s.ClaimNext(id, time.Minute); err == nil {
			t.Fatalf("unsafe worker ID accepted: %q", id)
		}
	}
	if _, err := s.ClaimNext("worker-safe.01", time.Minute); err != nil {
		t.Fatalf("safe worker ID unexpectedly rejected: %v", err)
	}
}

func TestSQLiteAuditActorConstraintsRejectSpoofing(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	_, err := s.db.Exec("INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) VALUES('spoof','tenant-a','user-a','worker','worker-7','test','x','y')")
	if err == nil {
		t.Fatal("audit schema accepted a forged worker event attributed to human")
	}
	_, err = s.db.Exec("INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) VALUES('spoof2','tenant-a',NULL,'user',NULL,'test','x','y')")
	if err == nil {
		t.Fatal("audit schema accepted a human event without a human identity")
	}
}
