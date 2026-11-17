package database

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotValidationRejectsDamagedBytesWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('validation-fixture','Synthetic snapshot validation')"); err != nil {
		t.Fatal(err)
	}
	image, err := CreateSnapshot(ctx, db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(image)
	if err != nil || len(valid) <= 4096 {
		t.Fatal("snapshot fixture is too small for page truncation")
	}
	for _, tc := range []struct {
		name  string
		bytes []byte
		valid bool
	}{
		{"valid image", valid, true},
		{"non SQLite bytes", []byte("synthetic fixture, not a SQLite database"), false},
		{"empty image", []byte{}, false},
		{"truncated header", valid[:24], false},
		{"truncated database pages", valid[:len(valid)/2], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "literal#%23 fixture.db")
			if err := os.WriteFile(path, tc.bytes, 0600); err != nil {
				t.Fatal(err)
			}
			err := validateSnapshot(ctx, path)
			if tc.valid {
				if err != nil {
					t.Fatal("valid snapshot rejected")
				}
			} else if !errors.Is(err, ErrSnapshotUnavailable) {
				t.Fatal("damaged snapshot was not rejected generically")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, tc.bytes) {
				t.Fatal("read-only validation changed snapshot bytes")
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
				t.Fatal("read-only validation created a sidecar or alternate file")
			}
		})
	}
}

func TestSnapshotValidationDoesNotCreateMissingImage(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "missing.db")
	if err := validateSnapshot(context.Background(), path); !errors.Is(err, ErrSnapshotUnavailable) {
		t.Fatal("missing snapshot was not rejected generically")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("read-only validation created a missing database")
	}
}
