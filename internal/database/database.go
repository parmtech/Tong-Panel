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

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

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
	fileURI := "file:" + filepath.ToSlash(abs)
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(FULL)")
	db, err := sql.Open("sqlite", fileURI+"?"+query.Encode())
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	// SQLite remains a single-writer store here; a one-connection pool also
	// makes pragma behavior deterministic for this development prototype.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect SQLite: %w", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	const schemaVersion = 5
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get SQLite migration connection: %w", err)
	}
	defer conn.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
			return fmt.Errorf("begin SQLite migration lock: %w", err)
		}
		var version int
		if err := conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("read SQLite schema version: %w", err)
		}
		if version > schemaVersion {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("SQLite schema version %d is newer than supported version %d", version, schemaVersion)
		}
		if version == schemaVersion {
			if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
				return fmt.Errorf("finish SQLite migration check: %w", err)
			}
			return nil
		}
		next := version + 1
		name := fmt.Sprintf("migrations/%04d_%s.sql", next, map[int]string{1: "initial", 2: "job_leases", 3: "audit_actor_provenance", 4: "totp_replay_state", 5: "mfa_enrollment"}[next])
		script, err := migrations.ReadFile(name)
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("read embedded SQLite migration %d: %w", next, err)
		}
		if _, err := conn.ExecContext(ctx, string(script)); err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("apply SQLite migration %d: %w", next, err)
		}
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", next)); err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("set SQLite schema version %d: %w", next, err)
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("commit SQLite migration %d: %w", next, err)
		}
	}
}
