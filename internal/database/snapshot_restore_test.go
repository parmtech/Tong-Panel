package database_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

// This restores only a disposable copy of synthetic control-plane metadata.
// It never replaces a live database or provisions the catalog's site/database.
func TestSnapshotRestoresCommittedWALMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	sourcePath := filepath.Join(t.TempDir(), "source#literal%23.db")
	source, err := database.Open(ctx, sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	// Open the second handle before taking its writer transaction. Both handles
	// are real SQLite connections; the source handle remains free for VACUUM.
	writer, err := database.Open(ctx, sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := source.ExecContext(ctx, "PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.ExecContext(ctx, "PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO tenants(id,name) VALUES('snapshot-tenant','Committed WAL fixture');
		INSERT INTO identities(id,username) VALUES('snapshot-user','snapshot-user');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('snapshot-tenant','snapshot-user','owner');`); err != nil {
		t.Fatal(err)
	}
	queue, err := jobs.NewSQLiteStore(source)
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.CreateFor("snapshot-tenant", "snapshot-user", jobs.KindNoop, "snapshot-idempotency-0001")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := queue.ClaimNext("snapshot-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claim.Job.ID != job.ID {
		t.Fatal("worker claimed unexpected fixture")
	}
	if _, err := queue.Start(claim); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Complete(claim); err != nil {
		t.Fatal(err)
	}
	catalog, err := sites.NewCatalog(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Create(ctx, sites.SiteSpec{TenantID: "snapshot-tenant", SiteID: "snapshot-site", PrimaryDomain: "snapshot.example.test", DocumentRoot: "sites/snapshot-site/public"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.CreateDatabase(ctx, sites.SiteDatabaseSpec{TenantID: "snapshot-tenant", SiteID: "snapshot-site", DatabaseID: "fixturedb"}); err != nil {
		t.Fatal(err)
	}
	wantSite, err := catalog.Get(ctx, "snapshot-tenant", "snapshot-site")
	if err != nil {
		t.Fatal(err)
	}
	wantDatabase, err := catalog.GetDatabase(ctx, "snapshot-tenant", "snapshot-site", "fixturedb")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewSQLiteSessionStore(source)
	if err != nil {
		t.Fatal(err)
	}
	principal := rbac.Principal{TenantID: "snapshot-tenant", UserID: "snapshot-user", Role: rbac.Owner}
	token, csrf, _, err := sessions.Issue(principal, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if wal, err := os.Stat(sourcePath + "-wal"); err != nil || wal.Size() <= 32 {
		t.Fatal("fixture has no committed WAL frames")
	}
	uncommitted, err := writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer uncommitted.Rollback()
	if _, err := uncommitted.ExecContext(ctx, "INSERT INTO tenants(id,name) VALUES('uncommitted-tenant','Must not enter snapshot')"); err != nil {
		t.Fatal(err)
	}
	image, err := database.CreateSnapshot(ctx, source, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	if err := uncommitted.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, "UPDATE tenants SET name='Changed after snapshot' WHERE id='snapshot-tenant'"); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Revoke(token); err != nil {
		t.Fatal(err)
	}
	if _, ok := sessions.Check(token); ok {
		t.Fatal("source session revocation did not take effect")
	}
	// The fresh destination receives exactly the standalone image, without
	// either source sidecar. Open may create its own WAL only after this copy.
	restoredPath := filepath.Join(t.TempDir(), "restored#metadata%23.db")
	f, err := os.OpenFile(restoredPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.Write(original)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal("copy snapshot fixture failed")
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(restoredPath + suffix); !os.IsNotExist(err) {
			t.Fatal("restore fixture unexpectedly has a source sidecar")
		}
	}
	restored, err := database.Open(ctx, restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	checkRestoredSnapshotIntegrity(t, ctx, restored)
	var tenantName string
	if err := restored.QueryRowContext(ctx, "SELECT name FROM tenants WHERE id='snapshot-tenant'").Scan(&tenantName); err != nil || tenantName != "Committed WAL fixture" {
		t.Fatal("committed WAL fixture was lost or later source mutation entered snapshot")
	}
	var n int
	if err := restored.QueryRowContext(ctx, "SELECT count(*) FROM tenants WHERE id='uncommitted-tenant'").Scan(&n); err != nil || n != 0 {
		t.Fatal("uncommitted source row entered snapshot")
	}
	restoredQueue, err := jobs.NewSQLiteStore(restored)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restoredQueue.GetFor("snapshot-tenant", "snapshot-user", job.ID)
	if err != nil || got.State != jobs.Done || got.RequestedBy != "snapshot-user" {
		t.Fatal("restored job ownership/state missing")
	}
	replay, err := restoredQueue.CreateFor("snapshot-tenant", "snapshot-user", jobs.KindNoop, "snapshot-idempotency-0001")
	if err != nil || replay.ID != job.ID {
		t.Fatal("restored queue lost idempotency")
	}
	if err := restored.QueryRowContext(ctx, "SELECT count(*) FROM job_attempts WHERE job_id=? AND status='completed'", job.ID).Scan(&n); err != nil || n != 1 {
		t.Fatal("restored completed attempt missing")
	}
	for _, event := range []struct{ action, kind, actor string }{{"job.created", "user", "snapshot-user"}, {"job.leased", "worker", "snapshot-worker"}, {"job.started", "worker", "snapshot-worker"}, {"job.completed", "worker", "snapshot-worker"}} {
		if err := restored.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE subject_id=? AND action=? AND actor_kind=? AND COALESCE(actor_identity_id,actor_ref)=?", job.ID, event.action, event.kind, event.actor).Scan(&n); err != nil || n != 1 {
			t.Fatal("restored job audit attribution missing")
		}
	}
	if err := restored.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE subject_id=?", job.ID).Scan(&n); err != nil || n != 4 {
		t.Fatal("restored idempotent replay duplicated audit")
	}
	restoredCatalog, err := sites.NewCatalog(restored)
	if err != nil {
		t.Fatal(err)
	}
	gotSite, err := restoredCatalog.Get(ctx, "snapshot-tenant", "snapshot-site")
	if err != nil || !reflect.DeepEqual(gotSite, wantSite) {
		t.Fatal("restored pending site/domain catalog differs")
	}
	gotDatabase, err := restoredCatalog.GetDatabase(ctx, "snapshot-tenant", "snapshot-site", "fixturedb")
	if err != nil || gotDatabase != wantDatabase {
		t.Fatal("restored database naming/ownership metadata differs")
	}
	restoredSessions, err := auth.NewSQLiteSessionStore(restored)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := restoredSessions.Check(token); !ok || got != principal || !restoredSessions.CheckCSRF(token, csrf) {
		t.Fatal("restored session metadata missing")
	}
	var plaintext int
	if err := restored.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE length(token_sha256)!=32 OR length(csrf_sha256)!=32").Scan(&plaintext); err != nil || plaintext != 0 {
		t.Fatal("restored session hashes invalid")
	}
	current, err := os.ReadFile(image)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatal("source updates or fresh-copy restore changed original snapshot")
	}
}

func checkRestoredSnapshotIntegrity(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var version, enabled int
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil || version != database.SchemaVersion {
		t.Fatal("restored schema version mismatch")
	}
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
		t.Fatal("restored foreign-key enforcement disabled")
	}
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal("restored integrity check failed")
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() || rows.Err() != nil {
		t.Fatal("restored foreign-key violations")
	}
}
