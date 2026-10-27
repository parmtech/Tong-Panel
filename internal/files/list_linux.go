//go:build linux

package files

import (
	"errors"
	"io"
	"os"
	"sort"

	"golang.org/x/sys/unix"
)

const MaxListEntries = 1000

var ErrTooManyEntries = errors.New("directory exceeds configured entry limit")

// EntryKind is a non-following classification of one directory entry.
type EntryKind string

const (
	EntryFile    EntryKind = "file"
	EntryDir     EntryKind = "directory"
	EntrySymlink EntryKind = "symlink"
	EntryOther   EntryKind = "other"
)

// Entry contains bounded directory metadata. Symlinks are named but never
// followed; only regular files include a size.
type Entry struct {
	Name string
	Kind EntryKind
	Size int64
}

// ListDir returns a sorted, bounded snapshot of a directory below the pinned
// root. An empty path selects the root itself. Concurrent directory changes may
// make the snapshot incomplete, but lookups never follow symlinks or escape
// the root. Kernels without openat2 support fail closed.
func (r *Root) ListDir(relative string, limit int) ([]Entry, error) {
	if r == nil {
		return nil, ErrClosed
	}
	if relative != "" && !validRelativePath(relative) {
		return nil, ErrInvalidPath
	}
	if limit < 1 || limit > MaxListEntries {
		return nil, ErrInvalidLimit
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return nil, ErrClosed
	}

	dir := relative
	if dir == "" {
		dir = "."
	}
	fd, err := unix.Openat2(r.fd, dir, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), dir)
	if f == nil {
		_ = unix.Close(fd)
		return nil, ErrInvalidPath
	}
	defer f.Close()

	names, err := f.Readdirnames(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(names) > limit {
		return nil, ErrTooManyEntries
	}

	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		var stat unix.Stat_t
		if err := unix.Fstatat(fd, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOENT) {
				// Concurrent removal means the snapshot is incomplete.
				continue
			}
			return nil, err
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
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}
