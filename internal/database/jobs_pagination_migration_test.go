package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestJobsPaginationMigrationUpgradesV12(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v12.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	names := []string{"initial", "job_leases", "audit_actor_provenance", "totp_replay_state", "mfa_enrollment", "login_attempt_buckets", "session_idle", "tenant_file_quota", "site_catalog", "domain_verification", "site_database_catalog", "mfa_recovery"}
	for index, name := range names {
		script, err := migrations.ReadFile(fmt.Sprintf("migrations/%04d_%s.sql", index+1, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := old.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := old.Exec("PRAGMA user_version=12;INSERT INTO tenants(id,name) VALUES('preserved','Existing tenant')"); err != nil {
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
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion {
		t.Fatal("migration did not reach current version")
	}
	rows, err := db.Query("PRAGMA index_info(jobs_requester_page)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var sequence, columnID int
		var name string
		if err := rows.Scan(&sequence, &columnID, &name); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(columns, []string{"tenant_id", "requested_by", "id"}) {
		t.Fatal("pagination index scope or order incorrect")
	}
	rows.Close()
	var name string
	if err := db.QueryRow("SELECT name FROM tenants WHERE id='preserved'").Scan(&name); err != nil || name != "Existing tenant" {
		t.Fatal("pagination migration changed baseline metadata")
	}
}
