package database

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var ErrSnapshotUnavailable = errors.New("SQLite metadata snapshot unavailable")

var ErrSnapshotCleanupIncomplete = errors.New("SQLite metadata snapshot cleanup incomplete")

// CreateSnapshot writes a coherent, validated metadata image inside a new
// private directory under an existing, trusted parent. The caller owns the
// returned file and its directory. This does not encrypt the image, provision
// external keys, apply a restore, or guarantee survival of host power loss.
// Permission modes request POSIX privacy; they do not establish Windows ACLs.
func CreateSnapshot(ctx context.Context, db *sql.DB, parent string) (path string, err error) {
	if ctx == nil || db == nil || strings.TrimSpace(parent) == "" {
		return "", ErrSnapshotUnavailable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(parent)
	if err != nil {
		return "", ErrSnapshotUnavailable
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", ErrSnapshotUnavailable
	}
	directory, err := os.MkdirTemp(abs, "tong-metadata-")
	if err != nil {
		return "", ErrSnapshotUnavailable
	}
	complete := false
	defer func() {
		if !complete {
			path = ""
			if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
				err = errors.Join(err, ErrSnapshotCleanupIncomplete)
			}
		}
	}()
	path = filepath.Join(directory, "metadata.db")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", ErrSnapshotUnavailable
	}
	if err := file.Close(); err != nil {
		return "", ErrSnapshotUnavailable
	}
	// SQLite accepts an existing empty output file. The parameter is a literal
	// filename, never SQL text, and the unique directory prevents clobbering.
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrSnapshotUnavailable
	}
	if err := validateSnapshot(ctx, path); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	complete = true
	return path, nil
}

func validateSnapshot(ctx context.Context, path string) error {
	uriPath := filepath.ToSlash(path)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	uri := (&url.URL{Scheme: "file", Path: uriPath, RawQuery: "mode=ro"}).String()
	image, err := sql.Open("sqlite", uri)
	if err != nil {
		return ErrSnapshotUnavailable
	}
	defer image.Close()
	image.SetMaxOpenConns(1)
	var version int
	if err := image.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion {
		return ErrSnapshotUnavailable
	}
	rows, err := image.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return ErrSnapshotUnavailable
	}
	count, valid := 0, true
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil || result != "ok" {
			valid = false
		}
		count++
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil || !valid || count != 1 {
		return ErrSnapshotUnavailable
	}
	rows, err = image.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return ErrSnapshotUnavailable
	}
	invalid := rows.Next()
	err = rows.Err()
	closeErr = rows.Close()
	if invalid || err != nil || closeErr != nil {
		return ErrSnapshotUnavailable
	}
	return nil
}
