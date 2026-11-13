//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
)

// End-to-end component evidence with real files and pinned temporary roots;
// caller authentication, quota and production restore remain separate gates.
func TestTenantEncryptedRestoreOnDiskIntegration(t *testing.T) {
	a, dirA := newTestRoot(t)
	b, dirB := newTestRoot(t)
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": a, "tenant-b": b})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{dirA, dirB} {
		if err := os.WriteFile(filepath.Join(dir, "existing-marker"), []byte("unchanged"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	data := make([]byte, 140000)
	if _, err := rand.New(rand.NewSource(42)).Read(data); err != nil {
		t.Fatal(err)
	}
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "content.bin", Typeflag: tar.TypeReg}, data: data})
	if len(archive) <= backups.ChunkSize {
		t.Fatal("fixture must span multiple authenticated records")
	}
	key := bytes.Repeat([]byte{0x41}, 32)
	inputDir := t.TempDir()
	writeEnvelope := func(name, tenant string) string {
		t.Helper()
		path := filepath.Join(inputDir, name)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if err := backups.Encrypt(f, bytes.NewReader(archive), tenant, key, backups.Options{}); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return path
	}
	pathA := writeEnvelope("a.tpbk", "tenant-a")
	pathB := writeEnvelope("b.tpbk", "tenant-b")
	corrupt, err := os.ReadFile(pathA)
	if err != nil {
		t.Fatal(err)
	}
	corrupt[len(corrupt)/2] ^= 1
	corruptPath := filepath.Join(inputDir, "corrupt.tpbk")
	if err := os.WriteFile(corruptPath, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	invoke := func(tenant, destination, source string, wrappingKey []byte) error {
		t.Helper()
		f, err := os.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		return roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), tenant, destination, f, wrappingKey, ArchiveLimits{})
	}
	assertPristine := func() {
		t.Helper()
		for _, dir := range []string{dirA, dirB} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			if !reflect.DeepEqual(names, []string{"existing-marker"}) {
				t.Fatalf("denied restore changed root: %v", names)
			}
			marker, err := os.ReadFile(filepath.Join(dir, "existing-marker"))
			if err != nil || string(marker) != "unchanged" {
				t.Fatalf("marker changed: %v", err)
			}
		}
	}
	for _, tc := range []struct {
		tenant, path string
		key          []byte
	}{
		{"tenant-b", pathA, key}, {"tenant-a", pathA, bytes.Repeat([]byte{0x42}, 32)}, {"tenant-a", corruptPath, key},
	} {
		if err := invoke(tc.tenant, "site", tc.path, tc.key); !errors.Is(err, backups.ErrInvalidBackup) {
			t.Fatalf("denied restore: %v", err)
		}
		assertPristine()
	}
	for _, tc := range []struct{ tenant, path, dir string }{{"tenant-a", pathA, dirA}, {"tenant-b", pathB, dirB}} {
		if err := invoke(tc.tenant, "site", tc.path, key); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(tc.dir, "site", "content.bin"))
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("restored bytes differ: %v", err)
		}
		marker, err := os.ReadFile(filepath.Join(tc.dir, "existing-marker"))
		if err != nil || string(marker) != "unchanged" {
			t.Fatalf("marker changed: %v", err)
		}
	}
}
