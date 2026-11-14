//go:build linux

package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/parmtech/tong-panel/internal/files"
)

// TenantFileQuiescer must stop every non-API writer for this tenant root and
// keep it stopped until the returned resume function is called. Implementations
// must fail closed if they cannot prove the service is quiesced.
type TenantFileQuiescer interface {
	Quiesce(ctx context.Context, tenantID string) (resume func(), err error)
}

// ConfigureTenantQuotaFromRoots measures the assigned pinned root and records
// that usage before quota admission. Callers must stop external tenant writers
// and keep quota-enabled uploads disabled for the duration of the scan.
func (q *SQLiteTenantFileQuota) ConfigureTenantQuotaFromRoots(
	roots *files.TenantRoots,
	tenantID string,
	limitBytes int64,
	maxEntries, maxDepth int,
) error {
	if roots == nil {
		return fmt.Errorf("measure tenant file usage: %w", ErrFilesUnavailable)
	}
	usage, err := roots.MeasureUsageForTenant(tenantID, maxEntries, maxDepth)
	if err != nil {
		return fmt.Errorf("measure tenant file usage: %w", err)
	}
	if err := q.ConfigureTenantQuota(tenantID, limitBytes, usage); err != nil {
		return fmt.Errorf("configure measured tenant quota: %w", err)
	}
	return nil
}

// ReconcileTenantQuotaFromRoots is the crash-recovery path. It first quiesces
// external writers, then obtains the exclusive cross-process API barrier,
// measures the assigned pinned root, and atomically replaces usage while
// deleting only reservations stranded by a prior process crash. The external
// quiescer must remain held through the scan and SQLite commit.
func (q *SQLiteTenantFileQuota) ReconcileTenantQuotaFromRoots(
	ctx context.Context,
	barrier FileMutationBarrier,
	quiescer TenantFileQuiescer,
	roots *files.TenantRoots,
	tenantID string,
	limitBytes int64,
	maxEntries, maxDepth int,
) error {
	if ctx == nil || q == nil || q.db == nil || roots == nil || barrier == nil || quiescer == nil ||
		strings.TrimSpace(tenantID) == "" || strings.TrimSpace(tenantID) != tenantID || limitBytes < 0 {
		return ErrFilesUnavailable
	}
	resume, err := quiescer.Quiesce(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("quiesce external tenant file writers: %w", err)
	}
	if resume == nil {
		return fmt.Errorf("%w: tenant file quiescer returned no resume function", ErrFilesUnavailable)
	}
	defer resume()
	return WithExclusiveFileMaintenance(ctx, barrier, tenantID, func() error {
		usage, err := roots.MeasureUsageForTenant(tenantID, maxEntries, maxDepth)
		if err != nil {
			return fmt.Errorf("measure tenant files before quota recovery: %w", err)
		}
		if usage > limitBytes {
			return ErrFileQuotaExceeded
		}
		tx, err := q.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tenant quota recovery: %w", err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `DELETE FROM tenant_file_quota_reservations WHERE tenant_id = ?`, tenantID); err != nil {
			return fmt.Errorf("remove crash-stranded quota reservations: %w", err)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO tenant_file_quotas(tenant_id, limit_bytes, used_bytes, reserved_bytes, reconciled_at)
			VALUES(?, ?, ?, 0, ?) ON CONFLICT(tenant_id) DO UPDATE SET limit_bytes=excluded.limit_bytes,
			used_bytes=excluded.used_bytes, reserved_bytes=0, reconciled_at=excluded.reconciled_at`,
			tenantID, limitBytes, usage, q.now().Unix())
		if err != nil {
			return fmt.Errorf("write reconciled tenant quota: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit tenant quota recovery: %w", err)
		}
		return nil
	})
}
