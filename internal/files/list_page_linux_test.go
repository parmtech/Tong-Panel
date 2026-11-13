//go:build linux

package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestListDirPageBoundedTenantScopedAndNoFollow(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for i := 0; i < 205; i++ {
		if err := os.WriteFile(filepath.Join(a, fmt.Sprintf("item-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(b, "private-b"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(b, filepath.Join(a, "link")); err != nil {
		t.Fatal(err)
	}
	ra, err := OpenRoot(a)
	if err != nil {
		t.Fatal(err)
	}
	defer ra.Close()
	rb, err := OpenRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()
	roots, err := NewTenantRoots(map[string]*Root{"a": ra, "b": rb})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	after := ""
	for {
		entries, next, err := roots.ListDirPageForTenant("a", "", after, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if seen[e.Name] || e.Name == "private-b" {
				t.Fatalf("duplicate or cross-tenant entry: %#v", e)
			}
			seen[e.Name] = true
			if e.Name == "link" && e.Kind != EntrySymlink {
				t.Fatal("link followed")
			}
		}
		if next == "" {
			break
		}
		if next <= after {
			t.Fatal("cursor did not advance")
		}
		after = next
	}
	if len(seen) != 206 {
		t.Fatalf("saw %d entries", len(seen))
	}
	if _, _, err := roots.ListDirPageForTenant("a", "link", "", 10); err == nil {
		t.Fatal("followed symlink")
	}
	if _, _, err := roots.ListDirPageForTenant("missing", "", "", 10); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("missing tenant: %v", err)
	}
	if _, _, err := ra.ListDirPage("", "../bad", 10); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("invalid cursor: %v", err)
	}
}

func TestListDirPageBoundaryAndConcurrentChanges(t *testing.T) {
	dir := t.TempDir()
	root, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for i := 0; i < MaxPageScanEntries; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%05d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	entries, next, err := root.ListDirPage("", "", 1)
	if err != nil || len(entries) != 1 || next != "00000" {
		t.Fatalf("at cap: %v %q %v", entries, next, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.ListDirPage("", "", 1); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("above cap: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "extra")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "00001")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "00000-new"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	entries, _, err = root.ListDirPage("", next, 1)
	if err != nil || len(entries) != 1 || entries[0].Name != "00000-new" {
		t.Fatalf("fresh scan after mutation: %#v %v", entries, err)
	}
}
