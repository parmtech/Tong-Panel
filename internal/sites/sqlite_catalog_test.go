package sites

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func openCatalog(t *testing.T, path string) *Catalog {
	t.Helper()
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES(?,?)", tenant, tenant); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := NewCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestCatalogCreateStoresPendingDomainsAndScopesGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES(?,?)", tenant, tenant); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := NewCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	catalog.now = func() time.Time { return time.Unix(1234, 0) }
	spec := SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "Example.com.", Aliases: []string{"www.example.com"}, DocumentRoot: "sites/site-a/public"}
	if err := catalog.Create(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	record, err := catalog.Get(context.Background(), "tenant-a", "site-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.CreatedAt != 1234 || record.DocumentRoot != "sites/site-a/public" || len(record.Domains) != 2 {
		t.Fatalf("unexpected site record: %#v", record)
	}
	if record.Domains[0] != (DomainClaim{Name: "example.com", Kind: "primary", OwnershipState: "pending"}) ||
		record.Domains[1] != (DomainClaim{Name: "www.example.com", Kind: "alias", OwnershipState: "pending"}) {
		t.Fatalf("unexpected domain claims: %#v", record.Domains)
	}
	if _, err := catalog.Get(context.Background(), "tenant-b", "site-a"); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("cross-tenant site lookup error=%v", err)
	}
}

func TestCatalogCreateRejectsDuplicateDomainAndRollsBackSite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES(?,?)", tenant, tenant); err != nil {
			t.Fatal(err)
		}
	}
	catalog, _ := NewCatalog(db)
	for _, spec := range []SiteSpec{
		{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "shared.example", DocumentRoot: "sites/site-a/public"},
		{TenantID: "tenant-b", SiteID: "site-b", PrimaryDomain: "SHARED.example.", DocumentRoot: "sites/site-b/public"},
	} {
		err := catalog.Create(context.Background(), spec)
		if spec.TenantID == "tenant-a" && err != nil {
			t.Fatal(err)
		}
		if spec.TenantID == "tenant-b" && !errors.Is(err, ErrDomainClaimed) {
			t.Fatalf("second claim error=%v", err)
		}
	}
	if _, err := catalog.Get(context.Background(), "tenant-b", "site-b"); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("failed create left a partial site row: %v", err)
	}
}

func TestCatalogConcurrentCrossTenantDomainClaimHasSingleWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	catalogA := openCatalog(t, path)
	dbB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()
	if _, err := dbB.Exec("INSERT INTO tenants(id,name) VALUES(?,?)", "tenant-b", "Tenant B"); err != nil {
		t.Fatal(err)
	}
	catalogB, err := NewCatalog(dbB)
	if err != nil {
		t.Fatal(err)
	}
	specs := []SiteSpec{
		{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "race.example", DocumentRoot: "sites/site-a/public"},
		{TenantID: "tenant-b", SiteID: "site-b", PrimaryDomain: "race.example", DocumentRoot: "sites/site-b/public"},
	}
	catalogs := []*Catalog{catalogA, catalogB}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := range catalogs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results <- catalogs[i].Create(context.Background(), specs[i])
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrDomainClaimed):
			conflict++
		default:
			t.Errorf("unexpected concurrent claim error: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("successes=%d conflicts=%d, want one each", success, conflict)
	}
}

func TestSiteCatalogSchemaRejectsUncanonicalizedDomainAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('tenant-a','Tenant A')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO sites(tenant_id,site_id,document_root,created_at) VALUES('tenant-a','site-a','sites/../escape',1)"); err == nil {
		t.Fatal("database accepted a document root outside the constrained site path")
	}
	if _, err := db.Exec("INSERT INTO sites(tenant_id,site_id,document_root,created_at) VALUES('tenant-a','site-a','sites/site-a/public',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO site_domains(domain,tenant_id,site_id,kind) VALUES('UPPER.example','tenant-a','site-a','primary')"); err == nil {
		t.Fatal("database accepted a noncanonical uppercase domain")
	}
}
