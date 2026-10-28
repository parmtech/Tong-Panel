//go:build linux

package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/files"
)

func TestSQLiteFileQuotaConfiguresFromTrustedTenantRootUsage(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	rootPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(rootPath, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "nested", "site.bin"), []byte("measured"), 0600); err != nil {
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
	if err := quota.ConfigureTenantQuotaFromRoots(roots, "tenant-a", 12, 100, 8); err != nil {
		t.Fatal(err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != int64(len("measured")) || reserved != 0 {
		t.Fatalf("measured quota usage=%d reserved=%d", used, reserved)
	}
	reservation, err := quota.Reserve("tenant-a", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.ConfigureTenantQuotaFromRoots(roots, "tenant-a", 12, 100, 8); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("reconfigure with pending reservation error=%v", err)
	}
	if err := reservation.Release(); err != nil {
		t.Fatal(err)
	}
}
