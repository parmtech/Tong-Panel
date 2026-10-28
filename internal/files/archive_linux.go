//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	MaxArchiveCompressedBytes int64 = 32 << 20
	MaxArchiveExpandedBytes   int64 = 80 << 20
	MaxArchiveFileBytes       int64 = 16 << 20
	MaxArchiveTotalBytes      int64 = 64 << 20
	MaxArchiveEntries               = 2048
)

var (
	ErrInvalidArchive = errors.New("invalid or unsupported archive entry")
	ErrArchiveLimit   = errors.New("archive exceeds configured limit")
)

// ArchiveLimits may lower the hard bounds for a caller. Zero fields select the
// package defaults; values above those defaults are rejected.
type ArchiveLimits struct {
	CompressedBytes int64
	ExpandedBytes   int64
	FileBytes       int64
	TotalBytes      int64
	Entries         int
}

func (l ArchiveLimits) withDefaults() (ArchiveLimits, error) {
	if l.CompressedBytes == 0 {
		l.CompressedBytes = MaxArchiveCompressedBytes
	}
	if l.ExpandedBytes == 0 {
		l.ExpandedBytes = MaxArchiveExpandedBytes
	}
	if l.FileBytes == 0 {
		l.FileBytes = MaxArchiveFileBytes
	}
	if l.TotalBytes == 0 {
		l.TotalBytes = MaxArchiveTotalBytes
	}
	if l.Entries == 0 {
		l.Entries = MaxArchiveEntries
	}
	if l.CompressedBytes < 1 || l.CompressedBytes > MaxArchiveCompressedBytes ||
		l.ExpandedBytes < 1 || l.ExpandedBytes > MaxArchiveExpandedBytes ||
		l.FileBytes < 1 || l.FileBytes > MaxArchiveFileBytes ||
		l.TotalBytes < 1 || l.TotalBytes > MaxArchiveTotalBytes ||
		l.Entries < 1 || l.Entries > MaxArchiveEntries {
		return ArchiveLimits{}, ErrInvalidLimit
	}
	return l, nil
}

// ReadGzipTarBounded reads at most maxBytes+1 bytes to determine whether the
// compressed input exceeds its configured cap. It never reads the full
// untrusted stream before checking the limit. Callers should pass request
// bodies directly to ExtractGzipTarNewFrom rather than allocating them first.
func ReadGzipTarBounded(src io.Reader, maxBytes int64) ([]byte, error) {
	if src == nil {
		return nil, ErrInvalidArchive
	}
	if maxBytes < 1 || maxBytes > MaxArchiveCompressedBytes {
		return nil, ErrInvalidLimit
	}
	data, err := io.ReadAll(io.LimitReader(src, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrArchiveLimit
	}
	return data, nil
}

// ExtractGzipTarNew extracts a bounded gzip-compressed tar archive into a new
// directory below the pinned root. It publishes the completed tree atomically
// and never replaces an existing path. Only regular files and directories are
// accepted; symlinks, hardlinks, devices, FIFOs, traversal and duplicate paths
// fail closed. Files use mode 0600 and directories mode 0700.
func (r *Root) ExtractGzipTarNew(relative string, archive []byte, limits ArchiveLimits) error {
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	var err error
	limits, err = limits.withDefaults()
	if err != nil {
		return err
	}
	if int64(len(archive)) > limits.CompressedBytes {
		return ErrArchiveLimit
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return ErrClosed
	}
	parent, base, err := r.openParent(relative)
	if err != nil {
		return err
	}
	defer unix.Close(parent)

	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	stage := ".tong-archive-stage-" + hex.EncodeToString(random[:])
	if err := unix.Mkdirat(r.fd, stage, 0700); err != nil {
		return err
	}
	stageFD, err := unix.Openat2(r.fd, stage, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		_ = unix.Unlinkat(r.fd, stage, unix.AT_REMOVEDIR)
		return err
	}
	if err := unix.Flock(stageFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = unix.Close(stageFD)
		_ = unix.Unlinkat(r.fd, stage, unix.AT_REMOVEDIR)
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = removeArchiveTree(stageFD)
			_ = unix.Unlinkat(r.fd, stage, unix.AT_REMOVEDIR)
		}
		_ = unix.Close(stageFD)
	}()

	if err := extractTarInto(stageFD, archive, limits); err != nil {
		return err
	}
	if err := syncArchiveTree(stageFD); err != nil {
		return err
	}
	if err := unix.Renameat2(r.fd, stage, parent, base, unix.RENAME_NOREPLACE); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return ErrAlreadyExists
		}
		return err
	}
	published = true
	if err := unix.Fsync(parent); err != nil {
		return err
	}
	rootDir, err := unix.Openat2(r.fd, ".", &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return err
	}
	defer unix.Close(rootDir)
	return unix.Fsync(rootDir)
}

