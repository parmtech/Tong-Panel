package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func TestRecoveryMigrationUpgradesV11AndPreservesMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v11.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	names := []string{"initial", "job_leases", "audit_actor_provenance", "totp_replay_state", "mfa_enrollment", "login_attempt_buckets", "session_idle", "tenant_file_quota", "site_catalog", "domain_verification", "site_database_catalog"}
	for index, name := range names {
		script, err := migrations.ReadFile(fmt.Sprintf("migrations/%04d_%s.sql", index+1, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := old.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := old.Exec("PRAGMA user_version=11; INSERT INTO tenants(id,name) VALUES('existing','Existing tenant')"); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	if err := db.QueryRow("SELECT name FROM tenants WHERE id='existing'").Scan(&name); err != nil || name != "Existing tenant" {
		t.Fatalf("migration lost existing metadata: %v", err)
	}
	var version, sets, codes int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 12 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_sets").Scan(&sets); err != nil || sets != 0 {
		t.Fatalf("unexpected recovery sets: %d err=%v", sets, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM mfa_recovery_codes").Scan(&codes); err != nil || codes != 0 {
		t.Fatalf("unexpected recovery codes: %d err=%v", codes, err)
	}
}
