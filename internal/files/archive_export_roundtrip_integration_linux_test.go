//go:build linux

package files

import (
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

// This is a temporary local component round trip, not an authorized job,
// consistent live-site snapshot, database restore, or production backup.
func TestEncryptedTenantArchiveDiskRoundTrip(t *testing.T) {
	source, sourceDir := newTestRoot(t)
	destA, dirA := newTestRoot(t)
	destB, dirB := newTestRoot(t)
	sources, err := NewTenantRoots(map[string]*Root{"tenant-a": source})
	if err != nil {
		t.Fatal(err)
	}
	destinations, err := NewTenantRoots(map[string]*Root{"tenant-a": destA, "tenant-b": destB})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "site", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(sourceDir, "site", "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 140000)
	if _, err := rand.New(rand.NewSource(81)).Read(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "site", "nested", "data.bin"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "site", "index.html"), []byte("synthetic site marker"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{dirA, dirB} {
		if err := os.WriteFile(filepath.Join(dir, "sentinel"), []byte("unchanged"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	key := bytes.Repeat([]byte{0x51}, 32)
	artifact := filepath.Join(t.TempDir(), "fixture.tpbk")
	file, err := os.OpenFile(artifact, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := sources.ExportEncryptedGzipTarForTenant(context.Background(), "tenant-a", "site", file, key, ArchiveLimits{}); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	sealed, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(sealed) <= backups.ChunkSize || bytes.Contains(sealed, []byte("synthetic site marker")) {
		t.Fatal("expected multi-record ciphertext without plaintext marker")
	}
	corrupt := bytes.Clone(sealed)
	corrupt[len(corrupt)/2] ^= 1
	corruptPath := filepath.Join(filepath.Dir(artifact), "corrupt.tpbk")
	if err := os.WriteFile(corruptPath, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	restore := func(tenant, input string, k []byte) error {
		t.Helper()
		f, err := os.Open(input)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		return destinations.RestoreEncryptedGzipTarNewForTenant(context.Background(), tenant, "restored", f, k, ArchiveLimits{})
	}
	assertPristine := func() {
		t.Helper()
		for _, dir := range []string{dirA, dirB} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			if !reflect.DeepEqual(names, []string{"sentinel"}) {
				t.Fatalf("denied restore modified root: %v", names)
			}
			value, err := os.ReadFile(filepath.Join(dir, "sentinel"))
			if err != nil || string(value) != "unchanged" {
				t.Fatalf("sentinel changed: %v", err)
			}
		}
	}
	for _, tc := range []struct {
		tenant, file string
		key          []byte
	}{
		{"tenant-b", artifact, key}, {"tenant-a", artifact, bytes.Repeat([]byte{0x52}, 32)}, {"tenant-a", corruptPath, key},
	} {
		if err := restore(tc.tenant, tc.file, tc.key); !errors.Is(err, backups.ErrInvalidBackup) {
			t.Fatalf("denied restore: %v", err)
		}
		assertPristine()
	}
	if err := restore("tenant-a", artifact, key); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dirA, "restored", "nested", "data.bin"))
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("nested file mismatch: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(dirA, "restored", "index.html"))
	if err != nil || string(index) != "synthetic site marker" {
		t.Fatalf("index mismatch: %v", err)
	}
	if info, err := os.Stat(filepath.Join(dirA, "restored", "empty")); err != nil || !info.IsDir() {
		t.Fatalf("empty directory missing: %v", err)
	}
	entries, err := os.ReadDir(dirB)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
		t.Fatalf("other tenant changed: %v %v", entries, err)
	}
}
