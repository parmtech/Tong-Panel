//go:build linux

package files

import (
	"io"
	"time"

	"golang.org/x/sys/unix"
)

// TenantRoots maps server-owned tenant identifiers to distinct pinned filesystem roots.
// Callers must derive tenantID from an authenticated, authorized principal;
// this mapping is not authentication or authorization by itself.
type TenantRoots struct {
	roots map[string]*Root
}

// NewTenantRoots copies a static tenant-to-root mapping. Empty identifiers or
// nil roots are rejected; callers cannot mutate the copied mapping afterward.
func NewTenantRoots(roots map[string]*Root) (*TenantRoots, error) {
	if len(roots) == 0 {
		return nil, ErrTenantDenied
	}
	type rootIdentity struct {
		dev uint64
		ino uint64
	}
	seen := make(map[rootIdentity]struct{}, len(roots))
	copied := make(map[string]*Root, len(roots))
	for tenantID, root := range roots {
		if tenantID == "" || root == nil {
			return nil, ErrTenantDenied
		}
		root.mu.RLock()
		if !root.open {
			root.mu.RUnlock()
			return nil, ErrTenantDenied
		}
		var stat unix.Stat_t
		statErr := unix.Fstat(root.fd, &stat)
		root.mu.RUnlock()
		if statErr != nil {
			return nil, ErrTenantDenied
		}
		identity := rootIdentity{dev: uint64(stat.Dev), ino: stat.Ino}
		if _, exists := seen[identity]; exists {
			return nil, ErrTenantDenied
		}
		seen[identity] = struct{}{}
		copied[tenantID] = root
	}
	return &TenantRoots{roots: copied}, nil
}

// ReadFileForTenant resolves only the root assigned to tenantID. It never
// falls back to a shared/default root. tenantID must come from the server's
// verified request principal, never directly from a client parameter.
func (t *TenantRoots) ReadFileForTenant(tenantID, relative string, maxBytes int64) ([]byte, error) {
	if t == nil || tenantID == "" {
		return nil, ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return nil, ErrTenantDenied
	}
	return root.ReadFile(relative, maxBytes)
}

// ListDirForTenant resolves only the root assigned to tenantID. The caller
// must derive this identifier from the verified server principal.
func (t *TenantRoots) ListDirForTenant(tenantID, relative string, limit int) ([]Entry, error) {
	if t == nil || tenantID == "" {
		return nil, ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return nil, ErrTenantDenied
	}
	return root.ListDir(relative, limit)
}

// WriteNewForTenant creates a file only under the pinned root assigned to
// tenantID. The ID must be derived from an authorized server principal.
func (t *TenantRoots) WriteNewForTenant(tenantID, relative string, content []byte) error {
	if t == nil || tenantID == "" {
		return ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return ErrTenantDenied
	}
	return root.WriteNew(relative, content)
}

// RemoveFileForTenant removes one file only from the root mapped to tenantID.
// The ID must be derived from the authorized server principal.
func (t *TenantRoots) RemoveFileForTenant(tenantID, relative string) error {
	if t == nil || tenantID == "" {
		return ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return ErrTenantDenied
	}
	return root.RemoveFile(relative)
}

// ExtractGzipTarNewForTenant extracts only beneath the pinned root selected by
// tenantID. The ID must come from an authenticated and authorized principal.
func (t *TenantRoots) ExtractGzipTarNewForTenant(tenantID, relative string, archive []byte, limits ArchiveLimits) error {
	if t == nil || tenantID == "" {
		return ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return ErrTenantDenied
	}
	return root.ExtractGzipTarNew(relative, archive, limits)
}

// ExtractGzipTarNewFromForTenant reads and extracts a bounded archive only
// beneath the pinned root selected by tenantID. The ID must come from an
// authenticated and authorized server-side principal.
func (t *TenantRoots) ExtractGzipTarNewFromForTenant(tenantID, relative string, src io.Reader, limits ArchiveLimits) error {
	if t == nil || tenantID == "" {
		return ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return ErrTenantDenied
	}
	return root.ExtractGzipTarNewFrom(relative, src, limits)
}

// CleanupStaleStagesForTenant cleans only internal staging directories beneath
// the root selected by tenantID.
func (t *TenantRoots) CleanupStaleStagesForTenant(tenantID string, olderThan time.Duration) (int, error) {
	if t == nil || tenantID == "" {
		return 0, ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return 0, ErrTenantDenied
	}
	return root.CleanupStaleStages(olderThan)
}

// CleanupStaleArchiveStagesForTenant is retained for callers of the earlier
// prototype and now includes create-only write stages as well.
func (t *TenantRoots) CleanupStaleArchiveStagesForTenant(tenantID string, olderThan time.Duration) (int, error) {
	return t.CleanupStaleStagesForTenant(tenantID, olderThan)
}
