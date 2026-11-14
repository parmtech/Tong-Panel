//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
	"golang.org/x/sys/unix"
)

func encryptedFixture(t *testing.T, archive []byte, tenant string, key []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := backups.Encrypt(&out, bytes.NewReader(archive), tenant, key, backups.Options{}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestEncryptedRestoreAuthenticatesBeforePublishing(t *testing.T) {
	r, dir := newTestRoot(t)
	key := bytes.Repeat([]byte{1}, 32)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index.html", Typeflag: tar.TypeReg}, data: []byte("restored site")})
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 1
	for _, tc := range []struct {
		name, tenant string
		data, key    []byte
	}{
		{"wrong-tenant", "tenant-b", sealed, key}, {"wrong-key", "tenant-a", sealed, bytes.Repeat([]byte{2}, 32)},
		{"tampered", "tenant-a", tampered, key}, {"truncated", "tenant-a", sealed[:len(sealed)-1], key},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := r.RestoreEncryptedGzipTarNew(context.Background(), tc.name, bytes.NewReader(tc.data), tc.tenant, tc.key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
				t.Fatalf("error: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(dir, tc.name)); !os.IsNotExist(err) {
				t.Fatalf("destination exists: %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("unexpected pre-auth stage: %#v %v", entries, err)
			}
		})
	}
	if err := r.RestoreEncryptedGzipTarNew(context.Background(), "site", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "site", "index.html"))
	if err != nil || string(got) != "restored site" {
		t.Fatalf("restored: %q %v", got, err)
	}
	if err := r.RestoreEncryptedGzipTarNew(context.Background(), "site", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestEncryptedRestoreRejectsInvalidArchiveAndLimit(t *testing.T) {
	r, dir := newTestRoot(t)
	key := bytes.Repeat([]byte{3}, 32)
	bad := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "../outside", Typeflag: tar.TypeReg}, data: []byte("escape")})
	encrypted := encryptedFixture(t, bad, "tenant-a", key)
	if err := r.RestoreEncryptedGzipTarNew(context.Background(), "site", bytes.NewReader(encrypted), "tenant-a", key, ArchiveLimits{}); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("traversal: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "site")); !os.IsNotExist(err) {
		t.Fatal("unsafe archive published")
	}
	if err := r.RestoreEncryptedGzipTarNew(context.Background(), "site", bytes.NewReader(encrypted), "tenant-a", key, ArchiveLimits{CompressedBytes: 1}); !errors.Is(err, backups.ErrBackupLimit) {
		t.Fatalf("compressed limit: %v", err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("staging remained: %#v %v", entries, err)
	}
}

type cancelBeforePublish struct {
	context.Context
	checks int
}

func (c *cancelBeforePublish) Err() error {
	c.checks++
	if c.checks >= 2 {
		return context.Canceled
	}
	return nil
}
func TestArchiveCancellationAndPublishedUncertain(t *testing.T) {
	r, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index", Typeflag: tar.TypeReg}, data: []byte("ok")})
	ctx := &cancelBeforePublish{Context: context.Background()}
	if err := r.extractGzipTarNewContext(ctx, "canceled", archive, ArchiveLimits{}, false, unix.Fsync); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "canceled")); !os.IsNotExist(err) {
		t.Fatal("canceled archive published")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("staging remained: %#v %v", entries, err)
	}
	if err := r.extractGzipTarNewContext(context.Background(), "published", archive, ArchiveLimits{}, false, func(int) error { return unix.EIO }); !errors.Is(err, ErrPublishedDurabilityUnconfirmed) || !errors.Is(err, unix.EIO) {
		t.Fatalf("post-publish error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "published", "index")); err != nil {
		t.Fatalf("published tree absent: %v", err)
	}
	count := 0
	err = r.extractGzipTarNewContext(context.Background(), "published-root-sync", archive, ArchiveLimits{}, false, func(fd int) error {
		count++
		if count == 2 {
			return unix.EIO
		}
		return unix.Fsync(fd)
	})
	if !errors.Is(err, ErrPublishedDurabilityUnconfirmed) || !errors.Is(err, unix.EIO) || count != 2 {
		t.Fatalf("root sync error: %v calls=%d", err, count)
	}
	if _, err := os.Stat(filepath.Join(dir, "published-root-sync", "index")); err != nil {
		t.Fatalf("published tree absent: %v", err)
	}
}

type cancelingReader struct{ cancel context.CancelFunc }

