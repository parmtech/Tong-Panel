package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestBeginImmediateCancellationRestoresBusyTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	primary, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	peer, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	writer, err := primary.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")
	conn, err := peer.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if restore, err := BeginImmediate(ctx, conn, 0); !errors.Is(err, context.DeadlineExceeded) {
		if restore != nil {
			restore()
		}
		t.Fatalf("BeginImmediate error=%v, want caller deadline", err)
	}
	assertBusyTimeout(t, conn, 5000)
}

func TestBeginImmediateInternalCapReturnsBusyAndRestoreIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	primary, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	peer, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	writer, err := primary.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")
	conn, err := peer.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	started := time.Now()
	restore, err := BeginImmediate(context.Background(), conn, 75*time.Millisecond)
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("BeginImmediate error=%v, want SQLite busy error", err)
	}
	if time.Since(started) < 50*time.Millisecond {
		t.Fatalf("internal cap expired too early: %v", time.Since(started))
	}
	if restore != nil {
		t.Fatal("restore callback returned on failed begin")
	}
	assertBusyTimeout(t, conn, 5000)
}

func TestBeginImmediateRestoresOriginalBusyTimeoutAfterSuccess(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "PRAGMA busy_timeout=1234"); err != nil {
		t.Fatal(err)
	}
	restore, err := BeginImmediate(context.Background(), conn, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	restore()
	restore()
	assertBusyTimeout(t, conn, 1234)
}

func TestApplyMigrationsHonorsCancellationAndCanRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	owner, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := owner.Ping(); err != nil {
		t.Fatal(err)
	}
	contender, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer contender.Close()
	contender.SetMaxOpenConns(1)
	if _, err := contender.Exec("PRAGMA busy_timeout=5000"); err != nil {
		t.Fatal(err)
	}

	writer, err := owner.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	started := time.Now()
	err = applyMigrations(ctx, contender)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("contended migration error=%v, want caller deadline", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("migration cancellation took too long: %v", elapsed)
	}
	var version int
	if err := contender.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 0 {
		t.Fatalf("schema version changed after canceled migration: version=%d err=%v", version, err)
	}
	if got := busyTimeout(t, contender); got != 5000 {
		t.Fatalf("busy_timeout after canceled migration=%d, want 5000", got)
	}

	if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrations(context.Background(), contender); err != nil {
		t.Fatalf("migration retry after writer release: %v", err)
	}
}

func busyTimeout(t *testing.T, db *sql.DB) int {
	t.Helper()
	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	return timeout
}

func assertBusyTimeout(t *testing.T, conn *sql.Conn, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&got); err != nil || got != want {
		t.Fatalf("busy_timeout=%d err=%v, want %dms", got, err, want)
	}
}
