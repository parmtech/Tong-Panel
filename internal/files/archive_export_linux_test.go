//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/backups"
	"golang.org/x/sys/unix"
)

func TestEncryptedSiteExportRoundTripAndTenantBinding(t *testing.T) {
	rootA, dirA := newTestRoot(t)
	rootB, dirB := newTestRoot(t)
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dirA, "site", "nested", "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "site", "nested", "content.txt"), []byte("private site fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "sentinel"), []byte("tenant-b"), 0600); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{6}, 32)
	var encrypted bytes.Buffer
	if err := roots.ExportEncryptedGzipTarForTenant(context.Background(), "tenant-a", "site", &encrypted, key, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted.Bytes(), []byte("private site fixture")) {
		t.Fatal("ciphertext exposed plaintext")
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-b", "restored", bytes.NewReader(encrypted.Bytes()), key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("cross-tenant restore: %v", err)
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-a", "restored", bytes.NewReader(encrypted.Bytes()), key, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dirA, "restored", "nested", "content.txt"))
	if err != nil || string(data) != "private site fixture" {
		t.Fatalf("restored: %q %v", data, err)
	}
	if info, err := os.Stat(filepath.Join(dirA, "restored", "nested", "empty")); err != nil || !info.IsDir() {
		t.Fatalf("empty directory: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dirB, "sentinel")); err != nil || string(data) != "tenant-b" {
		t.Fatalf("foreign tenant changed: %q %v", data, err)
	}
}

type shortExportWriter struct{}

func (shortExportWriter) Write([]byte) (int, error) { return 0, nil }
func TestEncryptedSiteExportRejectsUnsafeInputsAndLimits(t *testing.T) {
	key := bytes.Repeat([]byte{8}, 32)
	for _, kind := range []string{"symlink", "hardlink", "fifo", "reserved-stage", "ciphertext-stage"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := newTestRoot(t)
			site := filepath.Join(dir, "site")
			if err := os.Mkdir(site, 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(site, "link")); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				p := filepath.Join(site, "file")
				if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Link(p, filepath.Join(site, "second")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := unix.Mkfifo(filepath.Join(site, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			case "reserved-stage":
				if err := os.Mkdir(filepath.Join(site, ".tong-archive-stage-secret"), 0700); err != nil {
					t.Fatal(err)
				}
			case "ciphertext-stage":
				if err := os.WriteFile(filepath.Join(site, ".tong-ciphertext-stage-secret"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var dst bytes.Buffer
			err := root.ExportEncryptedGzipTar(context.Background(), "site", &dst, "tenant-a", key, ArchiveLimits{})
			if !errors.Is(err, ErrInvalidArchive) || dst.Len() != 0 {
				t.Fatalf("unsafe source exported: %v bytes=%d", err, dst.Len())
			}
		})
	}
	root, dir := newTestRoot(t)
	if err := os.Mkdir(filepath.Join(dir, "site"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "site", "large"), []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "site", "extra"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, limits := range []ArchiveLimits{{FileBytes: 4}, {TotalBytes: 4}, {CompressedBytes: 1}, {ExpandedBytes: 1}, {Entries: 1}} {
		var dst bytes.Buffer
		if err := root.ExportEncryptedGzipTar(context.Background(), "site", &dst, "tenant-a", key, limits); !errors.Is(err, ErrArchiveLimit) || dst.Len() != 0 {
			t.Fatalf("limit %+v err=%v bytes=%d", limits, err, dst.Len())
		}
	}
	if err := root.ExportEncryptedGzipTar(context.Background(), "site", shortExportWriter{}, "tenant-a", key, ArchiveLimits{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short writer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var dst bytes.Buffer
	if err := root.ExportEncryptedGzipTar(ctx, "site", &dst, "tenant-a", key, ArchiveLimits{}); !errors.Is(err, context.Canceled) || dst.Len() != 0 {
		t.Fatalf("cancel: %v", err)
	}
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": root})
	if err != nil {
		t.Fatal(err)
	}
	if err := roots.ExportEncryptedGzipTarForTenant(context.Background(), "unknown", "site", &dst, key, ArchiveLimits{}); !errors.Is(err, ErrTenantDenied) || dst.Len() != 0 {
		t.Fatalf("unknown tenant: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "site"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := root.ExportEncryptedGzipTar(context.Background(), "link", &dst, "tenant-a", key, ArchiveLimits{}); err == nil || dst.Len() != 0 {
		t.Fatalf("symlink source accepted: %v", err)
	}
	if err := root.ExportEncryptedGzipTar(context.Background(), "site", &dst, "tenant-a", []byte("short"), ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidKey) || dst.Len() != 0 {
		t.Fatalf("invalid key: %v", err)
	}
}

type mutationHookWriter struct {
	dst   io.Writer
	hook  func()
	fired bool
}

func (w *mutationHookWriter) Write(p []byte) (int, error) {
	if !w.fired {
		w.fired = true
		w.hook()
	}
	return w.dst.Write(p)
}
func TestSiteExportDetectsSourceReplacementDuringRead(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "file")
	if err := os.WriteFile(filename, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	hook := &mutationHookWriter{dst: io.Discard, hook: func() {
		if err := os.Rename(filename, filepath.Join(dir, "old")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte("replaced"), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	tw := tar.NewWriter(hook)
	state := exportArchiveState{ctx: context.Background(), tar: tw, limits: ArchiveLimits{CompressedBytes: MaxArchiveCompressedBytes, ExpandedBytes: MaxArchiveExpandedBytes, FileBytes: MaxArchiveFileBytes, TotalBytes: MaxArchiveTotalBytes, Entries: MaxArchiveEntries}}
	if err := state.addDirectory(fd, "", 0); !errors.Is(err, ErrUsageSnapshotDirty) {
		t.Fatalf("source replacement accepted: %v", err)
	}
}

type blockedExportWriter struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *blockedExportWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return len(p), nil
}
func TestExportDoesNotHoldRootLocksWhileWritingCiphertext(t *testing.T) {
	root, dir := newTestRoot(t)
	if err := os.Mkdir(filepath.Join(dir, "site"), 0700); err != nil {
		t.Fatal(err)
	}
	writer := &blockedExportWriter{entered: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- root.ExportEncryptedGzipTar(context.Background(), "site", writer, "tenant-a", bytes.Repeat([]byte{9}, 32), ArchiveLimits{})
	}()
	select {
	case <-writer.entered:
	case <-time.After(5 * time.Second):
		close(writer.release)
		t.Fatal("export did not reach destination")
	}
	written := make(chan error, 1)
	go func() { written <- root.WriteNew("other", []byte("ok")) }()
	select {
	case err := <-written:
		if err != nil {
			close(writer.release)
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		close(writer.release)
		t.Fatal("destination held root mutation lock")
	}
	close(writer.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
