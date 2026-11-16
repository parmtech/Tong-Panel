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
