//go:build linux

package api

import (
	"errors"
	"io"
	"net/http"
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

func (f tenantFileRoots) WriteNewForTenant(tenantID, relative string, content io.Reader, maxBytes int64) (int64, error) {
	counted := &countingReader{reader: content}
	err := f.roots.WriteNewFromForTenant(tenantID, relative, counted, maxBytes)
	if err != nil {
		if errors.Is(err, files.ErrPublishedDurabilityUnconfirmed) {
			return counted.read, ErrFilePublishedDurabilityUnconfirmed
		}
		return 0, mapTenantFileError(err)
	}
	return counted.read, nil
}

type countingReader struct {
	reader io.Reader
	read   int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	return n, err
}

func mapTenantFileError(err error) error {
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.Is(err, files.ErrTenantDenied):
		return ErrFilesTenantDenied
	case errors.Is(err, files.ErrInvalidPath), errors.Is(err, files.ErrNotRegular),
		errors.Is(err, unix.ELOOP), errors.Is(err, unix.EXDEV), errors.Is(err, unix.ENOTDIR):
		return ErrFilePathInvalid
	case errors.Is(err, files.ErrTooLarge), errors.As(err, &maxBytesErr):
		return ErrFileReadTooLarge
	case errors.Is(err, files.ErrAlreadyExists):
		return ErrFileAlreadyExists
	case errors.Is(err, files.ErrPublishedDurabilityUnconfirmed):
		return ErrFilePublishedDurabilityUnconfirmed
	case errors.Is(err, files.ErrInvalidLimit), errors.Is(err, files.ErrTooManyEntries):
		return ErrFileListTooLarge
	case errors.Is(err, os.ErrNotExist):
		return ErrFileNotFound
	default:
		return ErrFilesUnavailable
	}
}
