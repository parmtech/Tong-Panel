//go:build linux

package files

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func newTestRoot(t *testing.T) (*Root, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, dir
}

func TestReadFileBeneathRootAndRejectsUnsafePaths(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "inside.txt"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadFile("inside.txt", 16)
	if err != nil || string(got) != "inside" {
		t.Fatalf("read inside file = %q, %v", got, err)
	}
	for _, name := range []string{"../secret.txt", "./inside.txt", "sub/../inside.txt", "/etc/passwd", "", "a//b", "escape"} {
		t.Run(name, func(t *testing.T) {
			if data, err := r.ReadFile(name, 1024); err == nil {
				t.Fatalf("unsafe path %q returned %q", name, data)
			}
		})
	}
}

func TestReadFileEnforcesRegularFilesAndByteLimit(t *testing.T) {
	r, root := newTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "large.txt"), []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadFile("large.txt", 4); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized read error = %v", err)
	}
	if _, err := r.ReadFile("large.txt", MaxReadBytes+1); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("excessive configured limit error = %v", err)
	}
	if _, err := r.ReadFile(".", 100); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("directory-like path error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadFile("directory", 100); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directory read error = %v", err)
	}
}

func TestReadFileSymlinkSwapNeverReadsOutside(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "changing.txt")
	if err := os.WriteFile(target, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = os.Remove(target)
			_ = os.Symlink(outsideFile, target)
			_ = os.Remove(target)
			_ = os.WriteFile(target, []byte("safe"), 0600)
		}
	}()
	for i := 0; i < 1000; i++ {
		data, err := r.ReadFile("changing.txt", 64)
		if err == nil && string(data) != "safe" {
			close(stop)
			wg.Wait()
			t.Fatalf("symlink swap exposed outside data %q", data)
		}
	}
	close(stop)
	wg.Wait()
}

func TestOpenRootRefusesFinalSymlinkAndCloseIsSafe(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRoot(link); err == nil {
		t.Fatal("symlink root was accepted")
	}
	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
	if _, err := r.ReadFile("file", 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("read after close error = %v", err)
	}
}
