package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"modernc.org/sqlite"
)

func TestSQLiteStartupBusyHonorsCallerCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.db")
	owner, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	conn, err := owner.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "CREATE TABLE probe(id INTEGER); BEGIN EXCLUSIVE"); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	contender, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer contender.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := pingSQLite(ctx, contender); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("contended startup did not honor cancellation: %v", err)
	}
}

func TestSQLiteStartupDoesNotRetryFatalOpenError(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "missing", "panel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = pingSQLite(ctx, db)
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code()&0xff != 14 {
		t.Fatalf("fatal open error was not preserved: %v", err)
	}
}
