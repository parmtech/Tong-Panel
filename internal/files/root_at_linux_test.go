//go:build linux

package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestOpenRootAtUsesPinnedParentAfterPathReplacement(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "child", "marker"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	parentFD, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(parentFD)
	if err := os.Rename(parent, filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "child", "marker"), []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRootAt(parentFD, "child")
	if err != nil {
		t.Fatal(err)
	}
	got, err := root.ReadFile("marker", 128)
	if err != nil || string(got) != "original" {
		t.Fatalf("resolved replacement: %q %v", got, err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(parentFD, &st); err != nil {
		t.Fatalf("helper closed caller parent: %v", err)
	}
}
func TestOpenRootAtRejectsUnsafeNamesAndSymlinks(t *testing.T) {
	parent := t.TempDir()
	outside := t.TempDir()
	parentFD, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(parentFD)
	if err := os.Symlink(outside, filepath.Join(parent, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "file"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", ".", "..", "../outside", "nested/dir", `x\y`, "link", "file", "missing"} {
		if root, err := OpenRootAt(parentFD, name); !errors.Is(err, ErrInvalidRoot) || root != nil {
			if root != nil {
				root.Close()
			}
			t.Fatalf("unsafe name %q root=%v err=%v", name, root, err)
		}
	}
	if root, err := OpenRootAt(-1, "child"); !errors.Is(err, ErrInvalidRoot) || root != nil {
		t.Fatalf("invalid parent root=%v err=%v", root, err)
	}
}

type openRootAtFixture struct {
	parentPath string
	childPath  string
	parentFD   int
	fd         int
	before     unix.Stat_t
}

func newOpenRootAtFixture(t *testing.T) openRootAtFixture {
	t.Helper()
	parentPath := t.TempDir()
	childPath := filepath.Join(parentPath, "child")
	if err := os.Mkdir(childPath, 0700); err != nil {
		t.Fatal(err)
	}
	parentFD, err := unix.Open(parentPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unix.Close(parentFD) })

	var before unix.Stat_t
	if err := unix.Fstatat(parentFD, "child", &before, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Openat2(parentFD, "child", &unix.OpenHow{
		Flags:   uint64(unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW),
		Resolve: uint64(unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var st unix.Stat_t
		if unix.Fstat(fd, &st) == nil {
			_ = unix.Close(fd)
		}
	})
	return openRootAtFixture{
		parentPath: parentPath,
		childPath:  childPath,
		parentFD:   parentFD,
		fd:         fd,
		before:     before,
	}
}

func assertOpenedRootAtRejectedAndDescriptors(t *testing.T, fixture openRootAtFixture, name string, before unix.Stat_t) {
	t.Helper()
	root, err := validateOpenedRootAt(fixture.parentFD, name, fixture.fd, before)
	if !errors.Is(err, ErrInvalidRoot) || root != nil {
		t.Fatalf("validateOpenedRootAt() = (%v, %v), want (nil, %v)", root, err, ErrInvalidRoot)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fixture.fd, &st); !errors.Is(err, unix.EBADF) {
		t.Fatalf("child descriptor after rejection: Fstat() = %v, want %v", err, unix.EBADF)
	}
	if err := unix.Fstat(fixture.parentFD, &st); err != nil {
		t.Fatalf("validation closed caller parent descriptor: %v", err)
	}
}

func TestValidateOpenedRootAtClosesDescriptorAfterFinalFstatatFailure(t *testing.T) {
	fixture := newOpenRootAtFixture(t)
	if err := os.Rename(fixture.childPath, filepath.Join(fixture.parentPath, "moved")); err != nil {
		t.Fatal(err)
	}
	assertOpenedRootAtRejectedAndDescriptors(t, fixture, "child", fixture.before)
}

func TestValidateOpenedRootAtClosesDescriptorOnBeforeOpenedIdentityMismatch(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*unix.Stat_t)
	}{
		{name: "device", mutate: func(st *unix.Stat_t) { st.Dev ^= 1 }},
		{name: "inode", mutate: func(st *unix.Stat_t) { st.Ino ^= 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newOpenRootAtFixture(t)
			before := fixture.before
			tc.mutate(&before)
			assertOpenedRootAtRejectedAndDescriptors(t, fixture, "child", before)
		})
	}
}

func TestValidateOpenedRootAtClosesDescriptorAfterOpenedAfterIdentityMismatch(t *testing.T) {
	fixture := newOpenRootAtFixture(t)
	if err := os.Rename(fixture.childPath, filepath.Join(fixture.parentPath, "moved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(fixture.childPath, 0700); err != nil {
		t.Fatal(err)
	}
	assertOpenedRootAtRejectedAndDescriptors(t, fixture, "child", fixture.before)
}

func TestValidateOpenedRootAtTransfersDescriptorOwnershipOnSuccess(t *testing.T) {
	fixture := newOpenRootAtFixture(t)
	root, err := validateOpenedRootAt(fixture.parentFD, "child", fixture.fd, fixture.before)
	if err != nil || root == nil {
		t.Fatalf("validateOpenedRootAt() = (%v, %v), want non-nil root and nil error", root, err)
	}
	if root.fd != fixture.fd || !root.open {
		t.Fatalf("returned root does not own opened descriptor: fd=%d open=%t, want fd=%d open=true", root.fd, root.open, fixture.fd)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fixture.fd, &st); err != nil {
		t.Fatalf("successful validation closed transferred descriptor: %v", err)
	}
	if err := unix.Fstat(fixture.parentFD, &st); err != nil {
		t.Fatalf("validation closed caller parent descriptor: %v", err)
	}
	if err := root.Close(); err != nil {
		t.Fatalf("closing returned root: %v", err)
	}
	if err := unix.Fstat(fixture.fd, &st); !errors.Is(err, unix.EBADF) {
		t.Fatalf("child descriptor after Root.Close(): Fstat() = %v, want %v", err, unix.EBADF)
	}
	if err := unix.Fstat(fixture.parentFD, &st); err != nil {
		t.Fatalf("closing returned root closed caller parent descriptor: %v", err)
	}
}

// The Fstat(fd) failure branch is not directly exercised: Openat2 returns a
// valid descriptor in these tests, and syscall injection solely for that kernel
// error path would broaden the seam without improving ownership coverage.

func TestCheckOwnerPrivateDirectory0700(t *testing.T) {
	specialBits := os.ModeSetuid | os.ModeSetgid | os.ModeSticky

	var nilRoot *Root
	if err := nilRoot.CheckOwnerPrivateDirectory0700(); !errors.Is(err, ErrClosed) {
		t.Fatalf("nil root error = %v, want %v", err, ErrClosed)
	}

	closedPath := t.TempDir()
	if err := os.Chmod(closedPath, 0700); err != nil {
		t.Fatal(err)
	}
	closedRoot, err := OpenRoot(closedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := closedRoot.Close(); err != nil {
		t.Fatal(err)
	}
	if err := closedRoot.CheckOwnerPrivateDirectory0700(); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed root error = %v, want %v", err, ErrClosed)
	}

	t.Run("exactly private", func(t *testing.T) {
		path := t.TempDir()
		if err := os.Chmod(path, 0700); err != nil {
			t.Fatal(err)
		}
		root, err := OpenRoot(path)
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close()
		if err := root.CheckOwnerPrivateDirectory0700(); err != nil {
			t.Fatalf("CheckOwnerPrivateDirectory0700() = %v, want nil", err)
		}
	})

	for _, tc := range []struct {
		name string
		mode os.FileMode
	}{
		{name: "group permissions", mode: 0750},
		{name: "other permissions", mode: 0701},
		{name: "setuid bit", mode: os.ModeSetuid | 0700},
		{name: "setgid bit", mode: os.ModeSetgid | 0700},
		{name: "sticky bit", mode: os.ModeSticky | 0700},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir()
			if err := os.Chmod(path, tc.mode); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			gotMode := info.Mode()
			gotSpecialBits := gotMode & specialBits
			wantSpecialBits := tc.mode & specialBits
			if gotMode.Perm() != tc.mode.Perm() || gotSpecialBits != wantSpecialBits {
				t.Fatalf(
					"temporary directory mode = %v (special bits %#x), want permissions %v and special bits %#x",
					gotMode, gotSpecialBits, tc.mode.Perm(), wantSpecialBits,
				)
			}
			root, err := OpenRoot(path)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := root.CheckOwnerPrivateDirectory0700(); !errors.Is(err, ErrInvalidRoot) {
				t.Fatalf("CheckOwnerPrivateDirectory0700() = %v, want %v", err, ErrInvalidRoot)
			}
		})
	}
}
