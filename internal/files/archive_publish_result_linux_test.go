//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/backups"
	"golang.org/x/sys/unix"
)

func TestUnknownPublicationStageSurvivesStaleCleanup(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "asset", Typeflag: tar.TypeReg}, data: []byte("preserve for reconciliation")})
	var result ArchivePublishResult
	err := root.extractGzipTarNewWithResult(context.Background(), "site", archive, ArchiveLimits{}, false, unix.Fsync,
		func(int, string, int, string, uint) error { return unix.EIO }, &result)
	if !errors.Is(err, ErrPublicationOutcomeUnknown) || result.State != ArchivePublicationUnknown {
		t.Fatalf("expected unresolved publication: %+v %v", result, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), archiveStagePrefix) {
		t.Fatalf("expected one retained stage: %v %v", entries, err)
	}
	stage := filepath.Join(dir, entries[0].Name())
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stage, old, old); err != nil {
		t.Fatal(err)
	}
	removed, err := root.CleanupStaleStages(time.Hour)
	if err != nil || removed != 0 {
		t.Fatalf("cleanup = %d, %v", removed, err)
	}
	data, err := os.ReadFile(filepath.Join(stage, "asset"))
	if err != nil || string(data) != "preserve for reconciliation" {
		t.Fatalf("unresolved stage payload changed: %q %v", data, err)
	}
}

func TestRestoreResultReportsOnlyPublishedLogicalFileBytes(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t,
		archiveTestEntry{header: tar.Header{Name: "nested/a", Typeflag: tar.TypeReg}, data: []byte("abc")},
		archiveTestEntry{header: tar.Header{Name: "nested/empty", Typeflag: tar.TypeReg}},
		archiveTestEntry{header: tar.Header{Name: "second", Typeflag: tar.TypeReg}, data: []byte("de")},
	)
	key := bytes.Repeat([]byte{4}, 32)
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	outcome, err := root.RestoreEncryptedGzipTarNewWithResult(context.Background(), "site", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{})
	if err != nil || outcome.State != ArchivePublished || outcome.LogicalBytes != 5 {
		t.Fatalf("result=%+v err=%v", outcome, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "site", "nested", "empty")); err != nil {
		t.Fatal(err)
	}
	emptyArchive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "empty", Typeflag: tar.TypeReg}})
	emptySealed := encryptedFixture(t, emptyArchive, "tenant-a", key)
	emptyResult, err := root.RestoreEncryptedGzipTarNewWithResult(context.Background(), "empty-site", bytes.NewReader(emptySealed), "tenant-a", key, ArchiveLimits{})
	if err != nil || emptyResult.State != ArchivePublished || emptyResult.LogicalBytes != 0 {
		t.Fatalf("empty published result=%+v err=%v", emptyResult, err)
	}
	outcome, err = root.RestoreEncryptedGzipTarNewWithResult(context.Background(), "site", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{})
	if !errors.Is(err, ErrAlreadyExists) || outcome.State != ArchiveNotPublished || outcome.LogicalBytes != 0 {
		t.Fatalf("existing target result=%+v err=%v", outcome, err)
	}
	outcome, err = root.RestoreEncryptedGzipTarNewWithResult(context.Background(), "limited", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{TotalBytes: 4})
	if !errors.Is(err, ErrArchiveLimit) || outcome.State != ArchiveNotPublished || outcome.LogicalBytes != 0 {
		t.Fatalf("limit result=%+v err=%v", outcome, err)
	}
	bad := bytes.Clone(sealed)
	bad[len(bad)-1] ^= 1
	outcome, err = root.RestoreEncryptedGzipTarNewWithResult(context.Background(), "invalid", bytes.NewReader(bad), "tenant-a", key, ArchiveLimits{})
	if !errors.Is(err, backups.ErrInvalidBackup) || outcome.State != ArchiveNotPublished || outcome.LogicalBytes != 0 {
		t.Fatalf("auth result=%+v err=%v", outcome, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome, err = root.RestoreEncryptedGzipTarNewWithResult(ctx, "canceled", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{})
	if !errors.Is(err, context.Canceled) || outcome.State != ArchiveNotPublished || outcome.LogicalBytes != 0 {
		t.Fatalf("cancel result=%+v err=%v", outcome, err)
	}
}

func TestArchiveResultRetainsCountAfterDurabilityFailure(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "asset", Typeflag: tar.TypeReg}, data: []byte("1234567")})
	for _, failAt := range []int{1, 2} {
		var result ArchivePublishResult
		calls := 0
		dest := []string{"first", "second"}[failAt-1]
		err := root.extractGzipTarNewWithResult(context.Background(), dest, archive, ArchiveLimits{}, false, func(fd int) error {
			calls++
			if calls == failAt {
				return unix.EIO
			}
			return unix.Fsync(fd)
		}, unix.Renameat2, &result)
		if !errors.Is(err, ErrPublishedDurabilityUnconfirmed) || result.State != ArchivePublished || result.LogicalBytes != 7 {
			t.Fatalf("sync %d result=%+v err=%v", failAt, result, err)
		}
		if _, err := os.Stat(filepath.Join(dir, dest, "asset")); err != nil {
			t.Fatalf("published file absent: %v", err)
		}
	}
}

