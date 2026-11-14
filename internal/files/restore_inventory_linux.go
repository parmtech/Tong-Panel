//go:build linux

package files

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const MaxArchiveStageInventoryEntries = 10000

// CountRetainedArchiveStages reports a bounded, best-effort snapshot of
// private archive-stage directories directly under the pinned root. It does
// not read their contents, distinguish active stages, determine publication,
// authorize cleanup, or settle quota. Concurrent changes fail closed where
// observed but this count is not a durable journal or a filesystem snapshot.
// Cancellation is checked between operations but cannot interrupt a blocked
// filesystem call or mutex acquisition.
// A caller must establish current tenant authority and protect the root.
func (r *Root) CountRetainedArchiveStages(ctx context.Context) (int, error) {
	if ctx == nil {
		return 0, ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if r == nil {
		return 0, ErrClosed
	}
	r.operations.RLock()
	defer r.operations.RUnlock()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return 0, ErrClosed
	}
	fd, err := unix.Openat2(r.fd, ".", &unix.OpenHow{Flags: uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
	if err != nil {
		return 0, err
	}
	f := os.NewFile(uintptr(fd), "archive-stage-inventory")
	if f == nil {
		_ = unix.Close(fd)
		return 0, ErrUsageSnapshotDirty
	}
	defer f.Close()
	var before unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil {
		return 0, err
	}
	var names []string
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		remaining := MaxArchiveStageInventoryEntries + 1 - len(names)
		batch := 256
		if remaining < batch {
			batch = remaining
		}
		chunk, readErr := f.Readdirnames(batch)
		names = append(names, chunk...)
		if len(names) > MaxArchiveStageInventoryEntries {
			return 0, ErrUsageScanLimit
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
		if len(chunk) == 0 {
			return 0, ErrUsageSnapshotDirty
		}
	}
	count := 0
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if !strings.HasPrefix(name, archiveStagePrefix) {
			continue
		}
		if !validInternalStageName(name) {
			return 0, ErrUsageSnapshotDirty
		}
		var observed unix.Stat_t
		if err := unix.Fstatat(fd, name, &observed, unix.AT_SYMLINK_NOFOLLOW); err != nil || !privateArchiveStage(observed) {
			return 0, ErrUsageSnapshotDirty
		}
		child, err := unix.Openat2(fd, name, &unix.OpenHow{Flags: uint64(unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
		if err != nil {
			return 0, ErrUsageSnapshotDirty
		}
		var opened unix.Stat_t
		statErr := unix.Fstat(child, &opened)
		closeErr := unix.Close(child)
		if statErr != nil || closeErr != nil || !sameInternalStage(observed, opened) || !privateArchiveStage(opened) {
			return 0, ErrUsageSnapshotDirty
		}
		var current unix.Stat_t
		if err := unix.Fstatat(fd, name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || !sameInternalStage(opened, current) || !privateArchiveStage(current) {
			return 0, ErrUsageSnapshotDirty
		}
		count++
	}
	var after unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil || !sameInternalStage(before, after) || before.Mtim != after.Mtim || before.Ctim != after.Ctim {
		return 0, ErrUsageSnapshotDirty
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return count, nil
}
func privateArchiveStage(st unix.Stat_t) bool {
	return st.Mode&unix.S_IFMT == unix.S_IFDIR && st.Uid == uint32(unix.Geteuid()) && st.Mode&07777 == 0700
}

// CountRetainedArchiveStagesForTenant selects the server-mapped root, but does
// not authenticate the caller or bind the count to any publication record.
func (t *TenantRoots) CountRetainedArchiveStagesForTenant(ctx context.Context, tenantID string) (int, error) {
	if t == nil || tenantID == "" {
		return 0, ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return 0, ErrTenantDenied
	}
	return root.CountRetainedArchiveStages(ctx)
}
