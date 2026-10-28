//go:build linux

package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// LinuxFileMutationBarrier coordinates all quota-sensitive API mutations
// across Tong Panel processes. Its directory must be pre-created by the
// service administrator and must not be writable by group or other users.
type LinuxFileMutationBarrier struct {
	directory string
}

func NewLinuxFileMutationBarrier(directory string) (*LinuxFileMutationBarrier, error) {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, errors.New("file mutation lock directory must be a clean absolute path")
	}
	var st unix.Stat_t
	if err := unix.Lstat(directory, &st); err != nil {
		return nil, fmt.Errorf("inspect file mutation lock directory: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(unix.Geteuid()) || st.Mode&0022 != 0 {
		return nil, fmt.Errorf("file mutation lock directory is unsafe (mode=%#o owner=%d euid=%d)", st.Mode, st.Uid, unix.Geteuid())
	}
	return &LinuxFileMutationBarrier{directory: directory}, nil
}

func (b *LinuxFileMutationBarrier) Acquire(ctx context.Context, tenantID string, exclusive bool) (func(), error) {
	if b == nil || ctx == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(tenantID) != tenantID {
		return nil, errors.New("invalid file mutation barrier request")
	}
	digest := sha256.Sum256([]byte(tenantID))
	name := hex.EncodeToString(digest[:]) + ".lock"
	fd, err := unix.Openat(unix.AT_FDCWD, filepath.Join(b.directory, name), unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, fmt.Errorf("open tenant file mutation lock: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("inspect tenant file mutation lock: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(unix.Geteuid()) || st.Mode&0077 != 0 || st.Nlink != 1 {
		_ = unix.Close(fd)
		return nil, errors.New("tenant file mutation lock has unsafe type, owner, mode or link count")
	}
	operation := unix.LOCK_SH
	if exclusive {
		operation = unix.LOCK_EX
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = unix.Close(fd)
			return nil, err
		}
		err = unix.Flock(fd, operation|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EINTR) {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("acquire tenant file mutation lock: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = unix.Close(fd)
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = unix.Flock(fd, unix.LOCK_UN)
			_ = unix.Close(fd)
		})
	}, nil
}

// WithExclusiveFileMaintenance holds the tenant's exclusive barrier across a
// caller-supplied scan and reconciliation transaction.
func WithExclusiveFileMaintenance(ctx context.Context, barrier FileMutationBarrier, tenantID string, fn func() error) error {
	if barrier == nil || fn == nil {
		return ErrFilesUnavailable
	}
	release, err := barrier.Acquire(ctx, tenantID, true)
	if err != nil {
		return fmt.Errorf("acquire exclusive tenant file maintenance barrier: %w", err)
	}
	defer release()
	return fn()
}
