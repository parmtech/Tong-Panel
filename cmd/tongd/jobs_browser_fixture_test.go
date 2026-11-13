package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/api"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

// This explicitly selected test helper serves synthetic metadata over local TLS.
// It is not compiled into tongd and provides no credential-issuance HTTP route.
func TestJobsBrowserFixture(t *testing.T) {
	selected := false
	for _, arg := range os.Args[1:] {
		selected = selected || arg == "-test.run=^TestJobsBrowserFixture$"
	}
	if !selected || os.Getenv("TONG_UI_BROWSER_FIXTURE") != "yes" {
		return
	}
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "browser.db"))
	if err != nil {
		t.Fatal("fixture database unavailable")
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B'),('tenant-c','C');
		INSERT INTO identities(id,username) VALUES('alice','alice'),('bob','bob'),('carol','carol'),('auditor','auditor'),('empty','empty'),('bulk','bulk'),('siteempty','siteempty');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES
		('tenant-a','alice','owner'),('tenant-a','bob','customer'),('tenant-b','carol','customer'),
		('tenant-a','auditor','owner'),('tenant-a','empty','customer'),('tenant-a','bulk','customer'),('tenant-c','siteempty','customer');`); err != nil {
		t.Fatal("fixture identities unavailable")
	}
	queue, err := jobs.NewSQLiteStore(db)
	if err != nil {
		t.Fatal("fixture queue unavailable")
	}
	completed, err := queue.CreateFor("tenant-a", "alice", jobs.KindNoop, "browser-completed-0001")
	if err != nil {
		t.Fatal("fixture job unavailable")
	}
	claim, err := queue.ClaimNext("browser-fixture-worker", time.Minute)
	if err != nil || claim.Job.ID != completed.ID {
		t.Fatal("fixture claim unavailable")
	}
	if _, err := queue.Start(claim); err != nil {
		t.Fatal("fixture start unavailable")
	}
	if _, err := queue.Complete(claim); err != nil {
		t.Fatal("fixture completion unavailable")
	}
	ids := map[string]string{"completed": completed.ID}
	for _, fixture := range []struct{ key, tenant, identity string }{
		{"queued", "tenant-a", "alice"}, {"sameTenant", "tenant-a", "bob"},
		{"otherTenant", "tenant-b", "carol"}, {"auditor", "tenant-a", "auditor"},
	} {
		job, err := queue.CreateFor(fixture.tenant, fixture.identity, jobs.KindNoop, "browser-"+fixture.key+"-0001")
		if err != nil {
			t.Fatal("fixture job unavailable")
		}
		ids[fixture.key] = job.ID
	}
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE identity_id='auditor'"); err != nil {
		t.Fatal("fixture role unavailable")
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("fixture bulk transaction unavailable")
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key) VALUES(?,'tenant-a','bulk','validation.noop','queued',?)`)
	if err != nil {
		t.Fatal("fixture bulk statement unavailable")
	}
	for i := 1; i <= 1500; i++ {
		if _, err := stmt.Exec(fmt.Sprintf("%032x", i), fmt.Sprintf("bulk-fixture-%04d", i)); err != nil {
			stmt.Close()
			t.Fatal("fixture bulk jobs unavailable")
		}
	}
	if err := stmt.Close(); err != nil {
		t.Fatal("fixture bulk statement close failed")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal("fixture bulk commit failed")
	}
	catalog, err := sites.NewCatalog(db)
	if err != nil {
		t.Fatal("fixture site catalog unavailable")
	}
	for i := 0; i < 61; i++ {
		id := fmt.Sprintf("site-%03d", i)
		if err := catalog.Create(context.Background(), sites.SiteSpec{TenantID: "tenant-a", SiteID: id, PrimaryDomain: id + ".example.test", DocumentRoot: "sites/" + id + "/public"}); err != nil {
			t.Fatal("fixture site unavailable")
		}
	}
	if _, err := db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE tenant_id='tenant-a' AND site_id='site-000'"); err != nil {
		t.Fatal("fixture verified site unavailable")
	}
	if err := catalog.Create(context.Background(), sites.SiteSpec{TenantID: "tenant-b", SiteID: "other-site", PrimaryDomain: "other.example.test", DocumentRoot: "sites/other-site/public"}); err != nil {
		t.Fatal("fixture other site unavailable")
	}
	// Declarative catalog rows only: no external database or user is created.
	for _, spec := range []sites.SiteDatabaseSpec{
		{TenantID: "tenant-a", SiteID: "site-000", DatabaseID: "db-a"},
		{TenantID: "tenant-b", SiteID: "other-site", DatabaseID: "db-b"},
	} {
		if err := catalog.CreateDatabase(context.Background(), spec); err != nil {
			t.Fatal("fixture database metadata unavailable")
		}
	}
	sessions, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal("fixture sessions unavailable")
	}
	tokens := make(map[string]string)
	for _, principal := range []rbac.Principal{
		{UserID: "alice", TenantID: "tenant-a", Role: rbac.Owner},
		{UserID: "bob", TenantID: "tenant-a", Role: rbac.Customer},
		{UserID: "carol", TenantID: "tenant-b", Role: rbac.Customer},
		{UserID: "auditor", TenantID: "tenant-a", Role: rbac.Auditor},
		{UserID: "empty", TenantID: "tenant-a", Role: rbac.Customer},
		{UserID: "bulk", TenantID: "tenant-a", Role: rbac.Customer},
		{UserID: "siteempty", TenantID: "tenant-c", Role: rbac.Customer},
	} {
		token, _, _, err := sessions.Issue(principal, time.Hour)
		if err != nil {
			t.Fatal("fixture session unavailable")
		}
		tokens[principal.UserID] = token
	}
	primary := httptest.NewTLSServer(publicHandler(&api.Server{Sessions: sessions, SQLJobs: queue, SiteCatalog: catalog}))
	defer primary.Close()
	unconfigured := httptest.NewTLSServer(publicHandler(&api.Server{}))
	defer unconfigured.Close()
	noJobs := httptest.NewTLSServer(publicHandler(&api.Server{Sessions: sessions}))
	defer noJobs.Close()
	filesURL := setupFilesBrowserFixture(t, sessions)
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Primary, Unconfigured, NoJobs, Files string
		Tokens, IDs                          map[string]string
	}{primary.URL, unconfigured.URL, noJobs.URL, filesURL, tokens, ids}); err != nil {
		t.Fatal("fixture handshake failed")
	}
	// Closing stdin requests clean teardown. The browser harness also enforces
	// a process deadline and reaps the child on failure.
	_, _ = io.Copy(io.Discard, os.Stdin)
}
