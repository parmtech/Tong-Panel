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
// SQLite connection, applies the embedded initial schema to a new database, and
// rejects databases created by a newer schema version.
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
	const schemaVersion = 1
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		db.Close()
		return nil, fmt.Errorf("read SQLite schema version: %w", err)
	}
	if version > schemaVersion {
		db.Close()
		return nil, fmt.Errorf("SQLite schema version %d is newer than supported version %d", version, schemaVersion)
	}
	if version == schemaVersion {
		return db, nil
	}
	schema, err := migrations.ReadFile("migrations/0001_initial.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("read embedded schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(schema)); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply SQLite schema: %w", err)
	}
	return db, nil
}
