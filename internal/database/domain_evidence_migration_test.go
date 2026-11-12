package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func openDomainEvidenceV13(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		var version int
		if _, err := fmt.Sscanf(entry.Name(), "%04d_", &version); err != nil {
			t.Fatal(err)
		}
		if version > 13 {
			continue
		}
		script, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`PRAGMA user_version=13;
		INSERT INTO tenants(id,name) VALUES('tenant-a','Tenant A');
		INSERT INTO sites(tenant_id,site_id,document_root,created_at) VALUES('tenant-a','site-a','sites/site-a/public',1000);
		INSERT INTO site_domains(domain,tenant_id,site_id,kind,ownership_state,verification_token_sha256,verification_expires_at)
		VALUES('pending.example.test','tenant-a','site-a','primary','pending',printf('%064d',1),5000),
		      ('verified.example.test','tenant-a','site-a','alias','verified',NULL,NULL);`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDomainEvidenceMigrationPreservesLegacyUnknownProof(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v13.db")
	old := openDomainEvidenceV13(t, path)
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
		t.Fatal("evidence migration did not reach current schema")
	}
	for _, state := range []string{"pending", "verified"} {
		var actual string
		var evidenceAbsent bool
		err := db.QueryRow("SELECT ownership_state,ownership_observed_at IS NULL AND ownership_verified_at IS NULL AND ownership_verification_id IS NULL FROM site_domains WHERE domain=?", state+".example.test").Scan(&actual, &evidenceAbsent)
		if err != nil || actual != state || !evidenceAbsent {
			t.Fatalf("migration invented evidence or changed %s state", state)
		}
	}
	var digest string
	var expires int64
	if err := db.QueryRow("SELECT verification_token_sha256,verification_expires_at FROM site_domains WHERE domain='pending.example.test'").Scan(&digest, &expires); err != nil || digest != fmt.Sprintf("%064d", 1) || expires != 5000 {
		t.Fatal("migration changed pending proof")
	}
}

func TestDomainEvidenceMigrationFailureRollsBackColumnsAndRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v13-failure.db")
	old := openDomainEvidenceV13(t, path)
	if _, err := old.Exec(`CREATE TRIGGER site_domains_evidence_state_insert BEFORE INSERT ON site_domains BEGIN SELECT 1; END;`); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	failed, err := Open(context.Background(), path)
	if err == nil {
		failed.Close()
		t.Fatal("conflicting evidence trigger unexpectedly migrated")
	}
	if failed != nil || !strings.Contains(err.Error(), "apply SQLite migration 14") {
		t.Fatal("unexpected migration failure")
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var version, evidenceColumns int
	if err := raw.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 13 {
		t.Fatal("failed migration advanced version")
	}
	if err := raw.QueryRow("SELECT count(*) FROM pragma_table_info('site_domains') WHERE name IN ('ownership_observed_at','ownership_verified_at','ownership_verification_id')").Scan(&evidenceColumns); err != nil || evidenceColumns != 0 {
		t.Fatal("failed migration retained partial evidence columns")
	}
	var count int
	if err := raw.QueryRow("SELECT count(*) FROM site_domains").Scan(&count); err != nil || count != 2 {
		t.Fatal("failed migration changed domain rows")
	}
	if _, err := raw.Exec("DROP TRIGGER site_domains_evidence_state_insert"); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	retried, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer retried.Close()
	if err := retried.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion {
		t.Fatal("migration retry did not reach current schema")
	}
}
