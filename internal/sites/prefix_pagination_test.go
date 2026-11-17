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

func TestSitePrefixPageCurrentAuthorityAndLiteralMatching(t *testing.T) {
	ctx := context.Background()
	c := openCatalog(t, filepath.Join(t.TempDir(), "prefix.db"))
	if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner')"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 52; i++ {
		id := fmt.Sprintf("app_%02d", i)
		if err := c.Create(ctx, SiteSpec{TenantID: "tenant-a", SiteID: id, PrimaryDomain: fmt.Sprintf("a%d.example.test", i), DocumentRoot: "sites/" + id + "/public"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, spec := range []SiteSpec{{TenantID: "tenant-a", SiteID: "appX", PrimaryDomain: "x.example.test", DocumentRoot: "sites/appX/public"}, {TenantID: "tenant-b", SiteID: "app_00", PrimaryDomain: "b.example.test", DocumentRoot: "sites/app_00/public"}} {
		if err := c.Create(ctx, spec); err != nil {
			t.Fatal(err)
		}
	}
	page, next, err := c.ListPagePrefixFor(ctx, "tenant-a", "user-a", "", "app_", 50)
	if err != nil || len(page) != 50 || page[0].SiteID != "app_00" || next != "app_49" {
		t.Fatalf("first %#v %q %v", page, next, err)
	}
	page, next, err = c.ListPagePrefixFor(ctx, "tenant-a", "user-a", next, "app_", 50)
	if err != nil || len(page) != 2 || next != "" {
		t.Fatalf("next %#v %q %v", page, next, err)
	}
	other, _, err := c.ListPagePrefixFor(ctx, "tenant-b", "user-b", "", "app_", 50)
	if err != nil || len(other) != 1 || other[0].PrimaryDomain != "b.example.test" {
		t.Fatalf("tenant %#v %v", other, err)
	}
	for _, prefix := range []string{"app%", "a/b", string(make([]byte, 64))} {
		if _, _, err := c.ListPagePrefixFor(ctx, "tenant-a", "user-a", "", prefix, 50); !errors.Is(err, ErrInvalidPage) {
			t.Fatalf("invalid prefix %q: %v", prefix, err)
		}
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
		t.Fatal("proof")
	}
	if _, _, err := c.ListPagePrefixForSession(ctx, "tenant-a", "user-a", "", "app_", 50, proof); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.Exec("UPDATE sessions SET revoked_at=?", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if got, _, err := c.ListPagePrefixForSession(ctx, "tenant-a", "user-a", "zz", "app_", 50, proof); !errors.Is(err, ErrSiteSessionDenied) || len(got) != 0 {
		t.Fatalf("revoked %v", err)
	}
	if _, err := c.db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if got, _, err := c.ListPagePrefixFor(ctx, "tenant-a", "user-a", "", "app_", 50); !errors.Is(err, ErrSiteReadDenied) || len(got) != 0 {
		t.Fatalf("membership %v", err)
	}
}
