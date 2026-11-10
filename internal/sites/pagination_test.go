package sites

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestListPageTraversesTenantAndReturnsOnlyPrimary(t *testing.T) {
	c := openCatalog(t, filepath.Join(t.TempDir(), "sites.db"))
	for i := 0; i < 103; i++ {
		id := fmt.Sprintf("site-%03d", i)
		spec := SiteSpec{TenantID: "tenant-a", SiteID: id, PrimaryDomain: id + ".example.test", DocumentRoot: "sites/" + id + "/public"}
		if i == 0 {
			spec.Aliases = []string{"alias.example.test"}
		}
		if err := c.Create(context.Background(), spec); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Create(context.Background(), SiteSpec{TenantID: "tenant-b", SiteID: "site-001", PrimaryDomain: "other.example.test", DocumentRoot: "sites/site-001/public"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE domain='site-000.example.test'"); err != nil {
		t.Fatal(err)
	}
	var all []SiteSummary
	after := ""
	for {
		page, next, err := c.ListPage(context.Background(), "tenant-a", after, 50)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) > 50 || (next != "" && next != page[len(page)-1].SiteID) {
			t.Fatal("invalid page or cursor")
		}
		all = append(all, page...)
		if next == "" {
			break
		}
		after = next
	}
	if len(all) != 103 {
		t.Fatalf("got %d sites", len(all))
	}
	for i, item := range all {
		id := fmt.Sprintf("site-%03d", i)
		state := "pending"
		if i == 0 {
			state = "verified"
		}
		if item != (SiteSummary{SiteID: id, PrimaryDomain: id + ".example.test", OwnershipState: state}) {
			t.Fatalf("unexpected summary: %#v", item)
		}
	}
	other, next, err := c.ListPage(context.Background(), "tenant-b", "", 50)
	if err != nil || next != "" || len(other) != 1 || other[0].PrimaryDomain != "other.example.test" {
		t.Fatal("tenant isolation failed")
	}
	empty, next, err := c.ListPage(context.Background(), "tenant-b", "site-001", 50)
	if err != nil || next != "" || empty == nil || len(empty) != 0 {
		t.Fatal("empty page must be a nonnil slice")
	}
}

func TestListPageInvalidAndBackendErrors(t *testing.T) {
	c := openCatalog(t, filepath.Join(t.TempDir(), "sites.db"))
	for _, tc := range []struct {
		after string
		limit int
	}{{"", 0}, {"", 51}, {"../x", 1}, {"-bad", 1}, {"a/", 1}} {
		if _, _, err := c.ListPage(context.Background(), "tenant-a", tc.after, tc.limit); !errors.Is(err, ErrInvalidPage) {
			t.Fatal("invalid page accepted")
		}
	}
	if _, _, err := c.ListPage(nil, "tenant-a", "", 1); !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatal("nil context accepted")
	}
	if _, _, err := c.ListPage(context.Background(), "../x", "", 1); !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatal("invalid tenant accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := c.ListPage(ctx, "tenant-a", "", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error %v", err)
	}
	if err := c.Create(context.Background(), SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "missing.example.test", DocumentRoot: "sites/site-a/public"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("DELETE FROM site_domains WHERE domain='missing.example.test'"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ListPage(context.Background(), "tenant-a", "", 1); err == nil {
		t.Fatal("missing primary silently omitted")
	}
	if err := c.db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ListPage(context.Background(), "tenant-a", "", 1); err == nil {
		t.Fatal("closed database accepted")
	}
}

func TestListPageCursorIsBoundaryNotSnapshot(t *testing.T) {
	c := openCatalog(t, filepath.Join(t.TempDir(), "sites.db"))
	create := func(id string) {
		t.Helper()
		if err := c.Create(context.Background(), SiteSpec{TenantID: "tenant-a", SiteID: id, PrimaryDomain: id + ".example.test", DocumentRoot: "sites/" + id + "/public"}); err != nil {
			t.Fatal(err)
		}
	}
	create("site-b")
	create("site-d")
	first, next, err := c.ListPage(context.Background(), "tenant-a", "", 1)
	if err != nil || len(first) != 1 || next != "site-b" {
		t.Fatal("first page")
	}
	create("site-a")
	create("site-c")
	second, _, err := c.ListPage(context.Background(), "tenant-a", next, 1)
	if err != nil || len(second) != 1 || second[0].SiteID != "site-c" {
		t.Fatal("boundary failed")
	}
	refresh, _, err := c.ListPage(context.Background(), "tenant-a", "", 1)
	if err != nil || refresh[0].SiteID != "site-a" {
		t.Fatal("refresh missed earlier insertion")
	}
}
