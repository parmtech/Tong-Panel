//go:build linux

package files

// TenantRoots maps server-owned tenant identifiers to pinned filesystem roots.
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
	copied := make(map[string]*Root, len(roots))
	for tenantID, root := range roots {
		if tenantID == "" || root == nil {
			return nil, ErrTenantDenied
		}
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
