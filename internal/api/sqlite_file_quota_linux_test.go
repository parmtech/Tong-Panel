//go:build linux

package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

type testTenantFileQuiescer struct {
	err      error
	called   int
	resumed  int
	noResume bool
}

func (q *testTenantFileQuiescer) Quiesce(_ context.Context, _ string) (func(), error) {
	q.called++
	if q.err != nil {
		return nil, q.err
	}
	if q.noResume {
		return nil, nil
	}
	return func() { q.resumed++ }, nil
}

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

func TestSQLiteFileQuotaReconciliationClearsOnlyAfterQuiescedMeasuredScan(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	if err := quota.ConfigureTenantQuota("tenant-a", 20, 0); err != nil {
		t.Fatal(err)
	}
	reservation, err := quota.Reserve("tenant-a", 11)
	if err != nil {
		t.Fatal(err)
	}
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, "site.bin"), []byte("measured"), 0600); err != nil {
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
	if err := quota.ReconcileTenantQuotaFromRoots(context.Background(), barrier, quiescer, roots, "tenant-a", 20, 100, 8); err != nil {
		t.Fatal(err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 8 || reserved != 0 {
		t.Fatalf("reconciled usage=%d reserved=%d, want 8 and 0", used, reserved)
	}
	var remaining int
	if err := db.QueryRow(`SELECT count(*) FROM tenant_file_quota_reservations WHERE tenant_id='tenant-a'`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("remaining reservation rows=%d err=%v", remaining, err)
	}
	if quiescer.called != 1 || quiescer.resumed != 1 {
		t.Fatalf("quiescer called=%d resumed=%d", quiescer.called, quiescer.resumed)
	}
	if err := reservation.Release(); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("stranded reservation handle release error=%v", err)
	}
	if _, err := quota.Reserve("tenant-a", 12); err != nil {
		t.Fatalf("reservation after recovery failed: %v", err)
	}
}

func TestSQLiteFileQuotaReconciliationFailsClosedBeforeClearingReservations(t *testing.T) {
	quiesceErr := errors.New("web service still active")
	for _, tc := range []struct {
		name     string
		quiescer *testTenantFileQuiescer
		limit    int64
		makeFifo bool
		want     error
	}{
		{name: "quiescer error", quiescer: &testTenantFileQuiescer{err: quiesceErr}, limit: 20, want: quiesceErr},
		{name: "quiescer without resume", quiescer: &testTenantFileQuiescer{noResume: true}, limit: 20, want: ErrFilesUnavailable},
		{name: "scan has special file", quiescer: &testTenantFileQuiescer{}, limit: 20, makeFifo: true, want: files.ErrUsageUnsupported},
		{name: "usage exceeds limit", quiescer: &testTenantFileQuiescer{}, limit: 3, want: ErrFileQuotaExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			if tc.makeFifo {
				if err := unix.Mkfifo(filepath.Join(rootPath, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
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
			err = quota.ReconcileTenantQuotaFromRoots(context.Background(), barrier, tc.quiescer, roots, "tenant-a", tc.limit, 100, 8)
			if !errors.Is(err, tc.want) {
				t.Fatalf("reconciliation error=%v, want %v", err, tc.want)
			}
			if used, reserved := quotaCounts(t, db, "tenant-a"); used != 0 || reserved != 5 {
				t.Fatalf("failed recovery changed quota: used=%d reserved=%d", used, reserved)
			}
			var remaining int
			if err := db.QueryRow(`SELECT count(*) FROM tenant_file_quota_reservations WHERE tenant_id='tenant-a'`).Scan(&remaining); err != nil || remaining != 1 {
				t.Fatalf("failed recovery reservation rows=%d err=%v", remaining, err)
			}
		})
	}
}
