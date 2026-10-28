//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

func TestTenantRootsScopeCreateOnlyWrites(t *testing.T) {
	rootAPath := t.TempDir()
	rootBPath := t.TempDir()
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
	if err := roots.WriteNewForTenant("tenant-a", "created.txt", []byte("tenant-a")); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(rootAPath, "created.txt")); err != nil || string(got) != "tenant-a" {
		t.Fatalf("tenant A created content = %q, %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(rootBPath, "created.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tenant B unexpectedly received tenant A's file: %v", err)
	}
	for _, tenant := range []string{"", "unknown-tenant"} {
		if err := roots.WriteNewForTenant(tenant, "denied.txt", []byte("denied")); !errors.Is(err, ErrTenantDenied) {
			t.Fatalf("tenant %q write error = %v", tenant, err)
		}
	}
	if err := roots.WriteNewForTenant("tenant-a", "../tenant-b/escape.txt", []byte("escape")); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("tenant traversal write error = %v", err)
	}
}

func TestRemoveFileUnlinksEntriesWithoutFollowingSymlinks(t *testing.T) {
	r, root := newTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "remove.txt"), []byte("remove"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoveFile("remove.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, "remove.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed file still exists: %v", err)
	}

	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoveFile("link"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(outsideFile); err != nil || string(data) != "outside secret" {
		t.Fatalf("outside target changed to %q, %v", data, err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoveFile("directory"); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directory removal error = %v", err)
	}
	if err := unix.Mkfifo(filepath.Join(root, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.RemoveFile("pipe"); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("FIFO removal error = %v", err)
	}
	if err := r.RemoveFile("../escape"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("traversal removal error = %v", err)
	}
}

func TestRemoveFileSymlinkSwapNeverChangesOutsideTarget(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "changing-link")
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
	for i := 0; i < 500; i++ {
		err := r.RemoveFile("changing-link")
		if err != nil && !errors.Is(err, unix.ENOENT) {
			close(stop)
			wg.Wait()
			t.Fatalf("remove during symlink swap returned %v", err)
		}
	}
	close(stop)
	wg.Wait()
	if data, err := os.ReadFile(outsideFile); err != nil || string(data) != "outside secret" {
		t.Fatalf("outside target changed to %q, %v", data, err)
	}
}

func TestTenantRootsScopeFileRemoval(t *testing.T) {
	rootAPath := t.TempDir()
	rootBPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAPath, "a.txt"), []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBPath, "b.txt"), []byte("b"), 0600); err != nil {
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
	if err := roots.RemoveFileForTenant("tenant-a", "a.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(rootAPath, "a.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tenant A file remains: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(rootBPath, "b.txt")); err != nil || string(data) != "b" {
		t.Fatalf("tenant B file changed to %q, %v", data, err)
	}
	if err := roots.RemoveFileForTenant("unknown-tenant", "b.txt"); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("unknown tenant removal error = %v", err)
	}
	if err := roots.RemoveFileForTenant("tenant-a", "../tenant-b/b.txt"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("tenant traversal removal error = %v", err)
	}
}

func TestNewTenantRootsRejectsAliasedDirectories(t *testing.T) {
	dir := t.TempDir()
	first, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := NewTenantRoots(map[string]*Root{"tenant-a": first, "tenant-b": second}); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("aliased filesystem roots error = %v", err)
	}
	if _, err := NewTenantRoots(map[string]*Root{"tenant-a": first, "tenant-b": first}); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("same root pointer error = %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTenantRoots(map[string]*Root{"tenant-a": first, "tenant-b": second}); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("closed root mapping error = %v", err)
	}
}

type archiveTestEntry struct {
	header tar.Header
	data   []byte
}

func gzipTarForTest(t *testing.T, entries ...archiveTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		header := entry.header
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			header.Size = int64(len(entry.data))
		}
		if err := tw.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if len(entry.data) > 0 {
			if _, err := tw.Write(entry.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestExtractGzipTarNewPublishesCompletePrivateTree(t *testing.T) {
	r, root := newTestRoot(t)
	archive := gzipTarForTest(t,
		archiveTestEntry{header: tar.Header{Name: "public/", Typeflag: tar.TypeDir}},
		archiveTestEntry{header: tar.Header{Name: "public/index.html", Typeflag: tar.TypeReg}, data: []byte("complete")},
	)
	if err := r.ExtractGzipTarNew("unpacked", archive, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "unpacked", "public", "index.html"))
	if err != nil || string(content) != "complete" {
		t.Fatalf("extracted file = %q, %v", content, err)
	}
	info, err := os.Stat(filepath.Join(root, "unpacked", "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("extracted file permissions = %04o", info.Mode().Perm())
	}
	if err := r.ExtractGzipTarNew("unpacked", archive, ArchiveLimits{}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("existing target error = %v", err)
	}
	content, err = os.ReadFile(filepath.Join(root, "unpacked", "public", "index.html"))
	if err != nil || string(content) != "complete" {
		t.Fatalf("existing tree changed = %q, %v", content, err)
	}
	assertNoStagingDirectories(t, root)
}

func TestExtractGzipTarNewRejectsTraversalLinksDuplicatesAndSpecialFiles(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]archiveTestEntry{
		"traversal": {{header: tar.Header{Name: "../escape.txt", Typeflag: tar.TypeReg}, data: []byte("x")}},
		"absolute":  {{header: tar.Header{Name: "/etc/escape", Typeflag: tar.TypeReg}, data: []byte("x")}},
		"symlink":   {{header: tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: outsideFile}}},
		"hardlink":  {{header: tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "target"}}},
		"fifo":      {{header: tar.Header{Name: "pipe", Typeflag: tar.TypeFifo}}},
		"duplicate": {
			{header: tar.Header{Name: "same.txt", Typeflag: tar.TypeReg}, data: []byte("one")},
			{header: tar.Header{Name: "same.txt", Typeflag: tar.TypeReg}, data: []byte("two")},
		},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			archive := gzipTarForTest(t, entries...)
			if err := r.ExtractGzipTarNew("bad-"+name, archive, ArchiveLimits{}); !errors.Is(err, ErrInvalidArchive) {
				t.Fatalf("unsafe archive error = %v", err)
			}
			if _, err := os.Lstat(filepath.Join(root, "bad-"+name)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed extraction left target: %v", err)
			}
		})
	}
	if data, err := os.ReadFile(outsideFile); err != nil || string(data) != "outside secret" {
		t.Fatalf("outside file changed = %q, %v", data, err)
	}
	assertNoStagingDirectories(t, root)
}

