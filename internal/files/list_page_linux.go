//go:build linux

package files

import (
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// MaxPageScanEntries bounds work and memory for each directory page request.
const MaxPageScanEntries = 10000

// ListDirPage returns a name-ordered page and an exclusive continuation name.
// Each request scans afresh: concurrent changes can omit or repeat entries, so
// clients must restart for a fresh view. It is not a consistent snapshot.
// Directory names that cannot round-trip through JSON are rejected.
func (r *Root) ListDirPage(relative, after string, limit int) ([]Entry, string, error) {
	if r == nil {
		return nil, "", ErrClosed
	}
	if relative != "" && !validRelativePath(relative) {
		return nil, "", ErrInvalidPath
	}
	if limit < 1 || limit > MaxListEntries {
		return nil, "", ErrInvalidLimit
	}
	if after != "" && !pageNameValid(after) {
		return nil, "", ErrInvalidPath
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return nil, "", ErrClosed
	}
	dir := relative
	if dir == "" {
		dir = "."
	}
	fd, err := unix.Openat2(r.fd, dir, &unix.OpenHow{Flags: uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
	if err != nil {
		return nil, "", err
	}
	f := os.NewFile(uintptr(fd), dir)
	if f == nil {
		_ = unix.Close(fd)
		return nil, "", ErrInvalidPath
	}
	defer f.Close()
	names, err := f.Readdirnames(MaxPageScanEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, "", err
	}
	if len(names) > MaxPageScanEntries {
		return nil, "", ErrTooManyEntries
	}
	visible := names[:0]
	for _, name := range names {
		if !pageNameValid(name) {
			return nil, "", ErrInvalidPath
		}
		if relative == "" && (strings.HasPrefix(name, writeStagePrefix) || strings.HasPrefix(name, archiveStagePrefix)) {
			continue
		}
		if name > after {
			visible = append(visible, name)
		}
	}
	sort.Strings(visible)
	next := ""
	if len(visible) > limit {
		next = visible[limit-1]
		visible = visible[:limit]
	}
	entries := make([]Entry, 0, len(visible))
	for _, name := range visible {
		var stat unix.Stat_t
		if err := unix.Fstatat(fd, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			return nil, "", err
		}
		entry := Entry{Name: name}
		switch stat.Mode & unix.S_IFMT {
		case unix.S_IFREG:
			entry.Kind = EntryFile
			entry.Size = stat.Size
		case unix.S_IFDIR:
			entry.Kind = EntryDir
		case unix.S_IFLNK:
			entry.Kind = EntrySymlink
		default:
			entry.Kind = EntryOther
		}
		entries = append(entries, entry)
	}
	return entries, next, nil
}

// pageNameValid matches the explorer's single-component path constraints so
// every returned name and continuation can be represented and traversed.
func pageNameValid(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, "/\\") {
		return false
	}
	for _, c := range name {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func (t *TenantRoots) ListDirPageForTenant(tenantID, relative, after string, limit int) ([]Entry, string, error) {
	if t == nil || tenantID == "" {
		return nil, "", ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return nil, "", ErrTenantDenied
	}
	return root.ListDirPage(relative, after, limit)
}
