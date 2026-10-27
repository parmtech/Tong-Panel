//go:build linux

package files

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
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
	pipe := filepath.Join(root, "pipe")
	if err := unix.Mkfifo(pipe, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadFile("pipe", 100); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("FIFO read error = %v", err)
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
	safeTemp := filepath.Join(root, "changing.safe.tmp")
	if err := os.WriteFile(target, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	swapped := make(chan struct{}, 1)
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
			if err := os.Symlink(outsideFile, target); err == nil {
				select {
				case swapped <- struct{}{}:
				default:
				}
			}
			_ = os.Remove(target)
			_ = os.Remove(safeTemp)
			if err := os.WriteFile(safeTemp, []byte("safe"), 0600); err == nil {
				_ = os.Rename(safeTemp, target)
			}
		}
	}()
	<-swapped
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
	var zero Root
	if err := zero.Close(); err != nil {
		t.Fatalf("closing zero value failed: %v", err)
	}
	if _, err := zero.ReadFile("file", 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("zero-value root read error = %v", err)
	}
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

func TestTenantRootsKeepReadsWithinAssignedTenant(t *testing.T) {
	rootAPath := t.TempDir()
	rootBPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAPath, "account.txt"), []byte("tenant-a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBPath, "account.txt"), []byte("tenant-b-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	rootA, err := OpenRoot(rootAPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rootA.Close()
	rootB, err := OpenRoot(rootBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rootB.Close()
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	data, err := roots.ReadFileForTenant("tenant-a", "account.txt", 64)
	if err != nil || string(data) != "tenant-a" {
		t.Fatalf("tenant A read = %q, %v", data, err)
	}
	for _, tenant := range []string{"", "unknown-tenant"} {
		if data, err := roots.ReadFileForTenant(tenant, "account.txt", 64); !errors.Is(err, ErrTenantDenied) {
			t.Fatalf("tenant %q unexpectedly read %q, err=%v", tenant, data, err)
		}
	}
	if _, err := roots.ReadFileForTenant("tenant-a", "../tenant-b/account.txt", 64); err == nil {
		t.Fatal("tenant A traversed to tenant B")
	}
	if _, err := NewTenantRoots(map[string]*Root{"": rootA}); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("empty tenant mapping error = %v", err)
	}
}

func TestWriteNewCreatesAtomicallyWithoutReplacing(t *testing.T) {
	r, root := newTestRoot(t)
	if err := os.Mkdir(filepath.Join(root, "uploads"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteNew("uploads/created.txt", []byte("complete upload")); err != nil {
		t.Fatal(err)
	}
	data, err := r.ReadFile("uploads/created.txt", 64)
	if err != nil || string(data) != "complete upload" {
		t.Fatalf("created file read = %q, %v", data, err)
	}
	info, err := os.Stat(filepath.Join(root, "uploads", "created.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("created file mode = %04o", info.Mode().Perm())
	}
	if err := os.WriteFile(filepath.Join(root, "uploads", "existing.txt"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteNew("uploads/existing.txt", []byte("overwrite attempt")); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("existing target error = %v", err)
	}
	data, err = r.ReadFile("uploads/existing.txt", 64)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("existing file changed = %q, %v", data, err)
	}
	if err := r.WriteNew("../escape.txt", []byte("escape")); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("traversal write error = %v", err)
	}
	if err := r.WriteNew("uploads/large.txt", make([]byte, int(MaxReadBytes)+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized write error = %v", err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "linked-dir")); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteNew("linked-dir/escaped.txt", []byte("escape")); err == nil {
		t.Fatal("write through symlinked parent succeeded")
	}
	assertNoStagingDirectories(t, root)
}

func TestConcurrentWriteNewHasOneWinner(t *testing.T) {
	r, root := newTestRoot(t)
	const writers = 16
	start := make(chan struct{})
	results := make(chan error, writers)
	values := make(map[byte]struct{}, writers)
	for i := 0; i < writers; i++ {
		value := byte('A' + i)
		values[value] = struct{}{}
		go func() {
			<-start
			results <- r.WriteNew("winner.txt", []byte{value})
		}()
	}
	close(start)
	winners := 0
	for i := 0; i < writers; i++ {
		err := <-results
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("concurrent write returned unexpected error: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful writers = %d, want exactly one", winners)
	}
	data, err := r.ReadFile("winner.txt", 1)
	if err != nil || len(data) != 1 {
		t.Fatalf("winner file = %q, %v", data, err)
	}
	if _, ok := values[data[0]]; !ok {
		t.Fatalf("winner content %q was not written by a participant", data)
	}
	assertNoStagingDirectories(t, root)
}

func assertNoStagingDirectories(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tong-stage-") {
			t.Errorf("staging directory leaked after operation: %q", entry.Name())
		}
	}
}

func TestWriteNewSymlinkSwapNeverChangesOutsideFile(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "upload.txt")
	stop := make(chan struct{})
	ready := make(chan struct{}, 1)
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
			if err := os.Symlink(outsideFile, target); err == nil {
				select {
				case ready <- struct{}{}:
				default:
				}
			}
		}
	}()
	<-ready
	for i := 0; i < 100; i++ {
		err := r.WriteNew("upload.txt", []byte("new safe file"))
		if err != nil && !errors.Is(err, ErrAlreadyExists) {
			close(stop)
			wg.Wait()
			t.Fatalf("write during symlink swap returned %v", err)
		}
	}
	close(stop)
	wg.Wait()
	data, err := os.ReadFile(outsideFile)
	if err != nil || string(data) != "outside secret" {
		t.Fatalf("outside target changed to %q, %v", data, err)
	}
}

