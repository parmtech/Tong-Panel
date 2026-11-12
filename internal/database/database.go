// Package database opens the embedded SQLite metadata store and applies its schema.
// This is an experimental single-process database foundation, not a production installer.
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// SchemaVersion is the newest schema version supported by this binary.
const SchemaVersion = 14

// Open creates the parent directory, enables foreign-key enforcement on every
// SQLite connection, applies embedded migrations, and rejects databases created
// by a newer schema version.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" || path == ":memory:" {
		return nil, errors.New("a file-backed SQLite path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, fmt.Errorf("create SQLite directory: %w", err)
	}
	uriPath := filepath.ToSlash(abs)
	// A Windows drive must be a URI path (/C:/...), never a URI host.
	// Existing UNC paths retain their leading double slash and empty authority.
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(FULL)")
	fileURI := (&url.URL{Scheme: "file", Path: uriPath, RawQuery: query.Encode()}).String()
	db, err := sql.Open("sqlite", fileURI)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	// SQLite remains a single-writer store here; a one-connection pool also
	// makes pragma behavior deterministic for this development prototype.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := pingSQLite(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect SQLite: %w", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Opening concurrent handles can contend while the driver's per-connection
// WAL pragma initializes a new database. SQLite may return SQLITE_BUSY from
// that pragma without waiting for busy_timeout. Retry only this transient
// startup condition, within a bounded caller-cancellable five-second window.
func pingSQLite(ctx context.Context, db *sql.DB) error {
	startup, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		err := db.PingContext(startup)
		if err == nil {
			return nil
		}
		var sqliteErr *sqlite.Error
		if !errors.As(err, &sqliteErr) || sqliteErr.Code()&0xff != 5 {
			return err
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-startup.Done():
			timer.Stop()
			return startup.Err()
		case <-timer.C:
		}
	}
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get SQLite migration connection: %w", err)
	}
	defer conn.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		restore, err := BeginImmediate(ctx, conn, 5*time.Second)
		if err != nil {
			return fmt.Errorf("begin SQLite migration lock: %w", err)
		}
		done, err := applyNextMigration(ctx, conn)
		restore()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

// applyNextMigration applies at most one migration while holding the SQLite
// writer reservation. The caller restores connection-local settings after this
// function returns.
func applyNextMigration(ctx context.Context, conn *sql.Conn) (bool, error) {
	rollback := func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }
	var version int
	if err := conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		rollback()
		return false, fmt.Errorf("read SQLite schema version: %w", err)
	}
	if version > SchemaVersion {
		rollback()
		return false, fmt.Errorf("SQLite schema version %d is newer than supported version %d", version, SchemaVersion)
	}
	if version == SchemaVersion {
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			rollback()
			return false, fmt.Errorf("finish SQLite migration check: %w", err)
		}
		return true, nil
	}
	next := version + 1
	name := fmt.Sprintf("migrations/%04d_%s.sql", next, map[int]string{1: "initial", 2: "job_leases", 3: "audit_actor_provenance", 4: "totp_replay_state", 5: "mfa_enrollment", 6: "login_attempt_buckets", 7: "session_idle", 8: "tenant_file_quota", 9: "site_catalog", 10: "domain_verification", 11: "site_database_catalog", 12: "mfa_recovery", 13: "jobs_pagination", 14: "domain_verification_evidence"}[next])
	script, err := migrations.ReadFile(name)
	if err != nil {
		rollback()
		return false, fmt.Errorf("read embedded SQLite migration %d: %w", next, err)
	}
	if _, err := conn.ExecContext(ctx, string(script)); err != nil {
		rollback()
		return false, fmt.Errorf("apply SQLite migration %d: %w", next, err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", next)); err != nil {
		rollback()
		return false, fmt.Errorf("set SQLite schema version %d: %w", next, err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		rollback()
		return false, fmt.Errorf("commit SQLite migration %d: %w", next, err)
	}
	return false, nil
}
