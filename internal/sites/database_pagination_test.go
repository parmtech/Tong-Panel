package sites

import (
	"context"
	"errors"
	"fmt"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabasePagesBoundedAndCurrentAuthority(t *testing.T) {
	ctx := context.Background()
	c := openCatalog(t, filepath.Join(t.TempDir(), "catalog.db"))
	if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner')"); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "a.example.test", DocumentRoot: "sites/site-a/public"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, SiteSpec{TenantID: "tenant-b", SiteID: "site-a", PrimaryDomain: "b.example.test", DocumentRoot: "sites/site-a/public"}); err != nil {
		t.Fatal(err)
	}
	if err := c.CreateDatabase(ctx, SiteDatabaseSpec{TenantID: "tenant-b", SiteID: "site-a", DatabaseID: "db-00"}); err != nil {
		t.Fatal(err)
	}
	empty, _, err := c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "", 50, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty site: %#v %v", empty, err)
	}
	for i := 54; i >= 0; i-- {
		if err := c.CreateDatabase(ctx, SiteDatabaseSpec{TenantID: "tenant-a", SiteID: "site-a", DatabaseID: fmt.Sprintf("db-%02d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	page, next, err := c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "", 50, nil)
	if err != nil || len(page) != 50 || page[0].DatabaseID != "db-00" || page[49].DatabaseID != "db-49" || next != "db-49" {
		t.Fatalf("first page %#v %q %v", page, next, err)
	}
	page, next, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", next, 50, nil)
	if err != nil || len(page) != 5 || page[0].DatabaseID != "db-50" || next != "" {
		t.Fatalf("last page %#v %q %v", page, next, err)
	}
	page, next, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "zz", 50, nil)
	if err != nil || len(page) != 0 || next != "" {
		t.Fatalf("empty page %#v %q %v", page, next, err)
	}
	other, _, err := c.ListDatabasePageFor(ctx, "tenant-b", "site-a", "user-b", "", 50, nil)
	if err != nil || len(other) != 1 || other[0].TenantID != "tenant-b" {
		t.Fatalf("cross tenant: %#v %v", other, err)
	}
	if _, _, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-b", "", 50, nil); !errors.Is(err, ErrSiteReadDenied) {
		t.Fatalf("cross tenant identity: %v", err)
	}
	if _, _, err = c.ListDatabasePageFor(ctx, "tenant-a", "missing", "user-a", "", 50, nil); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("missing site: %v", err)
	}
	for _, limit := range []int{0, 51} {
		if _, _, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "", limit, nil); !errors.Is(err, ErrInvalidPage) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	if _, _, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "../bad", 1, nil); !errors.Is(err, ErrInvalidPage) {
		t.Fatalf("cursor: %v", err)
	}
	sessions, err := auth.NewSQLiteSessionStore(c.db)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := sessions.ReadProof(token)
	if !ok {
		t.Fatal("missing proof")
	}
	if _, _, err = c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "", 1, &proof); err != nil {
		t.Fatal(err)
	}
	if _, err = c.db.Exec("UPDATE sessions SET revoked_at=?", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if got, _, err := c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "zz", 1, &proof); !errors.Is(err, ErrSiteSessionDenied) || len(got) != 0 {
		t.Fatalf("revoked: %v", err)
	}
	if _, err = c.db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if got, _, err := c.ListDatabasePageFor(ctx, "tenant-a", "site-a", "user-a", "", 1, nil); !errors.Is(err, ErrSiteReadDenied) || len(got) != 0 {
		t.Fatalf("removed membership: %v", err)
	}
}
