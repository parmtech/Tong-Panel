package sites

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func TestGuardedSiteWriteHonorsCancellationDuringWriterWait(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	catalog := openCatalog(t, path)
	if _, err := catalog.db.Exec("INSERT INTO identities(id,username) VALUES('actor','actor'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','actor','customer')"); err != nil {
		t.Fatal(err)
	}
	peerDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	writer, err := peerDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, createErr := catalog.CreateFor(ctx, SiteSpec{
			TenantID: "tenant-a", SiteID: "canceled", PrimaryDomain: "canceled.example.test", DocumentRoot: "sites/canceled/public",
		}, "actor", nil)
		result <- createErr
	}()
	deadline := time.Now().Add(2 * time.Second)
	for catalog.db.Stats().InUse == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if catalog.db.Stats().InUse == 0 {
		t.Fatal("site mutation did not acquire a pooled connection")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("site mutation error=%v, want caller deadline", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("site mutation did not stop after caller deadline")
	}
	if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	var sites, audits int
	if err := catalog.db.QueryRow("SELECT count(*) FROM sites WHERE site_id='canceled'").Scan(&sites); err != nil {
		t.Fatal(err)
	}
	if err := catalog.db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.created'").Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if sites != 0 || audits != 0 {
		t.Fatalf("canceled mutation committed site=%d audit=%d", sites, audits)
	}
	var busyTimeout int
	if err := catalog.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatalf("busy_timeout=%d err=%v, want restored 5000ms", busyTimeout, err)
	}
}
