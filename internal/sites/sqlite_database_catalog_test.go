package sites

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

var safeDatabaseName = regexp.MustCompile(`^tp_[a-z2-7]{26}$`)
var safeDatabaseUser = regexp.MustCompile(`^tu_[a-z2-7]{26}$`)

func createTestSite(t *testing.T, catalog *Catalog, tenantID, siteID string) {
	t.Helper()
	err := catalog.Create(context.Background(), SiteSpec{
		TenantID: tenantID, SiteID: siteID,
		PrimaryDomain: siteID + ".example.test",
		DocumentRoot: "sites/" + siteID + "/public",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCatalogSiteDatabaseMetadataIsTenantScopedAndSecretFree(t *testing.T) {
	catalog := openCatalog(t, filepath.Join(t.TempDir(), "panel.db"))
	catalog.now = func() time.Time { return time.Unix(1234, 0) }
	createTestSite(t, catalog, "tenant-a", "site-a")
	createTestSite(t, catalog, "tenant-b", "site-b")

	specA := SiteDatabaseSpec{TenantID: "tenant-a", SiteID: "site-a", DatabaseID: "wordpress"}
	specB := SiteDatabaseSpec{TenantID: "tenant-b", SiteID: "site-b", DatabaseID: "wordpress"}
	for _, spec := range []SiteDatabaseSpec{specA, specB} {
		if err := catalog.CreateDatabase(context.Background(), spec); err != nil {
			t.Fatal(err)
		}
	}
	record, err := catalog.GetDatabase(context.Background(), specA.TenantID, specA.SiteID, specA.DatabaseID)
	if err != nil {
		t.Fatal(err)
	}
	if record.TenantID != specA.TenantID || record.SiteID != specA.SiteID ||
		record.DatabaseID != specA.DatabaseID || record.CreatedAt != 1234 {
		t.Fatalf("unexpected record: %#v", record)
	}
	if !safeDatabaseName.MatchString(record.DatabaseName) || !safeDatabaseUser.MatchString(record.Username) {
		t.Fatalf("generated unsafe MariaDB identifiers: %#v", record)
	}
	other, err := catalog.GetDatabase(context.Background(), specB.TenantID, specB.SiteID, specB.DatabaseID)
	if err != nil {
		t.Fatal(err)
	}
	if record.DatabaseName == other.DatabaseName || record.Username == other.Username {
		t.Fatalf("distinct owners received colliding identifiers: %#v %#v", record, other)
	}
	if _, err := catalog.GetDatabase(context.Background(), "tenant-b", "site-a", "wordpress"); !errors.Is(err, ErrSiteDatabaseNotFound) {
		t.Fatalf("cross-tenant database lookup error=%v", err)
	}
	if err := catalog.CreateDatabase(context.Background(), specA); !errors.Is(err, ErrSiteDatabaseExists) {
		t.Fatalf("duplicate create error=%v", err)
	}
}

func TestCatalogSiteDatabaseRejectsInvalidAndMissingOwner(t *testing.T) {
	catalog := openCatalog(t, filepath.Join(t.TempDir(), "panel.db"))
	for _, spec := range []SiteDatabaseSpec{
		{TenantID: "tenant-a", SiteID: "missing", DatabaseID: "wordpress"},
		{TenantID: "absent", SiteID: "site-a", DatabaseID: "wordpress"},
		{TenantID: "tenant-a", SiteID: "site-a", DatabaseID: "bad;drop"},
		{TenantID: "tenant-a", SiteID: "site-a", DatabaseID: ""},
	} {
		if err := catalog.CreateDatabase(context.Background(), spec); err == nil {
			t.Errorf("CreateDatabase(%#v) unexpectedly succeeded", spec)
		}
	}
	if err := catalog.CreateDatabase(context.Background(), SiteDatabaseSpec{
		TenantID: "tenant-a", SiteID: "site-a", DatabaseID: "wordpress",
	}); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("missing site's error=%v", err)
	}
	if _, err := catalog.GetDatabase(context.Background(), "tenant-a", "site-a", "wordpress"); !errors.Is(err, ErrSiteDatabaseNotFound) {
		t.Fatalf("missing database lookup error=%v", err)
	}
	if _, err := catalog.GetDatabase(context.Background(), "tenant-a", "site-a", "bad;drop"); !errors.Is(err, ErrInvalidDatabaseSpec) {
		t.Fatalf("invalid lookup error=%v", err)
	}
}

func TestCatalogConcurrentSameSiteDatabaseCreateHasSingleWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	catalogA := openCatalog(t, path)
	createTestSite(t, catalogA, "tenant-a", "site-a")
	dbB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()
	catalogB, err := NewCatalog(dbB)
	if err != nil {
		t.Fatal(err)
	}

	catalogs := []*Catalog{catalogA, catalogB}
	spec := SiteDatabaseSpec{TenantID: "tenant-a", SiteID: "site-a", DatabaseID: "wordpress"}
	start := make(chan struct{})
	results := make(chan error, len(catalogs))
	var wg sync.WaitGroup
	for _, catalog := range catalogs {
		wg.Add(1)
		go func(catalog *Catalog) {
			defer wg.Done()
			<-start
			results <- catalog.CreateDatabase(context.Background(), spec)
		}(catalog)
	}
	close(start)
	wg.Wait()
	close(results)
	success, duplicate := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrSiteDatabaseExists):
			duplicate++
		default:
			t.Errorf("unexpected concurrent create error: %v", err)
		}
	}
	if success != 1 || duplicate != 1 {
		t.Fatalf("successes=%d duplicates=%d, want exactly one each", success, duplicate)
	}
}
