//go:build linux

package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/files"
)

// Synthetic private stage in a test-owned temporary root models unresolved
// archive evidence; no real stage or production reservation is removed.
func TestQuotaReconcileRetainsReservationsForArchiveStage(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	if err := quota.ConfigureTenantQuota("tenant-a", 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := quota.Reserve("tenant-a", 5); err != nil {
		t.Fatal(err)
	}
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, "site.bin"), []byte("four"), 0600); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(rootPath, ".tong-archive-stage-0123456789abcdef0123456789abcdef")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "evidence"), []byte("candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	roots, err := files.NewTenantRoots(map[string]*files.Root{"tenant-a": root})
	if err != nil {
		t.Fatal(err)
	}
	barrier, _ := newTestLinuxFileMutationBarrier(t)
	quiescer := &testTenantFileQuiescer{}
	err = quota.ReconcileTenantQuotaFromRoots(context.Background(), barrier, quiescer, roots, "tenant-a", 20, 100, 8)
	if !errors.Is(err, files.ErrUsageSnapshotDirty) {
		t.Fatalf("unresolved archive stage accepted: %v", err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 0 || reserved != 5 {
		t.Fatalf("quota changed: used=%d reserved=%d", used, reserved)
	}
	var remaining int
	if err := db.QueryRow(`SELECT count(*) FROM tenant_file_quota_reservations WHERE tenant_id='tenant-a'`).Scan(&remaining); err != nil || remaining != 1 {
		t.Fatalf("reservation rows=%d err=%v", remaining, err)
	}
	if data, err := os.ReadFile(filepath.Join(stage, "evidence")); err != nil || string(data) != "candidate" {
		t.Fatalf("stage changed: %q %v", data, err)
	}
	if quiescer.called != 1 || quiescer.resumed != 1 {
		t.Fatalf("quiescer called=%d resumed=%d", quiescer.called, quiescer.resumed)
	}
	// Only the synthetic test fixture is removed here; real unresolved stages
	// require supervised reconciliation, which this test does not implement.
	if err := os.RemoveAll(stage); err != nil {
		t.Fatal(err)
	}
	if err := quota.ReconcileTenantQuotaFromRoots(context.Background(), barrier, quiescer, roots, "tenant-a", 20, 100, 8); err != nil {
		t.Fatal(err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 4 || reserved != 0 {
		t.Fatalf("recovery after test cleanup: used=%d reserved=%d", used, reserved)
	}
}
