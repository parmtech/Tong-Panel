//go:build linux

package files

import (
	"errors"
	"io"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

const (
	MaxUsageScanEntries = 100_000
	MaxUsageScanDepth   = 128
	usageReadBatch      = 256
)

var (
	ErrUsageScanLimit     = errors.New("tenant usage scan limit exceeded")
	ErrUsageSnapshotDirty = errors.New("tenant files changed during usage scan")
	ErrUsageUnsupported   = errors.New("tenant usage contains an unsupported file type")
)

// MeasureUsage reports logical bytes in regular files beneath the pinned root.
// Symlinks are skipped and never followed. The scan is bounded and serializes
// against mutations performed through this Root; callers must also quiesce
// external writers for a trustworthy reconciliation snapshot.
func (r *Root) MeasureUsage(maxEntries, maxDepth int) (int64, error) {
	if r == nil {
		return 0, ErrClosed
	}
	if maxEntries < 1 || maxEntries > MaxUsageScanEntries || maxDepth < 1 || maxDepth > MaxUsageScanDepth {
		return 0, ErrInvalidLimit
	}
	r.operations.Lock()
	defer r.operations.Unlock()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return 0, ErrClosed
	}
	rootFD, err := unix.Openat2(r.fd, ".", &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return 0, err
	}
	defer unix.Close(rootFD)
	state := usageScanState{maxEntries: maxEntries, maxDepth: maxDepth}
	if err := scanUsageDirectory(rootFD, 0, true, &state); err != nil {
		return 0, err
	}
	return state.total, nil
}

type usageScanState struct {
	maxEntries int
	maxDepth   int
	entries    int
	total      int64
}

func scanUsageDirectory(dirFD int, depth int, root bool, state *usageScanState) error {
	dup, err := unix.Dup(dirFD)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(dup), "tenant-usage-scan")
	if file == nil {
		_ = unix.Close(dup)
		return ErrUsageSnapshotDirty
	}
	defer file.Close()
	for {
		names, err := file.Readdirnames(usageReadBatch)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		for _, name := range names {
			state.entries++
			if state.entries > state.maxEntries {
				return ErrUsageScanLimit
			}
			var before unix.Stat_t
			if err := unix.Fstatat(dirFD, name, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				if errors.Is(err, unix.ENOENT) {
					return ErrUsageSnapshotDirty
				}
				return err
			}
			if root && (hasInternalStagePrefix(name)) {
				if validInternalStageName(name) && before.Mode&unix.S_IFMT == unix.S_IFDIR &&
					before.Uid == uint32(unix.Geteuid()) && before.Mode&0077 == 0 {
					continue
				}
				return ErrUsageSnapshotDirty
			}
			switch before.Mode & unix.S_IFMT {
			case unix.S_IFREG:
				if err := scanUsageRegularFile(dirFD, name, &before, state); err != nil {
					return err
				}
			case unix.S_IFDIR:
				if depth+1 > state.maxDepth {
					return ErrUsageScanLimit
				}
				childFD, err := unix.Openat2(dirFD, name, &unix.OpenHow{
					Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
					Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
				})
				if err != nil {
					if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ELOOP) || errors.Is(err, unix.EXDEV) {
						return ErrUsageSnapshotDirty
					}
					return err
				}
				var opened unix.Stat_t
				statErr := unix.Fstat(childFD, &opened)
				if statErr == nil && (opened.Dev != before.Dev || opened.Ino != before.Ino) {
					statErr = ErrUsageSnapshotDirty
				}
				if statErr == nil {
					statErr = scanUsageDirectory(childFD, depth+1, false, state)
				}
				closeErr := unix.Close(childFD)
				if statErr != nil {
					return statErr
				}
				if closeErr != nil {
					return closeErr
				}
			case unix.S_IFLNK:
				// Symlinks have no regular-file payload and are never traversed.
			default:
				return ErrUsageUnsupported
			}
		}
		if errors.Is(err, io.EOF) || len(names) == 0 {
			return nil
		}
	}
}

func scanUsageRegularFile(dirFD int, name string, before *unix.Stat_t, state *usageScanState) error {
	fd, err := unix.Openat2(dirFD, name, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ELOOP) || errors.Is(err, unix.EXDEV) {
			return ErrUsageSnapshotDirty
		}
		return err
	}
	defer unix.Close(fd)
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil {
		return err
	}
	if opened.Mode&unix.S_IFMT != unix.S_IFREG || opened.Dev != before.Dev || opened.Ino != before.Ino || opened.Size < 0 {
		return ErrUsageSnapshotDirty
	}
	if opened.Size > math.MaxInt64-state.total {
		return ErrUsageScanLimit
	}
	state.total += opened.Size
	return nil
}

func hasInternalStagePrefix(name string) bool {
	return len(name) >= len(writeStagePrefix) && name[:len(writeStagePrefix)] == writeStagePrefix ||
		len(name) >= len(archiveStagePrefix) && name[:len(archiveStagePrefix)] == archiveStagePrefix
}
