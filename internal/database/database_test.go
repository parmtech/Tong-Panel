package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigrationAndKeepsForeignKeysEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "panel.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version, foreignKeys int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatalf("migration version=%d err=%v", version, err)
	}
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d err=%v", foreignKeys, err)
	}
	_, err = db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('missing','missing','owner')")
	if err == nil {
		t.Fatal("foreign-key violation accepted")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(context.Background(), path)
	if err != nil {
		t.Fatalf("reopen should be idempotent: %v", err)
	}
	db.Close()
}
