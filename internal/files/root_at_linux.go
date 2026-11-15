//go:build linux

package files

import (
	"strings"

	"golang.org/x/sys/unix"
)

// OpenRootAt opens one directory component relative to an already-pinned,
// trusted parent descriptor, without traversing links or crossing mounts. It
// compares the opened inode with observations of the name before and after
// opening and closes the descriptor on any mismatch. The caller must retain a
// trusted, stable parent and prevent same-identity mutation: these checks do
// not establish authorization, ownership, quota, or a race-free filesystem
// fence against an actor who can rename the child before the first observation.
func OpenRootAt(parentFD int, name string) (*Root, error) {
	if parentFD < 0 || name == "" || name == "." || name == ".." || len(name) > 255 || strings.ContainsAny(name, "/\\\x00") {
		return nil, ErrInvalidRoot
	}
	var before unix.Stat_t
	if err := unix.Fstatat(parentFD, name, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil || before.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil, ErrInvalidRoot
	}
	fd, err := unix.Openat2(parentFD, name, &unix.OpenHow{Flags: uint64(unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW), Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV)})
	if err != nil {
		return nil, ErrInvalidRoot
	}
	var opened, after unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil {
		_ = unix.Close(fd)
		return nil, ErrInvalidRoot
	}
	if err := unix.Fstatat(parentFD, name, &after, unix.AT_SYMLINK_NOFOLLOW); err != nil || opened.Mode&unix.S_IFMT != unix.S_IFDIR || after.Mode&unix.S_IFMT != unix.S_IFDIR || !sameInternalStage(before, opened) || !sameInternalStage(opened, after) {
		_ = unix.Close(fd)
		return nil, ErrInvalidRoot
	}
	return &Root{fd: fd, open: true}, nil
}
