//go:build linux

package api

import (
	"fmt"

	"github.com/parmtech/tong-panel/internal/files"
)

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
