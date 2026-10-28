package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
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
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 8 {
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

func TestOpenMigratesV1JobsToFencedLeases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	v1, err := migrations.ReadFile("migrations/0001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(v1)); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A');
		INSERT INTO identities(id,username) VALUES('user-a','alice');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner');
		INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key)
		VALUES('legacy-job','tenant-a','user-a','validation.noop','running','legacy-0001');
		PRAGMA user_version=1`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	var state string
	var owner, tokenHash, availableAt sql.NullString
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 8 {
		t.Fatalf("migrated version=%d err=%v", version, err)
	}
	if err := db.QueryRow("SELECT state,lease_owner,lease_token_sha256,available_at FROM jobs WHERE id='legacy-job'").
		Scan(&state, &owner, &tokenHash, &availableAt); err != nil {
		t.Fatal(err)
	}
	if state != "queued" || owner.Valid || tokenHash.Valid || availableAt.Valid {
		t.Fatalf("legacy non-executing job was not safely requeued: state=%s owner=%v token=%v available=%v", state, owner, tokenHash, availableAt)
	}
}

func TestOpenRejectsNewerSchemaVersionWithoutDowngrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version = 9"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if db, err := Open(context.Background(), path); err == nil {
		db.Close()
		t.Fatal("opened database from a newer schema version")
	}
	db, err = sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 9 {
		t.Fatalf("newer schema version was changed: version=%d err=%v", version, err)
	}
}

func TestConcurrentOpenSerializesInitialMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	const openers = 6
	var wg sync.WaitGroup
	errs := make(chan error, openers)
	for i := 0; i < openers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db, err := Open(context.Background(), path)
			if err != nil {
				errs <- err
				return
			}
			if err := db.Close(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 8 {
		t.Fatalf("concurrent migration version=%d err=%v", version, err)
	}
}
