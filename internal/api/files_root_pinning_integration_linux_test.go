//go:build linux

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// This exercises steady-state API requests after test-only eager attachment.
// It does not cover startup attachment, TOCTOU during root opening, or
// production Files wiring.
func TestFilesAPIRootRemainsPinnedAfterTenantDirectoryRename(t *testing.T) {
	s, _, _, tokenA, csrfA := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)

	movedRoot := rootA + "-moved"
	if _, err := os.Lstat(movedRoot); !os.IsNotExist(err) {
		t.Fatalf("moved root path already exists or cannot be checked: %v", err)
	}
	if err := os.Rename(rootA, movedRoot); err != nil {
		t.Fatalf("rename attached tenant root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(movedRoot); err != nil {
			t.Errorf("remove moved tenant root: %v", err)
		}
	})
	if err := os.Mkdir(rootA, 0700); err != nil {
		t.Fatalf("recreate tenant root pathname: %v", err)
	}
	if err := os.Mkdir(filepath.Join(rootA, "public"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"private.txt":                  "replacement-a-secret",
		"replacement-only.txt":         "replacement-a-only",
		"public/index.html":            "replacement-public",
		"public/replacement-only.html": "replacement-only-public",
	} {
		if err := os.WriteFile(filepath.Join(rootA, filepath.FromSlash(path)), []byte(content), 0600); err != nil {
			t.Fatalf("write replacement sentinel %q: %v", path, err)
		}
	}

	assertFile := func(path, want string) {
		t.Helper()
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("file %q = %q, err=%v; want %q", path, got, err, want)
		}
	}
	assertAbsent := func(path string) {
		t.Helper()
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("path %q should be absent, stat err=%v", path, err)
		}
	}
	assertNames := func(dir string, want []string) {
		t.Helper()
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read directory %q: %v", dir, err)
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		if !reflect.DeepEqual(names, want) {
			t.Fatalf("directory %q has entries %v, want %v", dir, names, want)
		}
	}
	assertSentinelsUnchanged := func() {
		t.Helper()
		assertFile(filepath.Join(movedRoot, "private.txt"), "tenant-a-secret")
		assertFile(filepath.Join(movedRoot, "public", "index.html"), "safe")
		assertNames(rootA, []string{"private.txt", "public", "replacement-only.txt"})
		assertNames(filepath.Join(rootA, "public"), []string{"index.html", "replacement-only.html"})
		assertFile(filepath.Join(rootA, "private.txt"), "replacement-a-secret")
		assertFile(filepath.Join(rootA, "replacement-only.txt"), "replacement-a-only")
		assertFile(filepath.Join(rootA, "public", "index.html"), "replacement-public")
		assertFile(filepath.Join(rootA, "public", "replacement-only.html"), "replacement-only-public")
		assertFile(filepath.Join(rootB, "private.txt"), "tenant-b-secret")
		assertNames(rootB, []string{"private.txt"})
	}

	h := s.Handler()
	listing := sessionGET(h, "/api/v1/files", tokenA)
	if listing.Code != http.StatusOK {
		t.Fatalf("root list status=%d body=%s", listing.Code, listing.Body.String())
	}
	var listed struct {
		Entries []TenantFileEntry `json:"entries"`
	}
	if err := json.Unmarshal(listing.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	wantEntries := []TenantFileEntry{
		{Name: "escape", Kind: "symlink"},
		{Name: "private.txt", Kind: "file", Size: int64(len("tenant-a-secret"))},
		{Name: "public", Kind: "directory"},
	}
	if !reflect.DeepEqual(listed.Entries, wantEntries) {
		t.Fatalf("root list entries=%#v, want %#v", listed.Entries, wantEntries)
	}
	for path, want := range map[string]string{
		"private.txt":       "tenant-a-secret",
		"public/index.html": "safe",
	} {
		response := sessionGET(h, "/api/v1/files/content?path="+path, tokenA)
		if response.Code != http.StatusOK || response.Body.String() != want {
			t.Fatalf("read %q status=%d body=%q, want %q", path, response.Code, response.Body.String(), want)
		}
	}

	payload := "pinned-root-created"
	quota := newTestFileQuota(int64(len(payload)))
	s.FileQuota = quota
	created := fileWriteCall(h, "/api/v1/files/content?path=created-after-rename.txt", tokenA, csrfA, payload)
	if created.Code != http.StatusCreated {
		t.Fatalf("write status=%d body=%s", created.Code, created.Body.String())
	}
	var writeResult struct {
		Bytes int64 `json:"bytes"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &writeResult); err != nil || writeResult.Bytes != int64(len(payload)) {
		t.Fatalf("write response=%s bytes=%d err=%v", created.Body.String(), writeResult.Bytes, err)
	}
	pinnedWrite := filepath.Join(movedRoot, "created-after-rename.txt")
	assertFile(pinnedWrite, payload)
	assertAbsent(filepath.Join(rootA, "created-after-rename.txt"))
	assertAbsent(filepath.Join(rootB, "created-after-rename.txt"))
	readCreated := sessionGET(h, "/api/v1/files/content?path=created-after-rename.txt", tokenA)
	if readCreated.Code != http.StatusOK || readCreated.Body.String() != payload {
		t.Fatalf("read created file status=%d body=%q, want %q", readCreated.Code, readCreated.Body.String(), payload)
	}
	if used, reserved := quota.snapshot("tenant-a"); used != int64(len(payload)) || reserved != 0 {
		t.Fatalf("quota after write used=%d reserved=%d", used, reserved)
	}
	if used, reserved := quota.snapshot("tenant-b"); used != 0 || reserved != 0 {
		t.Fatalf("tenant B quota changed after tenant A write: used=%d reserved=%d", used, reserved)
	}
	assertSentinelsUnchanged()

	deleted := fileDeleteCall(h, "/api/v1/files/content?path=created-after-rename.txt", tokenA, csrfA)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	assertAbsent(pinnedWrite)
	assertAbsent(filepath.Join(rootA, "created-after-rename.txt"))
	assertAbsent(filepath.Join(rootB, "created-after-rename.txt"))
	if used, reserved := quota.snapshot("tenant-a"); used != 0 || reserved != 0 {
		t.Fatalf("quota after delete used=%d reserved=%d", used, reserved)
	}
	if used, reserved := quota.snapshot("tenant-b"); used != 0 || reserved != 0 {
		t.Fatalf("tenant B quota changed after tenant A delete: used=%d reserved=%d", used, reserved)
	}
	assertSentinelsUnchanged()
}
