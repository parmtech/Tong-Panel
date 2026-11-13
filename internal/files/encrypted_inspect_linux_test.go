//go:build linux

package files

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
)

func TestVerifyEncryptedSiteArchiveSharesRestoreValidation(t *testing.T) {
	key := bytes.Repeat([]byte{0x36}, 32)
	valid := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "nested/a", Typeflag: tar.TypeReg}, data: []byte("hello")}, archiveTestEntry{header: tar.Header{Name: "empty/", Typeflag: tar.TypeDir}})
	cases := []struct {
		name    string
		archive []byte
		want    error
	}{
		{"valid", valid, nil},
		{"traversal", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "../escape", Typeflag: tar.TypeReg}, data: []byte("x")}), ErrInvalidArchive},
		{"absolute", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "/escape", Typeflag: tar.TypeReg}, data: []byte("x")}), ErrInvalidArchive},
		{"symlink", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "outside"}}), ErrInvalidArchive},
		{"hardlink", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "outside"}}), ErrInvalidArchive},
		{"duplicate", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "x", Typeflag: tar.TypeReg}, data: []byte("1")}, archiveTestEntry{header: tar.Header{Name: "x", Typeflag: tar.TypeReg}, data: []byte("2")}), ErrInvalidArchive},
		{"file-directory-conflict", gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "x", Typeflag: tar.TypeReg}, data: []byte("1")}, archiveTestEntry{header: tar.Header{Name: "x/y", Typeflag: tar.TypeReg}, data: []byte("2")}), ErrInvalidArchive},
		{"truncated-gzip", valid[:len(valid)-1], ErrInvalidArchive},
		{"trailing-gzip", append(bytes.Clone(valid), 1), ErrInvalidArchive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sealed := encryptedFixture(t, tc.archive, "tenant-a", key)
			root, dir := newTestRoot(t)
			bytesCount, inspectErr := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{})
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("inspection wrote to root: %v %v", entries, err)
			}
			restoreErr := root.RestoreEncryptedGzipTarNew(context.Background(), "restored", bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{})
			if tc.want == nil {
				if inspectErr != nil || restoreErr != nil || bytesCount != 5 {
					t.Fatalf("valid inspect=%v restore=%v size=%d", inspectErr, restoreErr, bytesCount)
				}
			} else if !errors.Is(inspectErr, tc.want) || !errors.Is(restoreErr, tc.want) || bytesCount != 0 {
				t.Fatalf("inspect=%v restore=%v size=%d want=%v", inspectErr, restoreErr, bytesCount, tc.want)
			}
		})
	}
	sealed := encryptedFixture(t, valid, "tenant-a", key)
	if _, err := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(sealed), "tenant-b", key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("wrong tenant: %v", err)
	}
	if _, err := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{FileBytes: 4}); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("limit: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := VerifyEncryptedGzipTar(ctx, bytes.NewReader(sealed), "tenant-a", key, ArchiveLimits{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestVerifyEncryptedSiteArchiveLimitsMatchRestore(t *testing.T) {
	key := bytes.Repeat([]byte{0x39}, 32)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "nested/a", Typeflag: tar.TypeReg}, data: []byte("hello")}, archiveTestEntry{header: tar.Header{Name: "nested/b", Typeflag: tar.TypeReg}, data: []byte("bye")})
	sealed := encryptedFixture(t, archive, "tenant-a", key)
	for _, tc := range []struct {
		name   string
		limits ArchiveLimits
		want   error
	}{
		{"file", ArchiveLimits{FileBytes: 4}, ErrArchiveLimit}, {"total", ArchiveLimits{TotalBytes: 7}, ErrArchiveLimit},
		{"entries", ArchiveLimits{Entries: 1}, ErrArchiveLimit}, {"expanded", ArchiveLimits{ExpandedBytes: 512}, ErrArchiveLimit},
		{"compressed", ArchiveLimits{CompressedBytes: int64(len(archive) - 1)}, backups.ErrBackupLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := newTestRoot(t)
			count, verifyErr := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(sealed), "tenant-a", key, tc.limits)
			restoreErr := root.RestoreEncryptedGzipTarNew(context.Background(), "target", bytes.NewReader(sealed), "tenant-a", key, tc.limits)
			if !errors.Is(verifyErr, tc.want) || !errors.Is(restoreErr, tc.want) || count != 0 {
				t.Fatalf("verify=%v restore=%v count=%d", verifyErr, restoreErr, count)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failure changed root: %v %v", entries, err)
			}
		})
	}
	corrupted := bytes.Clone(sealed)
	corrupted[len(corrupted)-1] ^= 1
	if _, err := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(corrupted), "tenant-a", key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("corrupt envelope: %v", err)
	}
	if _, err := VerifyEncryptedGzipTar(context.Background(), bytes.NewReader(sealed[:len(sealed)-1]), "tenant-a", key, ArchiveLimits{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("truncated envelope: %v", err)
	}
}
