//go:build linux

package jobs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

// This proves queue metadata recovery after a worker process is killed. It does
// not exercise tongd's fixed 30-second worker lease or simulate host power loss.
func TestSQLiteProcessCrashRecovery(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	job, err := store.CreateFor("tenant-a", "user-a", KindNoop, "process-crash-recovery-0001")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSQLiteCrashProcessHelper$")
	child.Env = crashHelperEnvironment("claim", path)
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	input, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	}()
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(io.LimitReader(output, 16)).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if line != "READY\n" {
			t.Fatal("child did not commit and announce its running job")
		}
	case <-ctx.Done():
		t.Fatal("worker handshake timed out")
	}
	killedAt := time.Now()
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = child.Wait()
	waited = true
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("worker did not exit from SIGKILL")
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatal("worker termination was not SIGKILL")
	}

	db, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var state, deadline string
	var attempts int
	err = db.QueryRowContext(ctx, "SELECT state,attempt_count,lease_until FROM jobs WHERE id=?", job.ID).Scan(&state, &attempts, &deadline)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if state != Running || attempts != 1 {
		db.Close()
		t.Fatal("killed worker's committed attempt was not durable")
	}
	expires, err := time.Parse(time.RFC3339Nano, deadline)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if !killedAt.Before(expires) {
		db.Close()
		t.Fatal("worker lease expired before SIGKILL was attempted")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	// Wait on the recorded real-time deadline; do not rewrite leases or clocks.
	if delay := time.Until(expires.Add(25 * time.Millisecond)); delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			t.Fatal("lease expiry timed out")
		}
	}
	recovery := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSQLiteCrashProcessHelper$")
	recovery.Env = crashHelperEnvironment("recover", path)
	if err := recovery.Run(); err != nil {
		t.Fatal("fresh worker failed to recover expired attempt")
	}
	db, err = database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err = NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Get("tenant-a", job.ID)
	if err != nil || current.State != Done {
		t.Fatal("recovered job did not complete")
	}
	var total, completed, expired, cleared int
	if err := db.QueryRowContext(ctx, "SELECT count(*),sum(status='completed'),sum(status='failed' AND error_code='lease_expired') FROM job_attempts WHERE job_id=?", job.ID).Scan(&total, &completed, &expired); err != nil || total != 2 || completed != 1 || expired != 1 {
		t.Fatal("unexpected durable attempt ledger")
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM jobs WHERE id=? AND attempt_count=2 AND lease_until IS NULL AND lease_owner IS NULL AND lease_token_sha256 IS NULL", job.ID).Scan(&cleared); err != nil || cleared != 1 {
		t.Fatal("completed job retained lease authority")
	}
	for _, event := range []struct {
		action, kind, actor string
		count               int
	}{
		{"job.created", "user", "user-a", 1},
		{"job.leased", "worker", "process-crash-test", 1}, {"job.started", "worker", "process-crash-test", 1},
		{"job.leased", "worker", "process-recover-test", 1}, {"job.started", "worker", "process-recover-test", 1},
		{"job.completed", "worker", "process-recover-test", 1}, {"job.lease_expired", "system", "lease-reconciler", 1},
	} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE subject_id=? AND action=? AND actor_kind=? AND COALESCE(actor_identity_id,actor_ref)=?", job.ID, event.action, event.kind, event.actor).Scan(&count); err != nil || count != event.count {
			t.Fatal("unexpected crash recovery audit provenance")
		}
	}
	var audits int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE subject_id=?", job.ID).Scan(&audits); err != nil || audits != 7 {
		t.Fatal("unexpected extra lifecycle audit")
	}
	retry, err := store.CreateFor("tenant-a", "user-a", KindNoop, "process-crash-recovery-0001")
	if err != nil || retry.ID != job.ID {
		t.Fatal("restart idempotency changed job identity")
	}
	if _, err := store.ClaimNext("process-final-test", time.Second); !errors.Is(err, ErrNoJobs) {
		t.Fatal("completed job could be claimed again")
	}
}

func TestSQLiteCrashProcessHelper(t *testing.T) {
	// An inherited helper environment alone must never activate this subprocess.
	selected := false
	for _, arg := range os.Args[1:] {
		if arg == "-test.run=^TestSQLiteCrashProcessHelper$" {
			selected = true
		}
	}
	if !selected {
		return
	}
	mode := os.Getenv("TONG_QUEUE_CRASH_HELPER")
	if mode == "" {
		return
	}
	if mode != "claim" && mode != "recover" {
		t.Fatal("invalid helper mode")
	}
	db, err := database.Open(context.Background(), os.Getenv("TONG_QUEUE_CRASH_DB"))
	if err != nil {
		t.Fatal("helper database unavailable")
	}
	defer db.Close()
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal("helper queue unavailable")
	}
	worker := "process-crash-test"
	if mode == "recover" {
		worker = "process-recover-test"
	}
	claim, err := store.ClaimNext(worker, 3*time.Second)
	if err != nil {
		t.Fatal("helper claim failed")
	}
	if _, err := store.Start(claim); err != nil {
		t.Fatal("helper start failed")
	}
	if mode == "claim" {
		if _, err := fmt.Fprintln(os.Stdout, "READY"); err != nil {
			t.Fatal("helper handshake failed")
		}
		var sentinel [1]byte
		_, _ = os.Stdin.Read(sentinel[:])
		t.Fatal("crash helper resumed without being killed")
	}
	if _, err := store.Complete(claim); err != nil {
		t.Fatal("helper completion failed")
	}
}

func crashHelperEnvironment(mode, path string) []string {
	var env []string
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "TONG_QUEUE_CRASH_HELPER=") && !strings.HasPrefix(value, "TONG_QUEUE_CRASH_DB=") {
			env = append(env, value)
		}
	}
	return append(env, "TONG_QUEUE_CRASH_HELPER="+mode, "TONG_QUEUE_CRASH_DB="+path)
}
