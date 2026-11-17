package jobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"modernc.org/sqlite"
)

func TestSQLiteFullClaimRollsBackAndPreservesIdempotency(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	const key = "sqlite-full-claim-0001"
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`CREATE TABLE queue_full_probe(payload BLOB NOT NULL);
		CREATE TRIGGER force_queue_sqlite_full
		AFTER INSERT ON audit_events WHEN NEW.action='job.leased'
		BEGIN INSERT INTO queue_full_probe(payload) VALUES(zeroblob(1048576)); END;`); err != nil {
		t.Fatal(err)
	}

	var pageCount int
	if err := store.db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		t.Fatal(err)
	}
	pageLimit := pageCount + 16 // 64 KiB of growth; the trigger requests 1 MiB.
	var actualLimit int
	if err := store.db.QueryRow(fmt.Sprintf("PRAGMA max_page_count=%d", pageLimit)).Scan(&actualLimit); err != nil {
		t.Fatal(err)
	}
	if actualLimit != pageLimit {
		t.Fatalf("SQLite page limit=%d, want %d", actualLimit, pageLimit)
	}

	claim, err := store.ClaimNext("full-test-worker", time.Minute)
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code()&0xff != 13 {
		t.Fatalf("ClaimNext error=%v, want primary SQLITE_FULL (13)", err)
	}
	if claim.Token != "" || claim.Job.ID != "" {
		t.Fatal("failed claim returned a worker capability")
	}
	var busyTimeout int
	if err := store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy timeout after SQLITE_FULL=%d, want 5000", busyTimeout)
	}

	var state string
	var attempts int
	var leaseUntilNull, leaseOwnerNull, leaseTokenNull bool
	if err := store.db.QueryRow(`SELECT state,attempt_count,lease_until IS NULL,lease_owner IS NULL,lease_token_sha256 IS NULL
		FROM jobs WHERE id=?`, job.ID).Scan(&state, &attempts, &leaseUntilNull, &leaseOwnerNull, &leaseTokenNull); err != nil {
		t.Fatal(err)
	}
	if state != Queued || attempts != 0 || !leaseUntilNull || !leaseOwnerNull || !leaseTokenNull {
		t.Fatalf("failed claim left job state=%q attempts=%d null lease fields=(%t,%t,%t)",
			state, attempts, leaseUntilNull, leaseOwnerNull, leaseTokenNull)
	}
	var attemptRows, createAudits, leaseAudits, probeRows, keyedJobs int
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=?", job.ID).Scan(&attemptRows); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.created'", job.ID).Scan(&createAudits); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.leased'", job.ID).Scan(&leaseAudits); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM queue_full_probe").Scan(&probeRows); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM jobs WHERE tenant_id='tenant-a' AND idempotency_key=?", key).Scan(&keyedJobs); err != nil {
		t.Fatal(err)
	}
	if attemptRows != 0 || createAudits != 1 || leaseAudits != 0 || probeRows != 0 || keyedJobs != 1 {
		t.Fatalf("rollback counts: attempts=%d createAudits=%d leaseAudits=%d probeRows=%d keyedJobs=%d",
			attemptRows, createAudits, leaseAudits, probeRows, keyedJobs)
	}

	if _, err := store.db.Exec("DROP TRIGGER force_queue_sqlite_full"); err != nil {
		t.Fatal(err)
	}
	var currentPages int
	if err := store.db.QueryRow("PRAGMA page_count").Scan(&currentPages); err != nil {
		t.Fatal(err)
	}
	pageLimit = currentPages + 64
	if err := store.db.QueryRow(fmt.Sprintf("PRAGMA max_page_count=%d", pageLimit)).Scan(&actualLimit); err != nil {
		t.Fatal(err)
	}
	if actualLimit != pageLimit {
		t.Fatalf("raised SQLite page limit=%d, want %d", actualLimit, pageLimit)
	}

	retry, err := store.CreateForContext(context.Background(), "tenant-a", "user-a", KindNoop, key)
	if err != nil || retry.ID != job.ID {
		t.Fatalf("idempotent retry=%+v err=%v, want original id %q", retry, err, job.ID)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM jobs WHERE tenant_id='tenant-a' AND idempotency_key=?", key).Scan(&keyedJobs); err != nil || keyedJobs != 1 {
		t.Fatalf("idempotent retry job count=%d err=%v", keyedJobs, err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.created'", job.ID).Scan(&createAudits); err != nil || createAudits != 1 {
		t.Fatalf("idempotent retry create audit count=%d err=%v", createAudits, err)
	}
	claim, err = store.ClaimNext("full-test-worker", time.Minute)
	if err != nil || claim.Job.ID != job.ID || claim.Token == "" {
		t.Fatalf("claim after recovery=%+v err=%v", claim, err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM job_attempts WHERE job_id=?", job.ID).Scan(&attemptRows); err != nil || attemptRows != 1 {
		t.Fatalf("recovered attempt count=%d err=%v", attemptRows, err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.leased'", job.ID).Scan(&leaseAudits); err != nil || leaseAudits != 1 {
		t.Fatalf("recovered lease audit count=%d err=%v", leaseAudits, err)
	}
}