// ExtractGzipTarNewFrom reads a gzip tar stream under its compressed-byte
// limit, then extracts it using the same atomic no-replace contract as
// ExtractGzipTarNew. It does not close src.
func (r *Root) ExtractGzipTarNewFrom(relative string, src io.Reader, limits ArchiveLimits) error {
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	limits, err := limits.withDefaults()
	if err != nil {
		return err
	}
	r.mu.RLock()
	open := r.open
	r.mu.RUnlock()
	if !open {
		return ErrClosed
	}
	archive, err := ReadGzipTarBounded(src, limits.CompressedBytes)
	if err != nil {
		return err
	}
	return r.ExtractGzipTarNew(relative, archive, limits)
}

func extractTarInto(stageFD int, archive []byte, limits ArchiveLimits) error {
	source := bytes.NewReader(archive)
	gz, err := gzip.NewReader(source)
	if err != nil {
		return ErrInvalidArchive
	}
	gz.Multistream(false)
	defer gz.Close()

	// Read one byte beyond the inclusive limit so the caller can distinguish
	// an archive that ends exactly at the limit from one that exceeds it.
	bounded := &io.LimitedReader{R: gz, N: limits.ExpandedBytes + 1}
	tr := tar.NewReader(bounded)
	seenHeaders := make(map[string]struct{})
	directories := make(map[string]struct{})
	files := make(map[string]struct{})
	nodes := 0
	totalBytes := int64(0)
	headerCount := 0

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if bounded.N == 0 {
				return ErrArchiveLimit
			}
			return ErrInvalidArchive
		}
		headerCount++
		if headerCount > limits.Entries {
			return ErrArchiveLimit
		}
		name, err := cleanArchivePath(header.Name, header.Typeflag == tar.TypeDir)
		if err != nil {
			return err
		}
		if _, exists := seenHeaders[name]; exists {
			return ErrInvalidArchive
		}
		seenHeaders[name] = struct{}{}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := ensureArchiveDirs(stageFD, name, directories, files, &nodes, limits.Entries); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > limits.FileBytes || header.Size > limits.TotalBytes-totalBytes {
				return ErrArchiveLimit
			}
			parent := path.Dir(name)
			if parent != "." {
				if err := ensureArchiveDirs(stageFD, parent, directories, files, &nodes, limits.Entries); err != nil {
					return err
				}
			}
			if _, exists := directories[name]; exists {
				return ErrInvalidArchive
			}
			fd, err := unix.Openat2(stageFD, name, &unix.OpenHow{
				Flags:   uint64(unix.O_WRONLY | unix.O_CREAT | unix.O_EXCL | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_NOCTTY),
				Mode:    0600,
				Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
			})
			if err != nil {
				return ErrInvalidArchive
			}
			f := os.NewFile(uintptr(fd), name)
			written, copyErr := io.CopyN(f, tr, header.Size)
			if copyErr == nil && written != header.Size {
				copyErr = io.ErrUnexpectedEOF
			}
			if copyErr == nil {
				copyErr = f.Sync()
			}
			closeErr := f.Close()
			if copyErr != nil {
				if bounded.N == 0 {
					return ErrArchiveLimit
				}
				return ErrInvalidArchive
			}
			if closeErr != nil {
				return closeErr
			}
			nodes++
			if nodes > limits.Entries {
				return ErrArchiveLimit
			}
			files[name] = struct{}{}
			totalBytes += header.Size
		default:
			return ErrInvalidArchive
		}
	}

	var tail zeroOnlyWriter
	if _, err := io.Copy(&tail, bounded); err != nil {
		return ErrInvalidArchive
	}
	if bounded.N == 0 {
		return ErrArchiveLimit
	}
	if tail.nonZero || source.Len() != 0 {
		return ErrInvalidArchive
	}
	return nil
}