func TestArchiveResultPreservesDestinationOnAmbiguousRename(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "private", Typeflag: tar.TypeReg}, data: []byte("payload")})
	var result ArchivePublishResult
	renameThenError := func(oldFD int, old string, newFD int, newName string, flags uint) error {
		if err := unix.Renameat2(oldFD, old, newFD, newName, flags); err != nil {
			return err
		}
		return unix.EIO
	}
	err := root.extractGzipTarNewWithResult(context.Background(), "site", archive, ArchiveLimits{}, false, unix.Fsync, renameThenError, &result)
	if !errors.Is(err, unix.EIO) || !errors.Is(err, ErrPublicationOutcomeUnknown) || result.State != ArchivePublicationUnknown || result.LogicalBytes != 7 {
		t.Fatalf("ambiguous rename result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "site", "private"))
	if err != nil || string(data) != "payload" {
		t.Fatalf("ambiguous published content destroyed: %q %v", data, err)
	}
	var noMove ArchivePublishResult
	err = root.extractGzipTarNewWithResult(context.Background(), "other", archive, ArchiveLimits{}, false, unix.Fsync, func(int, string, int, string, uint) error { return unix.EIO }, &noMove)
	if !errors.Is(err, unix.EIO) || !errors.Is(err, ErrPublicationOutcomeUnknown) || noMove.State != ArchivePublicationUnknown || noMove.LogicalBytes != 7 {
		t.Fatalf("failed rename result=%+v err=%v", noMove, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "other")); !os.IsNotExist(err) {
		t.Fatal("failed rename created destination")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), archiveStagePrefix) {
			info, err := entry.Info()
			if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
				t.Fatalf("unsafe retained stage: %v %v", info, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("ambiguous failed rename should retain private stage for reconciliation")
	}
}

func TestArchiveResultPreservesDestinationOnAmbiguousExistenceError(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "asset", Typeflag: tar.TypeReg}, data: []byte("keep")})
	var result ArchivePublishResult
	renameThenExists := func(a int, b string, c int, d string, flags uint) error {
		if err := unix.Renameat2(a, b, c, d, flags); err != nil {
			return err
		}
		return unix.EEXIST
	}
	err := root.extractGzipTarNewWithResult(context.Background(), "site", archive, ArchiveLimits{}, false, unix.Fsync, renameThenExists, &result)
	if !errors.Is(err, ErrAlreadyExists) || !errors.Is(err, ErrPublicationOutcomeUnknown) || result.State != ArchivePublicationUnknown || result.LogicalBytes != 4 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "site", "asset"))
	if err != nil || string(data) != "keep" {
		t.Fatalf("destination damaged: %q %v", data, err)
	}
}

func TestTenantRestoreResultDeniesUnknownAndBindsAccounting(t *testing.T) {
	rootA, dirA := newTestRoot(t)
	rootB, dirB := newTestRoot(t)
	roots, err := NewTenantRoots(map[string]*Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "item", Typeflag: tar.TypeReg}, data: []byte("12345")})
	key := bytes.Repeat([]byte{7}, 32)
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	unread := &observedRestoreReader{data: bytes.NewReader(sealed)}
	result, err := roots.RestoreEncryptedGzipTarNewForTenantWithResult(context.Background(), "unknown", "site", unread, key, ArchiveLimits{})
	if !errors.Is(err, ErrTenantDenied) || result.State != ArchiveNotPublished || result.LogicalBytes != 0 || unread.reads != 0 {
		t.Fatalf("unknown result=%+v reads=%d err=%v", result, unread.reads, err)
	}
	result, err = roots.RestoreEncryptedGzipTarNewForTenantWithResult(context.Background(), "tenant-b", "site", bytes.NewReader(sealed), key, ArchiveLimits{})
	if !errors.Is(err, backups.ErrInvalidBackup) || result.State != ArchiveNotPublished || result.LogicalBytes != 0 {
		t.Fatalf("cross-tenant result=%+v err=%v", result, err)
	}
	if entries, err := os.ReadDir(dirB); err != nil || len(entries) != 0 {
		t.Fatalf("foreign root changed: %v %v", entries, err)
	}
	result, err = roots.RestoreEncryptedGzipTarNewForTenantWithResult(context.Background(), "tenant-a", "site", bytes.NewReader(sealed), key, ArchiveLimits{})
	if err != nil || result.State != ArchivePublished || result.LogicalBytes != 5 {
		t.Fatalf("success result=%+v err=%v", result, err)
	}
	if data, err := os.ReadFile(filepath.Join(dirA, "site", "item")); err != nil || string(data) != "12345" {
		t.Fatalf("content %q %v", data, err)
	}
}

type cancelOnThirdCheck struct {
	context.Context
	checks int
}

func (c *cancelOnThirdCheck) Err() error {
	c.checks++
	if c.checks >= 3 {
		return context.Canceled
	}
	return nil
}
func TestArchiveResultCancellationBeforeRename(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "item", Typeflag: tar.TypeReg}, data: []byte("x")})
	ctx := &cancelOnThirdCheck{Context: context.Background()}
	var result ArchivePublishResult
	renamed := false
	err := root.extractGzipTarNewWithResult(ctx, "site", archive, ArchiveLimits{}, false, unix.Fsync, func(int, string, int, string, uint) error { renamed = true; return nil }, &result)
	if !errors.Is(err, context.Canceled) || renamed || result.State != ArchiveNotPublished || result.LogicalBytes != 0 {
		t.Fatalf("cancel result=%+v renamed=%t err=%v", result, renamed, err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("canceled stage: %v %v", entries, err)
	}
}
