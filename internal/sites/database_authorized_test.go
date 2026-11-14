package sites

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestGuardedDatabaseCreationNameCollisionRollsBackAudit(t *testing.T) {
	c := openCatalog(t, filepath.Join(t.TempDir(), "collision.db"))
	ctx := context.Background()
	if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES('actor','actor'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','actor','customer')"); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, SiteSpec{TenantID: "tenant-a", SiteID: "example", PrimaryDomain: "example.test", DocumentRoot: "sites/example/public"}); err != nil {
		t.Fatal(err)
	}
	target := SiteDatabaseSpec{TenantID: "tenant-a", SiteID: "example", DatabaseID: "target"}
	dbName, _ := siteDatabaseNames(target)
	if _, err := c.db.Exec("INSERT INTO site_databases(tenant_id,site_id,database_id,database_name,username,created_at) VALUES('tenant-a','example','other',?,?,1)", dbName, "tu_abcdefghijklmnopqrstuvwxyz"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateDatabaseFor(ctx, target, "actor", nil); !errors.Is(err, ErrSiteDatabaseNameCollision) {
		t.Fatalf("collision: %v", err)
	}
	var n int
	if err := c.db.QueryRow("SELECT count(*) FROM site_databases WHERE database_id='target'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("metadata %d %v", n, err)
	}
	if err := c.db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.database_metadata_created'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("audit %d %v", n, err)
	}
}