type zeroOnlyWriter struct{ nonZero bool }

func (w *zeroOnlyWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b != 0 {
			w.nonZero = true
			return 0, ErrInvalidArchive
		}
	}
	return len(p), nil
}

func cleanArchivePath(name string, directory bool) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.Contains(name, "\\") || len(name) > 4096 || path.IsAbs(name) {
		return "", ErrInvalidArchive
	}
	if directory {
		name = strings.TrimSuffix(name, "/")
	}
	if name == "" || path.Clean(name) != name {
		return "", ErrInvalidArchive
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 255 {
			return "", ErrInvalidArchive
		}
	}
	return name, nil
}

func ensureArchiveDirs(stageFD int, name string, directories, files map[string]struct{}, nodes *int, maxNodes int) error {
	current := ""
	for _, part := range strings.Split(name, "/") {
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		if _, isFile := files[current]; isFile {
			return ErrInvalidArchive
		}
		if _, exists := directories[current]; exists {
			continue
		}
		if err := unix.Mkdirat(stageFD, current, 0700); err != nil {
			if !errors.Is(err, unix.EEXIST) {
				return ErrInvalidArchive
			}
			var stat unix.Stat_t
			if err := unix.Fstatat(stageFD, current, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFDIR {
				return ErrInvalidArchive
			}
		}
		*nodes++
		if *nodes > maxNodes {
			return ErrArchiveLimit
		}
		directories[current] = struct{}{}
	}
	return nil
}

func syncArchiveTree(dirFD int) error {
	dup, err := unix.Openat2(dirFD, ".", &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(dup), "archive-dir")
	names, readErr := f.Readdirnames(-1)
	closeErr := f.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	for _, name := range names {
		var stat unix.Stat_t
		if err := unix.Fstatat(dirFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		switch stat.Mode & unix.S_IFMT {
		case unix.S_IFDIR:
			child, err := unix.Openat2(dirFD, name, &unix.OpenHow{
				Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
				Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
			})
			if err != nil {
				return err
			}
			err = syncArchiveTree(child)
			if err == nil {
				err = unix.Fsync(child)
			}
			_ = unix.Close(child)
			if err != nil {
				return err
			}
		case unix.S_IFREG:
		default:
			return ErrInvalidArchive
		}
	}
	return unix.Fsync(dirFD)
}

func removeArchiveTree(dirFD int) error {
	dup, err := unix.Dup(dirFD)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(dup), "archive-dir")
	names, readErr := f.Readdirnames(-1)
	closeErr := f.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	for _, name := range names {
		var stat unix.Stat_t
		if err := unix.Fstatat(dirFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			return err
		}
		if stat.Mode&unix.S_IFMT == unix.S_IFDIR {
			child, err := unix.Openat2(dirFD, name, &unix.OpenHow{
				Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
				Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
			})
			if err != nil {
				return err
			}
			err = removeArchiveTree(child)
			_ = unix.Close(child)
			if err != nil {
				return err
			}
			if err := unix.Unlinkat(dirFD, name, unix.AT_REMOVEDIR); err != nil {
				return err
			}
		} else if err := unix.Unlinkat(dirFD, name, 0); err != nil {
			return err
		}
	}
	return nil
}

const archiveStagePrefix = ".tong-archive-stage-"

// CleanupStaleStages removes abandoned internal write and archive staging
// directories older than olderThan. It skips recent, aliased, wrongly-owned,
// permissive, and actively locked stages. Require a one-hour minimum age to
// avoid cleanup of a recently interrupted operation.
func (r *Root) CleanupStaleStages(olderThan time.Duration) (int, error) {
	if r == nil {
		return 0, ErrClosed
	}
	if olderThan < time.Hour {
		return 0, ErrInvalidLimit
	}
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
	scanFD, err := unix.Openat2(r.fd, ".", &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return 0, err
	}
	rootFile := os.NewFile(uintptr(scanFD), "file-root-scan")
	if rootFile == nil {
		_ = unix.Close(scanFD)
		return 0, ErrInvalidRoot
	}
	names, readErr := rootFile.Readdirnames(-1)
	closeErr := rootFile.Close()
	if readErr != nil {
		return 0, readErr
	}
	if closeErr != nil {
		return 0, closeErr
	}

	cutoff := time.Now().Add(-olderThan)
	removed := 0
	for _, name := range names {
		if !validInternalStageName(name) {
			continue
		}
		var before unix.Stat_t
		if err := unix.Fstatat(r.fd, name, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			return removed, err
		}
		if !staleInternalStage(before, cutoff) {
			continue
		}
		stageFD, err := unix.Openat2(r.fd, name, &unix.OpenHow{
			Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
			Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
		})
		if err != nil {
			if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ELOOP) {
				continue
			}
			return removed, err
		}
		var opened unix.Stat_t
		if err := unix.Fstat(stageFD, &opened); err != nil {
			_ = unix.Close(stageFD)
			return removed, err
		}
		if !sameInternalStage(before, opened) || !staleInternalStage(opened, cutoff) {
			_ = unix.Close(stageFD)
			continue
		}
		if err := unix.Flock(stageFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
			_ = unix.Close(stageFD)
			if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
				continue
			}
			return removed, err
		}
		var current unix.Stat_t
		statErr := unix.Fstatat(r.fd, name, &current, unix.AT_SYMLINK_NOFOLLOW)
		if statErr != nil || !sameInternalStage(opened, current) || !staleInternalStage(current, cutoff) {
			_ = unix.Flock(stageFD, unix.LOCK_UN)
			_ = unix.Close(stageFD)
			if statErr != nil && !errors.Is(statErr, unix.ENOENT) {
				return removed, statErr
			}
			continue
		}
		if err := removeArchiveTree(stageFD); err != nil {
			_ = unix.Flock(stageFD, unix.LOCK_UN)
			_ = unix.Close(stageFD)
			return removed, err
		}
		_ = unix.Flock(stageFD, unix.LOCK_UN)
		_ = unix.Close(stageFD)
		if err := unix.Unlinkat(r.fd, name, unix.AT_REMOVEDIR); err != nil {
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			return removed, err
		}
		removed++
	}
	if removed > 0 {
		if err := unix.Fsync(rootFD); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// CleanupStaleArchiveStages is retained for callers of the earlier prototype;
// it now also cleans abandoned create-only write stages.
func (r *Root) CleanupStaleArchiveStages(olderThan time.Duration) (int, error) {
	return r.CleanupStaleStages(olderThan)
}

func validInternalStageName(name string) bool {
	for _, prefix := range []string{writeStagePrefix, archiveStagePrefix} {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(name, prefix)
		if len(suffix) != 32 {
			return false
		}
		for _, char := range suffix {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
				return false
			}
		}
		return true
	}
	return false
}

func staleInternalStage(stat unix.Stat_t, cutoff time.Time) bool {
	modified := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)
	return stat.Mode&unix.S_IFMT == unix.S_IFDIR &&
		stat.Uid == uint32(unix.Geteuid()) &&
		stat.Mode&0077 == 0 &&
		!modified.After(cutoff)
}

func sameInternalStage(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino
}
