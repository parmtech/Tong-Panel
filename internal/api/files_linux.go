//go:build linux

package api

import (
	"errors"
	"os"

	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

type tenantFileRoots struct {
	roots *files.TenantRoots
}

// NewTenantFileService adapts the Linux pinned-root implementation to the
// read-only API. Passing nil leaves Files unconfigured and every request fails
// closed with 503.
func NewTenantFileService(roots *files.TenantRoots) TenantFileService {
	if roots == nil {
		return nil
	}
	return tenantFileRoots{roots: roots}
}

func (f tenantFileRoots) ListForTenant(tenantID, relative string, limit int) ([]TenantFileEntry, error) {
	entries, err := f.roots.ListDirForTenant(tenantID, relative, limit)
	if err != nil {
		return nil, mapTenantFileError(err)
	}
	result := make([]TenantFileEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, TenantFileEntry{Name: entry.Name, Kind: string(entry.Kind), Size: entry.Size})
	}
	return result, nil
}

func (f tenantFileRoots) ReadForTenant(tenantID, relative string, maxBytes int64) ([]byte, error) {
	content, err := f.roots.ReadFileForTenant(tenantID, relative, maxBytes)
	if err != nil {
		return nil, mapTenantFileError(err)
	}
	return content, nil
}

func mapTenantFileError(err error) error {
	switch {
	case errors.Is(err, files.ErrTenantDenied):
		return ErrFilesTenantDenied
	case errors.Is(err, files.ErrInvalidPath), errors.Is(err, files.ErrNotRegular),
		errors.Is(err, unix.ELOOP), errors.Is(err, unix.EXDEV), errors.Is(err, unix.ENOTDIR):
		return ErrFilePathInvalid
	case errors.Is(err, files.ErrTooLarge):
		return ErrFileReadTooLarge
	case errors.Is(err, files.ErrInvalidLimit), errors.Is(err, files.ErrTooManyEntries):
		return ErrFileListTooLarge
	case errors.Is(err, os.ErrNotExist):
		return ErrFileNotFound
	default:
		return ErrFilesUnavailable
	}
}
