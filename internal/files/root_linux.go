//go:build linux

// Package files contains unprivileged file-access primitives for the P2
// hosting prototype. It is deliberately not wired to the HTTP API or agent.
package files

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const MaxReadBytes int64 = 16 << 20

var (
	ErrInvalidRoot  = errors.New("invalid file root")
	ErrInvalidPath  = errors.New("invalid relative file path")
	ErrInvalidLimit = errors.New("invalid file size limit")
	ErrNotRegular   = errors.New("requested path is not a regular file")
	ErrTooLarge     = errors.New("file exceeds configured read limit")
	ErrClosed       = errors.New("file root is closed")
	ErrTenantDenied = errors.New("tenant file root unavailable")
)

// Root pins a trusted directory using an open descriptor. Close may be called
// concurrently with ReadFile; later reads fail with ErrClosed. The configured
// root and its parent path must be administrator-controlled.
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

func validRelativePath(name string) bool {
	if name == "" || strings.ContainsRune(name, '\x00') || path.IsAbs(name) || path.Clean(name) != name {
		return false
	}
	for _, component := range strings.Split(name, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}
