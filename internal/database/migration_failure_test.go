package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This exercises SQL-error rollback in a real embedded migration, followed by
// repair and retry. It does not simulate process death or host power loss.
func TestMigrationFailureRollsBackDDLAndRetries(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "migration.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	names := []string{"initial", "job_leases", "audit_actor_provenance", "totp_replay_state", "mfa_enrollment", "login_attempt_buckets", "session_idle", "tenant_file_quota", "site_catalog", "domain_verification", "site_database_catalog"}
	for index, name := range names {
		script, err := migrations.ReadFile(fmt.Sprintf("migrations/%04d_%s.sql", index+1, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.ExecContext(ctx, string(script)); err != nil {
			t.Fatal(err)
		}
	}
	// Seed after v11 so v7's intentional historical-session revocation is not
	// confused with a side effect of the attempted v12 upgrade.
	if _, err := raw.ExecContext(ctx, `PRAGMA user_version=11;
		PRAGMA foreign_keys=ON;
		INSERT INTO tenants(id,name) VALUES('migration-tenant','Baseline tenant');
		INSERT INTO identities(id,username,password_phc) VALUES('migration-user','migration-user','$argon2id$synthetic-fixture');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('migration-tenant','migration-user','owner');
		INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at,idle_deadline_unix)
		VALUES('migration-session','migration-tenant','migration-user',zeroblob(32),zeroblob(32),'2099-01-01T00:00:00Z',4070908800);
		INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key)
		VALUES('migration-job','migration-tenant','migration-user','validation.noop','queued','migration-idempotency-0001');
		INSERT INTO audit_events(id,tenant_id,actor_identity_id,action,subject_id,request_id)
		VALUES('migration-audit','migration-tenant','migration-user','job.created','migration-job','migration-request');
		CREATE TABLE mfa_recovery_codes(sentinel TEXT NOT NULL) STRICT;
		INSERT INTO mfa_recovery_codes(sentinel) VALUES('Preserve poison fixture');`); err != nil {
		t.Fatal(err)
	}
	baseline := migrationBaseline(t, raw)
	var first int
	if err := raw.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name='mfa_recovery_sets'").Scan(&first); err != nil || first != 0 {
		t.Fatal("first migration target already exists")
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	failed, err := Open(ctx, path)
	if err == nil {
		failed.Close()
		t.Fatal("conflicting second migration target unexpectedly succeeded")
	}
	if failed != nil || !strings.Contains(err.Error(), "apply SQLite migration 12") {
		t.Fatal("failure did not occur in embedded migration 12")
	}
	// Inspect without Open: it would immediately attempt the migration again.
	raw, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var version int
	if err := raw.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 11 {
		t.Fatal("failed migration advanced schema version")
	}
	if err := raw.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name='mfa_recovery_sets'").Scan(&first); err != nil || first != 0 {
		t.Fatal("first DDL statement survived failed migration")
	}
	var sentinel string
	if err := raw.QueryRow("SELECT sentinel FROM mfa_recovery_codes").Scan(&sentinel); err != nil || sentinel != "Preserve poison fixture" {
		t.Fatal("failed migration modified preexisting poison table")
	}
	if got := migrationBaseline(t, raw); !reflect.DeepEqual(got, baseline) {
		t.Fatal("failed migration changed baseline metadata")
	}
	if _, err := raw.Exec("DROP TABLE mfa_recovery_codes"); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	retried, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer retried.Close()
	if err := retried.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion {
		t.Fatal("migration retry did not reach current schema")
	}
	if got := migrationBaseline(t, retried); !reflect.DeepEqual(got, baseline) {
		t.Fatal("successful retry changed baseline metadata")
	}
	for _, table := range []string{"mfa_recovery_sets", "mfa_recovery_codes"} {
		var count int
		if err := retried.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("retry did not create empty recovery tables")
		}
	}
	var integrity string
	if err := retried.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal("retried database failed integrity check")
	}
	rows, err := retried.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() || rows.Err() != nil {
		t.Fatal("retry left foreign key violations")
	}
}

// SQLite quote() makes every value (including NULL and hashes) comparable
// without printing authentication metadata or depending on driver byte types.
func migrationBaseline(t *testing.T, db *sql.DB) []string {
	t.Helper()
	var result []string
	for _, table := range []string{"tenants", "identities", "memberships", "sessions", "jobs", "audit_events"} {
		rows, err := db.Query("SELECT * FROM " + table)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		var expressions []string
		for _, column := range columns {
			expressions = append(expressions, "quote("+column+")")
		}
		rows, err = db.Query("SELECT " + strings.Join(expressions, ", ") + " FROM " + table)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for rows.Next() {
			values := make([]string, len(columns))
			pointers := make([]any, len(columns))
			for index := range values {
				pointers[index] = &values[index]
			}
			if err := rows.Scan(pointers...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			result = append(result, table+":"+strings.Join(values, "|"))
			count++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatal("baseline fixture row count changed")
		}
	}
	return result
}
