package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigratingV2RelabelsHistoricallyMisattributedWorkerActions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "historical.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"migrations/0001_initial.sql", "migrations/0002_job_leases.sql"} {
		b, err := migrations.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(string(b)); err != nil {
			t.Fatalf("apply historical migration %s: %v", file, err)
		}
	}
	_, err = raw.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A');
		INSERT INTO identities(id,username) VALUES('user-a','alice');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner');
		INSERT INTO audit_events(id,tenant_id,actor_identity_id,action,subject_id,request_id)
			VALUES('old-human','tenant-a','user-a','job.created','job-a','old-human'),
			('old-worker','tenant-a','user-a','job.leased','job-a','old-worker'),
			('old-completed','tenant-a','user-a','job.completed','job-a','old-completed');
		PRAGMA user_version=2`)
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("v3/v4 migrations not applied, version=%d err=%v", version, err)
	}
	for _, tc := range []struct{ id, kind, user string }{
		{"old-human", "user", "user-a"},
		{"old-worker", "unknown", ""},
		{"old-completed", "unknown", ""},
	} {
		var kind string
		var user sql.NullString
		var worker sql.NullString
		if err := db.QueryRow("SELECT actor_kind,actor_identity_id,actor_ref FROM audit_events WHERE id=?", tc.id).
			Scan(&kind, &user, &worker); err != nil {
			t.Fatal(err)
		}
		if kind != tc.kind || user.String != tc.user || worker.Valid {
			t.Fatalf("unsafe migration %s: kind=%s user=%v worker=%v", tc.id, kind, user, worker)
		}
	}
}
