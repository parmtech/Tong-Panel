//go:build linux

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"github.com/parmtech/tong-panel/internal/backups"
	"github.com/parmtech/tong-panel/internal/files"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifySiteRejectsAuthenticatedUnsafeArchiveWithoutOutput(t *testing.T) {
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "../escape", Typeflag: tar.TypeReg, Size: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	keyBytes := bytes.Repeat([]byte{0x51}, 32)
	var sealed bytes.Buffer
	if err := backups.Encrypt(&sealed, bytes.NewReader(raw.Bytes()), "tenant-a", keyBytes, backups.Options{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	key := writeKeyFile(t, dir, "key", keyBytes, 0600)
	var out, stderr bytes.Buffer
	base := []string{"--tenant", "tenant-a", "--key-file", key}
	if err := run(append([]string{"verify"}, base...), bytes.NewReader(sealed.Bytes()), &out, &stderr); err != nil {
		t.Fatalf("envelope verify: %v", err)
	}
	if err := run(append([]string{"verify-site"}, base...), bytes.NewReader(sealed.Bytes()), &out, &stderr); !errors.Is(err, files.ErrInvalidArchive) {
		t.Fatalf("unsafe site verify: %v", err)
	}
	if out.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("verification exposed data")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(key) {
		t.Fatalf("inspection created files: %v %v", entries, err)
	}
}

func TestVerifySiteAcceptsValidSiteArchiveAndRejectsWrongTenant(t *testing.T) {
	dir := t.TempDir()
	keyBytes := bytes.Repeat([]byte{0x52}, 32)
	key := writeKeyFile(t, dir, "key", keyBytes, 0600)
	sealed := labEncryptedArchive(t, keyBytes)
	var out, stderr bytes.Buffer
	if err := run([]string{"verify-site", "--tenant", "tenant-a", "--key-file", key}, bytes.NewReader(sealed), &out, &stderr); err != nil {
		t.Fatalf("valid site verify: %v", err)
	}
	if err := run([]string{"verify-site", "--tenant", "tenant-b", "--key-file", key}, bytes.NewReader(sealed), &out, &stderr); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("wrong tenant: %v", err)
	}
	if out.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("verification emitted data")
	}
}