func TestListDirReturnsBoundedSortedNoFollowMetadata(t *testing.T) {
	r, root := newTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("1234"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "a-dir"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "z-link")); err != nil {
		t.Fatal(err)
	}

	entries, err := r.ListDir("", 3)
	if !errors.Is(err, ErrTooManyEntries) || entries != nil {
		t.Fatalf("over-limit listing = %#v, %v", entries, err)
	}
	entries, err = r.ListDir("", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("entry count = %d, want 4", len(entries))
	}
	if entries[0].Name != "a-dir" || entries[0].Kind != EntryDir {
		t.Fatalf("first entry = %#v", entries[0])
	}
	if entries[1].Name != "b.txt" || entries[1].Kind != EntryFile || entries[1].Size != 4 {
		t.Fatalf("file entry = %#v", entries[1])
	}
	if entries[2].Name != "c.txt" || entries[2].Kind != EntryFile || entries[2].Size != 1 {
		t.Fatalf("second file entry = %#v", entries[2])
	}
	if entries[3].Name != "z-link" || entries[3].Kind != EntrySymlink || entries[3].Size != 0 {
		t.Fatalf("symlink entry = %#v", entries[3])
	}
	if _, err := r.ListDir("z-link", 10); err == nil {
		t.Fatal("listing through a symlink succeeded")
	}
	if _, err := r.ListDir("../", 10); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("traversal listing error = %v", err)
	}
	if _, err := r.ListDir("", 0); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("zero listing limit error = %v", err)
	}
	if _, err := r.ListDir("", MaxListEntries+1); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("excessive listing limit error = %v", err)
	}
}

func TestListDirSymlinkSwapNeverListsOutsideEntries(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside-secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "changing")
	stop := make(chan struct{})
	ready := make(chan struct{}, 1)
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
			if err := os.Symlink(outside, target); err == nil {
				select {
				case ready <- struct{}{}:
				default:
				}
			}
			_ = os.Remove(target)
		}
	}()
	<-ready
	for i := 0; i < 1000; i++ {
		entries, err := r.ListDir("changing", 100)
		if err == nil {
			for _, entry := range entries {
				if entry.Name == "outside-secret.txt" {
					close(stop)
					wg.Wait()
					t.Fatal("listing through swapped symlink exposed outside entry")
				}
			}
		}
	}
	close(stop)
	wg.Wait()
}

func TestTenantRootsKeepDirectoryListingsWithinAssignedRoot(t *testing.T) {
	rootAPath := t.TempDir()
	rootBPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAPath, "tenant-a.txt"), []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBPath, "tenant-b-secret.txt"), []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	rootA, err := OpenRoot(rootAPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rootA.Close()
	rootB, err := OpenRoot(rootBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rootB.Close()
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := roots.ListDirForTenant("tenant-a", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "tenant-a.txt" {
		t.Fatalf("tenant A listing = %#v", entries)
	}
	for _, tenant := range []string{"", "unknown-tenant"} {
		if entries, err := roots.ListDirForTenant(tenant, "", 10); !errors.Is(err, ErrTenantDenied) {
			t.Fatalf("tenant %q unexpectedly listed %#v, err=%v", tenant, entries, err)
		}
	}
	if _, err := roots.ListDirForTenant("tenant-a", "../tenant-b", 10); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("tenant traversal listing error = %v", err)
	}
}
