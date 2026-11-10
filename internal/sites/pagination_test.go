package sites

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPageForUsesCurrentMembershipAndDistinguishesEmpty(t *testing.T) {
	c := openCatalog(t, filepath.Join(t.TempDir(), "sites.db"))
	ctx := context.Background()
	longTenant, longIdentity := strings.Repeat("t", 80), strings.Repeat("u", 90)
	if _, err := c.db.Exec("INSERT INTO tenants(id,name) VALUES(?, 'long')", longTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES(?, 'long-user')", longIdentity); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES(?,?,'auditor')", longTenant, longIdentity); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("INSERT INTO sites(tenant_id,site_id,document_root,created_at) VALUES(?, 'example','sites/example/public',1)", longTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("INSERT INTO site_domains(domain,tenant_id,site_id,kind,ownership_state) VALUES('long.example.test',?,'example','primary','pending')", longTenant); err != nil {
		t.Fatal(err)
	}
	page, next, err := c.ListPageFor(ctx, longTenant, longIdentity, "", 1)
	if err != nil || len(page) != 1 || page[0].SiteID != "example" || next != "" {
		t.Fatalf("valid long principal: %#v %q %v", page, next, err)
	}
	empty, _, err := c.ListPageFor(ctx, longTenant, longIdentity, "example", 1)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("authorized empty: %#v %v", empty, err)
	}
	if _, _, err := c.ListPageFor(ctx, "tenant-a", longIdentity, "", 1); !errors.Is(err, ErrSiteReadDenied) {
		t.Fatalf("cross-tenant: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := c.ListPageFor(canceled, longTenant, longIdentity, "", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
	for _, tc := range []struct{ disable, restore string }{
		{"UPDATE identities SET disabled=1 WHERE id=?", "UPDATE identities SET disabled=0 WHERE id=?"},
		{"UPDATE tenants SET status='suspended' WHERE id=?", "UPDATE tenants SET status='active' WHERE id=?"},
	} {
		id := longIdentity
		if strings.Contains(tc.disable, "tenants") {
			id = longTenant
		}
		if _, err := c.db.Exec(tc.disable, id); err != nil {
			t.Fatal(err)
		}
		if _, _, err := c.ListPageFor(ctx, longTenant, longIdentity, "", 1); !errors.Is(err, ErrSiteReadDenied) {
			t.Fatalf("revoked: %v", err)
		}
		if _, err := c.db.Exec(tc.restore, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.db.Exec("DELETE FROM memberships WHERE tenant_id=? AND identity_id=?", longTenant, longIdentity); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ListPageFor(ctx, longTenant, longIdentity, "", 1); !errors.Is(err, ErrSiteReadDenied) {
		t.Fatalf("removed membership: %v", err)
	}
	if err := c.db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.ListPageFor(ctx, longTenant, longIdentity, "", 1); err == nil || errors.Is(err, ErrSiteReadDenied) {
		t.Fatalf("database failure masked: %v", err)
	}
}

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
