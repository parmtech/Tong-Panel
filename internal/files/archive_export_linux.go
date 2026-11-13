//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/parmtech/tong-panel/internal/backups"
	"golang.org/x/sys/unix"
)

const maxExportDepth = 128

// ExportEncryptedGzipTar builds a bounded gzip-tar in memory from a nonempty
// tenant-relative source directory and streams it into the tenant-bound backup
// envelope. This helper neither authorizes access nor publishes the ciphertext:
// dst may contain a partial envelope on failure and must be privately staged by
// the caller. The caller must quiesce all external writers and source-directory
// moves; metadata rechecks detect some races but are not a filesystem snapshot.
// Symlinks, hardlinks, mount crossings and non-regular/special entries fail
// closed. Modes and timestamps are not preserved. Plaintext is held in memory;
// clearing the archive buffer is best effort and does not erase all Go copies.
// Cancellation is checked during reads/writes but cannot unblock an arbitrary
// blocked Reader or Writer. There is no production route or quota integration.
func (r *Root) ExportEncryptedGzipTar(ctx context.Context, relative string, dst io.Writer, tenantID string, key []byte, limits ArchiveLimits) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	if dst == nil {
		return backups.ErrInvalidBackup
	}
	if len(key) != 32 {
		return backups.ErrInvalidKey
	}
	if len(tenantID) == 0 || len(tenantID) > 128 {
		return backups.ErrInvalidTenant
	}
	for _, c := range []byte(tenantID) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return backups.ErrInvalidTenant
		}
	}
	for _, component := range strings.Split(relative, "/") {
		if hasInternalStagePrefix(component) {
			return ErrInvalidPath
		}
	}
	normalized, err := limits.withDefaults()
	if err != nil {
		return err
	}
	var compressed bytes.Buffer
	defer func() { clear(compressed.Bytes()) }()
	// Release the pinned-root locks before writing to an arbitrary destination.
	err = func() error {
		r.operations.Lock()
		defer r.operations.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		r.mu.RLock()
		defer r.mu.RUnlock()
		if !r.open {
			return ErrClosed
		}
		fd, err := unix.Openat2(r.fd, relative, &unix.OpenHow{Flags: uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
		if err != nil {
			return err
		}
		defer unix.Close(fd)
		boundedCompressed := &exportLimitWriter{dst: &compressed, remaining: normalized.CompressedBytes}
		gz := gzip.NewWriter(boundedCompressed)
		boundedExpanded := &exportLimitWriter{dst: gz, remaining: normalized.ExpandedBytes}
		tw := tar.NewWriter(boundedExpanded)
		state := exportArchiveState{ctx: ctx, tar: tw, limits: normalized}
		if err := state.addDirectory(fd, "", 0); err != nil {
			return err
		}
		if err := tw.Close(); err != nil {
			return err
		}
		return gz.Close()
	}()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return backups.Encrypt(contextExportWriter{ctx: ctx, dst: dst}, bytes.NewReader(compressed.Bytes()), tenantID, key, backups.Options{MaxPlaintextBytes: normalized.CompressedBytes})
}

// ExportEncryptedGzipTarForTenant binds registered root and envelope tenant ID.
// The identifier must come from currently authorized server-side state.
func (t *TenantRoots) ExportEncryptedGzipTarForTenant(ctx context.Context, tenantID, relative string, dst io.Writer, key []byte, limits ArchiveLimits) error {
	if t == nil || tenantID == "" {
		return ErrTenantDenied
	}
	root, ok := t.roots[tenantID]
	if !ok {
		return ErrTenantDenied
	}
	return root.ExportEncryptedGzipTar(ctx, relative, dst, tenantID, key, limits)
}

type exportLimitWriter struct {
	dst       io.Writer
	remaining int64
}

func (w *exportLimitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, ErrArchiveLimit
	}
	n, err := w.dst.Write(p)
	w.remaining -= int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

type contextExportWriter struct {
	ctx context.Context
	dst io.Writer
}

func (w contextExportWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.dst.Write(p)
	if e := w.ctx.Err(); e != nil {
		return n, e
	}
	return n, err
}

type exportArchiveState struct {
	ctx     context.Context
	tar     *tar.Writer
	limits  ArchiveLimits
	entries int
	total   int64
}

func sameExportStat(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Size == b.Size && a.Nlink == b.Nlink && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}
func (s *exportArchiveState) addDirectory(fd int, prefix string, depth int) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if depth > maxExportDepth {
		return ErrArchiveLimit
	}
	var before unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil {
		return err
	}
	if before.Mode&unix.S_IFMT != unix.S_IFDIR {
		return ErrUsageSnapshotDirty
	}
	dup, err := unix.Dup(fd)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(dup), "site-backup-dir")
	if f == nil {
		_ = unix.Close(dup)
		return ErrUsageSnapshotDirty
	}
	names, err := f.Readdirnames(s.limits.Entries - s.entries + 1)
	closeErr := f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if len(names) > s.limits.Entries-s.entries {
		return ErrArchiveLimit
	}
	sort.Strings(names)
	for _, name := range names {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00") || name == "." || name == ".." || hasInternalStagePrefix(name) {
			return ErrInvalidArchive
		}
		full := path.Join(prefix, name)
		clean, err := cleanArchivePath(full, false)
		if err != nil || clean != full {
			return ErrInvalidArchive
		}
		s.entries++
		if s.entries > s.limits.Entries {
			return ErrArchiveLimit
		}
		var observed unix.Stat_t
		if err := unix.Fstatat(fd, name, &observed, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return ErrUsageSnapshotDirty
		}
		switch observed.Mode & unix.S_IFMT {
		case unix.S_IFDIR:
			child, err := unix.Openat2(fd, name, &unix.OpenHow{Flags: uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
			if err != nil {
				return ErrUsageSnapshotDirty
			}
			var opened unix.Stat_t
			statErr := unix.Fstat(child, &opened)
			if statErr == nil && !sameExportStat(observed, opened) {
				statErr = ErrUsageSnapshotDirty
			}
			if statErr == nil {
				statErr = s.tar.WriteHeader(&tar.Header{Name: full + "/", Typeflag: tar.TypeDir, Mode: 0700})
			}
			if statErr == nil {
				statErr = s.addDirectory(child, full, depth+1)
			}
			closeErr := unix.Close(child)
			if statErr != nil {
				return statErr
			}
			if closeErr != nil {
				return closeErr
			}
			var current unix.Stat_t
			if err := unix.Fstatat(fd, name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || !sameExportStat(observed, current) {
				return ErrUsageSnapshotDirty
			}
		case unix.S_IFREG:
			if observed.Nlink != 1 {
				return ErrInvalidArchive
			}
			if observed.Size < 0 || observed.Size > s.limits.FileBytes || observed.Size > s.limits.TotalBytes-s.total {
				return ErrArchiveLimit
			}
			child, err := unix.Openat2(fd, name, &unix.OpenHow{Flags: uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_NOCTTY), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
			if err != nil {
				return ErrUsageSnapshotDirty
			}
			file := os.NewFile(uintptr(child), "site-backup-file")
			if file == nil {
				_ = unix.Close(child)
				return ErrUsageSnapshotDirty
			}
			var opened unix.Stat_t
			statErr := unix.Fstat(child, &opened)
			if statErr == nil && !sameExportStat(observed, opened) {
				statErr = ErrUsageSnapshotDirty
			}
			if statErr == nil {
				statErr = s.tar.WriteHeader(&tar.Header{Name: full, Typeflag: tar.TypeReg, Mode: 0600, Size: opened.Size})
			}
			if statErr == nil {
				_, statErr = io.CopyN(s.tar, contextBackupReader{ctx: s.ctx, src: file}, opened.Size)
			}
			var final unix.Stat_t
			if statErr == nil {
				statErr = unix.Fstat(child, &final)
				if statErr == nil && !sameExportStat(opened, final) {
					statErr = ErrUsageSnapshotDirty
				}
			}
			closeErr := file.Close()
			if statErr != nil {
				return statErr
			}
			if closeErr != nil {
				return closeErr
			}
			var current unix.Stat_t
			if err := unix.Fstatat(fd, name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || !sameExportStat(opened, current) {
				return ErrUsageSnapshotDirty
			}
			s.total += opened.Size
		default:
			return ErrInvalidArchive
		}
	}
	var after unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil || !sameExportStat(before, after) {
		return ErrUsageSnapshotDirty
	}
	return nil
}
