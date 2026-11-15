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
