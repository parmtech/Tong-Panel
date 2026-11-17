package sites

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestACMEHTTP01PreflightAuthorizationAndOwnership(t *testing.T) {
	ctx := context.Background()
	catalog := openCatalog(t, filepath.Join(t.TempDir(), "panel.db"))
	if _, err := catalog.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner')"); err != nil {
		t.Fatal(err)
	}
	spec := SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "primary.example.test", Aliases: []string{"z.example.test", "a.example.test", "pending.example.test"}, DocumentRoot: "sites/site-a/public"}
	if err := catalog.Create(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if plan, err := catalog.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil); !errors.Is(err, ErrSiteNotReady) || plan.TenantID != "" || len(plan.Domains) != 0 {
		t.Fatal("pending primary produced preflight")
	}
	if _, err := catalog.db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE tenant_id='tenant-a' AND domain!='pending.example.test'"); err != nil {
		t.Fatal(err)
	}
	plan, err := catalog.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.TenantID != "tenant-a" || plan.SiteID != "site-a" || !reflect.DeepEqual(plan.Domains, []string{"primary.example.test", "a.example.test", "z.example.test"}) {
		t.Fatalf("wrong verified-domain snapshot: %+v", plan)
	}
	for _, tc := range []struct {
		tenant, user string
		want         error
	}{
		{"tenant-a", "user-b", ErrSiteReadDenied},
		{"tenant-b", "user-b", ErrSiteNotFound},
	} {
		got, err := catalog.ACMEHTTP01PreflightFor(ctx, tc.tenant, "site-a", tc.user, nil)
		if !errors.Is(err, tc.want) || got.TenantID != "" || len(got.Domains) != 0 {
			t.Fatal("cross-tenant metadata disclosed")
		}
	}
	// A snapshot cannot preserve authority after ownership or membership changes.
	if _, err := catalog.db.Exec("UPDATE site_domains SET ownership_state='pending' WHERE domain='primary.example.test'"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil); !errors.Is(err, ErrSiteNotReady) {
		t.Fatal("ownership was not reloaded")
	}
	if _, err := catalog.db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil); !errors.Is(err, ErrSiteReadDenied) {
		t.Fatal("membership revocation was not reloaded")
	}
}

func TestACMEHTTP01PreflightCancellationAndIsolation(t *testing.T) {
	catalog := openCatalog(t, filepath.Join(t.TempDir(), "panel.db"))
	if _, err := catalog.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner')"); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Create(context.Background(), SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "primary.example.test", DocumentRoot: "sites/site-a/public"}); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.db.Exec("UPDATE site_domains SET ownership_state='verified'"); err != nil {
		t.Fatal(err)
	}
	before, err := catalog.Get(context.Background(), "tenant-a", "site-a")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := catalog.ACMEHTTP01PreflightFor(context.Background(), "tenant-a", "site-a", "user-a", nil)
	if err != nil {
		t.Fatal(err)
	}
	plan.Domains[0] = "changed.example.test"
	sessions, err := auth.NewSQLiteSessionStore(catalog.db)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := sessions.ReadProof(token)
	if !ok {
		t.Fatal("missing session proof")
	}
	if _, err := catalog.ACMEHTTP01PreflightFor(context.Background(), "tenant-a", "site-a", "user-a", &proof); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.db.Exec("UPDATE sessions SET revoked_at=?", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if got, err := catalog.ACMEHTTP01PreflightFor(context.Background(), "tenant-a", "site-a", "user-a", &proof); !errors.Is(err, ErrSiteSessionDenied) || len(got.Domains) != 0 {
		t.Fatal("revoked session produced preflight")
	}
	after, err := catalog.Get(context.Background(), "tenant-a", "site-a")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("preflight mutated catalog")
	}
	var jobs int
	if err := catalog.db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobs); err != nil || jobs != 0 {
		t.Fatal("preflight created a job")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := catalog.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil); !errors.Is(err, context.Canceled) || len(got.Domains) != 0 {
		t.Fatal("cancellation did not fail closed")
	}
}