func (r cancelingReader) Read([]byte) (int, error) { r.cancel(); return 0, io.EOF }
func TestEncryptedRestoreCancellationDoesNotCreateStage(t *testing.T) {
	root, dir := newTestRoot(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := root.RestoreEncryptedGzipTarNew(ctx, "site", cancelingReader{cancel}, "tenant-a", bytes.Repeat([]byte{1}, 32), ArchiveLimits{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("canceled read created stage: %#v %v", entries, err)
	}
}

// observingBackupReader checks the private root while the encrypted stream is
// still being consumed, including the final tag and EOF/trailing-byte probe.
type observingBackupReader struct {
	reader      *bytes.Reader
	root        string
	sawTerminal bool
	sawEOF      bool
	observedErr error
}

func (r *observingBackupReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	entries, readErr := os.ReadDir(r.root)
	if readErr != nil {
		r.observedErr = readErr
	} else if len(entries) != 0 {
		r.observedErr = errors.New("restore created a stage before input authentication completed")
	}
	if r.reader.Len() == 0 {
		r.sawTerminal = true
	}
	if errors.Is(err, io.EOF) {
		r.sawEOF = true
	}
	return n, err
}

func TestEncryptedRestoreMultiChunkAuthenticatesBeforeCreatingStage(t *testing.T) {
	// Pseudorandom bytes keep gzip from collapsing the archive into one record.
	payload := make([]byte, 3*backups.ChunkSize)
	if _, err := rand.New(rand.NewSource(17)).Read(payload); err != nil {
		t.Fatal(err)
	}
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "asset.bin", Typeflag: tar.TypeReg}, data: payload})
	if len(archive) <= 2*backups.ChunkSize {
		t.Fatalf("fixture compressed to %d bytes; expected multiple records", len(archive))
	}
	key := bytes.Repeat([]byte{0x31}, 32)
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	badFinal := bytes.Clone(sealed)
	badFinal[len(badFinal)-1] ^= 1
	trailing := append(bytes.Clone(sealed), 0x42)
	for _, tc := range []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"valid", sealed, false}, {"final-tag-tamper", badFinal, true}, {"trailing-byte", trailing, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := newTestRoot(t)
			observed := &observingBackupReader{reader: bytes.NewReader(tc.data), root: dir}
			err := root.RestoreEncryptedGzipTarNew(context.Background(), "restored", observed, "tenant-a", key, ArchiveLimits{})
			if observed.observedErr != nil || !observed.sawTerminal {
				t.Fatalf("authentication read observation: terminal=%t error=%v", observed.sawTerminal, observed.observedErr)
			}
			if tc.wantErr {
				if !errors.Is(err, backups.ErrInvalidBackup) {
					t.Fatalf("invalid input: %v", err)
				}
				entries, readErr := os.ReadDir(dir)
				if readErr != nil || len(entries) != 0 {
					t.Fatalf("invalid backup changed root: %v %v", entries, readErr)
				}
			} else {
				if !observed.sawEOF {
					t.Fatal("valid stream was not checked through EOF")
				}
				if err != nil {
					t.Fatal(err)
				}
				got, readErr := os.ReadFile(filepath.Join(dir, "restored", "asset.bin"))
				if readErr != nil || !bytes.Equal(got, payload) {
					t.Fatalf("valid restore mismatch: %v", readErr)
				}
			}
		})
	}
}

type observedRestoreReader struct {
	data  *bytes.Reader
	reads int
}

func (r *observedRestoreReader) Read(p []byte) (int, error) { r.reads++; return r.data.Read(p) }

func TestTenantEncryptedRestoreBindsRegisteredRootAndEnvelope(t *testing.T) {
	rootA, dirA := newTestRoot(t)
	rootB, dirB := newTestRoot(t)
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{0x24}, 32)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "index.html", Typeflag: tar.TypeReg}, data: []byte("tenant-a private site")})
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	source := &observedRestoreReader{data: bytes.NewReader(sealed)}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "missing", "site", source, key, ArchiveLimits{}); !errors.Is(err, ErrTenantDenied) || source.reads != 0 {
		t.Fatalf("unknown tenant consumed input or accepted: reads=%d err=%v", source.reads, err)
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-b", "site", bytes.NewReader(sealed), key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("cross-tenant backup accepted: %v", err)
	}
	for _, dir := range []string{dirA, dirB} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("failed operation changed root: %v %v", entries, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source = &observedRestoreReader{data: bytes.NewReader(sealed)}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(ctx, "tenant-a", "site", source, key, ArchiveLimits{}); !errors.Is(err, context.Canceled) || source.reads != 0 {
		t.Fatalf("canceled operation consumed input: reads=%d err=%v", source.reads, err)
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-a", "site", bytes.NewReader(sealed), key, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dirA, "site", "index.html"))
	if err != nil || string(got) != "tenant-a private site" {
		t.Fatalf("tenant A restore: %q %v", got, err)
	}
	if entries, err := os.ReadDir(dirB); err != nil || len(entries) != 0 {
		t.Fatalf("tenant B changed: %v %v", entries, err)
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-a", "site", bytes.NewReader(sealed), key, ArchiveLimits{}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("overwrote destination: %v", err)
	}
}
