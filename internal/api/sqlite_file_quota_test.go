package api

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
)

func openFileQuota(t *testing.T) (*sql.DB, *SQLiteTenantFileQuota, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','Quota test A'),('tenant-b','Quota test B')`); err != nil {
		t.Fatal(err)
	}
	quota, err := NewSQLiteTenantFileQuota(db)
	if err != nil {
		t.Fatal(err)
	}
	return db, quota, path
}

func quotaCounts(t *testing.T, db *sql.DB, tenant string) (used, reserved int64) {
	t.Helper()
	if err := db.QueryRow(`SELECT used_bytes, reserved_bytes FROM tenant_file_quotas WHERE tenant_id=?`, tenant).Scan(&used, &reserved); err != nil {
		t.Fatal(err)
	}
	return used, reserved
}

func TestSQLiteFileQuotaRequiresTrustedConfigurationAndAccountsReservations(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	if _, err := quota.Reserve("tenant-a", 1); !errors.Is(err, ErrFileQuotaUnconfigured) {
		t.Fatalf("unconfigured tenant reserve err=%v", err)
	}
	if err := quota.ConfigureTenantQuota("tenant-a", 10, 4); err != nil {
		t.Fatal(err)
	}
	empty, err := quota.Reserve("tenant-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.ConfigureTenantQuota("tenant-a", 10, 4); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("configuration with zero-byte active reservation err=%v", err)
	}
	if err := empty.Release(); err != nil {
		t.Fatal(err)
	}
	reservation, err := quota.Reserve("tenant-a", 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := quota.Reserve("tenant-a", 2); !errors.Is(err, ErrFileQuotaExceeded) {
		t.Fatalf("over-quota reserve err=%v", err)
	}
	if err := reservation.Commit(3); err != nil {
		t.Fatal(err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 7 || reserved != 0 {
		t.Fatalf("after commit used=%d reserved=%d", used, reserved)
	}
	if err := reservation.Commit(3); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("duplicate commit err=%v", err)
	}
}

func TestSQLiteFileQuotaReleaseAndReconciliationRefuseActiveReservation(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	if err := quota.ConfigureTenantQuota("tenant-a", 20, 0); err != nil {
		t.Fatal(err)
	}
	reservation, err := quota.Reserve("tenant-a", 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := quota.ConfigureTenantQuota("tenant-a", 20, 2); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("configuration during active reservation err=%v", err)
	}
	if err := reservation.Release(); err != nil {
		t.Fatal(err)
	}
	if err := reservation.Release(); !errors.Is(err, ErrFileQuotaReservation) {
		t.Fatalf("duplicate release err=%v", err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 0 || reserved != 0 {
		t.Fatalf("after release used=%d reserved=%d", used, reserved)
	}
	if err := quota.ConfigureTenantQuota("tenant-a", 20, 12); err != nil {
		t.Fatal(err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 12 || reserved != 0 {
		t.Fatalf("after trusted reconciliation used=%d reserved=%d", used, reserved)
	}
}

func TestSQLiteFileQuotaTenantIsolationAndConcurrentAdmission(t *testing.T) {
	db, quota, _ := openFileQuota(t)
	if err := quota.ConfigureTenantQuota("tenant-a", 10, 0); err != nil {
		t.Fatal(err)
	}
	if err := quota.ConfigureTenantQuota("tenant-b", 10, 0); err != nil {
		t.Fatal(err)
	}
	const workers = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successes int
	reservations := make([]FileQuotaReservation, 0, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := quota.Reserve("tenant-a", 2)
			if err == nil {
				mu.Lock()
				successes++
				reservations = append(reservations, r)
				mu.Unlock()
				return
			}
			if !errors.Is(err, ErrFileQuotaExceeded) {
				t.Errorf("unexpected reserve error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes != 5 {
		t.Fatalf("successful reservations=%d, want 5", successes)
	}
	if _, err := quota.Reserve("tenant-b", 10); err != nil {
		t.Fatalf("tenant B quota affected by tenant A: %v", err)
	}
	if used, reserved := quotaCounts(t, db, "tenant-a"); used != 0 || reserved != 10 {
		t.Fatalf("concurrent accounting used=%d reserved=%d", used, reserved)
	}
}

func TestSQLiteFileQuotaSurvivesDatabaseReopenWithPendingReservation(t *testing.T) {
	db, quota, path := openFileQuota(t)
	if err := quota.ConfigureTenantQuota("tenant-a", 10, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := quota.Reserve("tenant-a", 7); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered, err := NewSQLiteTenantFileQuota(reopened)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovered.Reserve("tenant-a", 4); !errors.Is(err, ErrFileQuotaExceeded) {
		t.Fatalf("pending reservation was not retained after restart: %v", err)
	}
	var count int
	if err := reopened.QueryRow(`SELECT count(*) FROM tenant_file_quota_reservations WHERE tenant_id='tenant-a'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("pending rows=%d err=%v", count, err)
	}
}
