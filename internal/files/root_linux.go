//go:build linux

// Package files contains unprivileged file-access primitives for the P2
// hosting prototype. It is deliberately not wired to the HTTP API or agent.
package files

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	MaxReadBytes     int64 = 16 << 20
	writeStagePrefix       = ".tong-stage-"
)

var (
	ErrInvalidRoot   = errors.New("invalid file root")
	ErrInvalidPath   = errors.New("invalid relative file path")
	ErrInvalidLimit  = errors.New("invalid file size limit")
	ErrNotRegular    = errors.New("requested path is not a regular file")
	ErrTooLarge      = errors.New("file exceeds configured read limit")
	ErrClosed        = errors.New("file root is closed")
	ErrTenantDenied  = errors.New("tenant file root unavailable")
	ErrAlreadyExists = errors.New("file already exists")
	ErrInvalidReader = errors.New("invalid file content reader")
)

// Root pins a trusted directory using an open descriptor. Close prevents new
// operations; already-started bounded writes retain a duplicate descriptor and
// may finish independently. The configured root and its parent path must be
// administrator-controlled.
type Root struct {
	mu   sync.RWMutex
	fd   int
	open bool
}

// OpenRoot opens a directory without following a symlink at its final path
// component. Unsupported or unsafe paths fail closed.
func OpenRoot(name string) (*Root, error) {
	if name == "" {
		return nil, ErrInvalidRoot
	}
	fd, err := unix.Open(name, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return &Root{fd: fd, open: true}, nil
}

// Close releases the pinned directory descriptor. Repeated calls are safe.
func (r *Root) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.open {
		return nil
	}
	fd := r.fd
	r.open = false
	return unix.Close(fd)
}

// ReadFile reads one regular file below the pinned root. openat2 resolution
// flags prevent traversal outside the root, symlink traversal, and procfs
// magic-link traversal. Kernels without openat2 support fail closed.
func (r *Root) ReadFile(relative string, maxBytes int64) ([]byte, error) {
	if r == nil {
		return nil, ErrClosed
	}
	if !validRelativePath(relative) {
		return nil, ErrInvalidPath
	}
	if maxBytes < 0 || maxBytes > MaxReadBytes {
		return nil, ErrInvalidLimit
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open {
		return nil, ErrClosed
	}
	fd, err := unix.Openat2(r.fd, relative, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NONBLOCK | unix.O_NOCTTY),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), relative)
	if f == nil {
		_ = unix.Close(fd)
		return nil, ErrInvalidPath
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotRegular
	}
	if info.Size() > maxBytes {
		return nil, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

// WriteNew atomically creates a new regular file below the pinned root with
// mode 0600. It never replaces an existing path, including a symlink. If the
// final directory sync fails, the file may already be present but durability
// is unconfirmed. Kernels without the required openat2/renameat2 support fail
// closed.
func (r *Root) WriteNew(relative string, content []byte) error {
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	if len(content) > int(MaxReadBytes) {
		return ErrTooLarge
	}
	return r.WriteNewFrom(relative, bytes.NewReader(content), int64(len(content)))
}

// WriteNewFrom stages a reader directly to a private file, using constant
// memory, and reads at most maxBytes+1 bytes to detect overflow. It publishes
// only after the stream finishes within its cap, via atomic no-replace rename.
// It duplicates the pinned root descriptor before reading, so Close prevents
// new operations but does not wait on an in-flight reader. maxBytes may be
// lower than MaxReadBytes but cannot exceed it. The source is not closed. This
// method does not authorize the caller or enforce a tenant-wide disk quota or
// request deadline.
func (r *Root) WriteNewFrom(relative string, src io.Reader, maxBytes int64) error {
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	if src == nil {
		return ErrInvalidReader
	}
	if maxBytes < 0 || maxBytes > MaxReadBytes {
		return ErrInvalidLimit
	}
	r.mu.RLock()
	if !r.open {
		r.mu.RUnlock()
		return ErrClosed
	}
	rootFD, err := unix.FcntlInt(uintptr(r.fd), unix.F_DUPFD_CLOEXEC, 0)
	r.mu.RUnlock()
	if err != nil {
		return err
	}
	defer unix.Close(rootFD)
	return writeNewFromRootFD(rootFD, relative, src, maxBytes)
}

func writeNewFromRootFD(rootFD int, relative string, src io.Reader, maxBytes int64) error {
	parent, base, err := openParentFD(rootFD, relative)
	if err != nil {
		return err
	}
	defer unix.Close(parent)

	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	stage := writeStagePrefix + hex.EncodeToString(random[:])
	if err := unix.Mkdirat(rootFD, stage, 0700); err != nil {
		return err
	}
	stageFD, err := unix.Openat2(rootFD, stage, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		_ = unix.Unlinkat(rootFD, stage, unix.AT_REMOVEDIR)
		return err
	}
	if err := unix.Flock(stageFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = unix.Close(stageFD)
		_ = unix.Unlinkat(rootFD, stage, unix.AT_REMOVEDIR)
		return err
	}
	defer func() {
		_ = unix.Unlinkat(stageFD, "payload", 0)
		_ = unix.Flock(stageFD, unix.LOCK_UN)
		_ = unix.Close(stageFD)
		_ = unix.Unlinkat(rootFD, stage, unix.AT_REMOVEDIR)
	}()
	const temp = "payload"
	fd, err := unix.Openat2(stageFD, temp, &unix.OpenHow{
		Flags:   uint64(unix.O_WRONLY | unix.O_CREAT | unix.O_EXCL | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_NOCTTY),
		Mode:    0600,
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), temp)
	if f == nil {
		_ = unix.Close(fd)
		return ErrInvalidPath
	}
	written, writeErr := io.Copy(f, io.LimitReader(src, maxBytes+1))
	if writeErr == nil && written > maxBytes {
		writeErr = ErrTooLarge
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := unix.Renameat2(stageFD, temp, parent, base, unix.RENAME_NOREPLACE); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return ErrAlreadyExists
		}
		return err
	}
	return unix.Fsync(parent)
}

// RemoveFile unlinks one regular file or symlink entry below the pinned root.
// It never follows a symlink; directories and special files are rejected. A
// successful unlink followed by a directory-sync error means durability is
// uncertain even though the path may already be absent.
func (r *Root) RemoveFile(relative string) error {
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
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

	var stat unix.Stat_t
	if err := unix.Fstatat(parent, base, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	switch stat.Mode & unix.S_IFMT {
	case unix.S_IFREG, unix.S_IFLNK:
		// unlinkat removes the directory entry and never dereferences symlinks.
	default:
		return ErrNotRegular
	}
	if err := unix.Unlinkat(parent, base, 0); err != nil {
		return err
	}
	return unix.Fsync(parent)
}

func (r *Root) openParent(relative string) (int, string, error) {
	return openParentFD(r.fd, relative)
}

func openParentFD(rootFD int, relative string) (int, string, error) {
	fd, err := unix.Openat2(rootFD, path.Dir(relative), &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		return -1, "", err
	}
	return fd, path.Base(relative), nil
}

func validRelativePath(name string) bool {
	if name == "" || strings.ContainsRune(name, '\x00') || path.IsAbs(name) || path.Clean(name) != name {
		return false
	}
	components := strings.Split(name, "/")
	if strings.HasPrefix(components[0], writeStagePrefix) || strings.HasPrefix(components[0], archiveStagePrefix) {
		return false
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}