func TestExtractGzipTarNewEnforcesCompressedFileAndEntryLimits(t *testing.T) {
	r, root := newTestRoot(t)
	oneFile := gzipTarForTest(t, archiveTestEntry{
		header: tar.Header{Name: "one.txt", Typeflag: tar.TypeReg},
		data:   []byte("four"),
	})
	if err := r.ExtractGzipTarNew("compressed-limit", oneFile, ArchiveLimits{CompressedBytes: 1}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("compressed size error = %v", err)
	}
	if err := r.ExtractGzipTarNew("file-limit", oneFile, ArchiveLimits{FileBytes: 3}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("per-file size error = %v", err)
	}
	if err := r.ExtractGzipTarNew("total-limit", oneFile, ArchiveLimits{TotalBytes: 3}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("total file data error = %v", err)
	}
	if err := r.ExtractGzipTarNew("expanded-limit", oneFile, ArchiveLimits{ExpandedBytes: 512}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("expanded size error = %v", err)
	}
	if err := r.ExtractGzipTarNew("invalid-limit", oneFile, ArchiveLimits{ExpandedBytes: MaxArchiveExpandedBytes + 1}); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("above-hard-limit error = %v", err)
	}
	twoFiles := gzipTarForTest(t,
		archiveTestEntry{header: tar.Header{Name: "one.txt", Typeflag: tar.TypeReg}, data: []byte("1")},
		archiveTestEntry{header: tar.Header{Name: "two.txt", Typeflag: tar.TypeReg}, data: []byte("2")},
	)
	if err := r.ExtractGzipTarNew("entry-limit", twoFiles, ArchiveLimits{Entries: 1}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("entry count error = %v", err)
	}
	for _, name := range []string{"compressed-limit", "file-limit", "total-limit", "expanded-limit", "invalid-limit", "entry-limit"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed extraction %q left target: %v", name, err)
		}
	}
	assertNoStagingDirectories(t, root)
}

func TestExtractGzipTarNewSymlinkSwapNeverChangesOutsideTarget(t *testing.T) {
	r, root := newTestRoot(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index.html", Typeflag: tar.TypeReg}, data: []byte("safe")})
	target := filepath.Join(root, "site")
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
		}
	}()
	<-ready
	for i := 0; i < 50; i++ {
		err := r.ExtractGzipTarNew("site", archive, ArchiveLimits{})
		if err != nil && !errors.Is(err, ErrAlreadyExists) && !errors.Is(err, unix.ENOENT) {
			close(stop)
			wg.Wait()
			t.Fatalf("extract during symlink swap returned %v", err)
		}
	}
	close(stop)
	wg.Wait()
	if data, err := os.ReadFile(outsideFile); err != nil || string(data) != "outside secret" {
		t.Fatalf("outside target changed = %q, %v", data, err)
	}
	assertNoStagingDirectories(t, root)
}

func TestExtractGzipTarNewConcurrentPublicationHasOneWinner(t *testing.T) {
	r, root := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index.html", Typeflag: tar.TypeReg}, data: []byte("winner")})
	const callers = 12
	start := make(chan struct{})
	results := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			results <- r.ExtractGzipTarNew("atomic-site", archive, ArchiveLimits{})
		}()
	}
	close(start)
	winners := 0
	for i := 0; i < callers; i++ {
		err := <-results
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("concurrent extraction error = %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful extractions = %d, want exactly one", winners)
	}
	data, err := os.ReadFile(filepath.Join(root, "atomic-site", "index.html"))
	if err != nil || string(data) != "winner" {
		t.Fatalf("published archive content = %q, %v", data, err)
	}
	assertNoStagingDirectories(t, root)
}

func TestTenantRootsScopeArchiveExtraction(t *testing.T) {
	rootAPath := t.TempDir()
	rootBPath := t.TempDir()
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
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index.html", Typeflag: tar.TypeReg}, data: []byte("tenant-a")})
	if err := roots.ExtractGzipTarNewForTenant("tenant-a", "site", archive, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(rootAPath, "site", "index.html")); err != nil || string(data) != "tenant-a" {
		t.Fatalf("tenant A archive content = %q, %v", data, err)
	}
	if _, err := os.Lstat(filepath.Join(rootBPath, "site")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tenant B unexpectedly received extracted tree: %v", err)
	}
	if err := roots.ExtractGzipTarNewForTenant("unknown", "site", archive, ArchiveLimits{}); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("unknown tenant extraction error = %v", err)
	}
	if err := roots.ExtractGzipTarNewForTenant("tenant-a", "../tenant-b/site", archive, ArchiveLimits{}); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("tenant traversal extraction error = %v", err)
	}
}
