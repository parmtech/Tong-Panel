//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

func TestExportSiteCommandRoundTrip(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root")
	private := filepath.Join(dir, "private")
	for _, p := range []string{rootPath, private, filepath.Join(rootPath, "site")} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(rootPath, "site", "index.html"), []byte("private site data"), 0600); err != nil {
		t.Fatal(err)
	}
	key := writeKeyFile(t, private, "key", bytes.Repeat([]byte{0x21}, 32), 0600)
	output := filepath.Join(private, "site.tpbk")
	args := []string{"export-site", "--tenant", "tenant-a", "--key-file", key, "--root", rootPath, "--site", "site", "--output", output, "--source-quiesced"}
	var stdout, stderr bytes.Buffer
	if err := run(args, strings.NewReader("unused"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatal("command emitted data")
	}
	encrypted, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("private site data")) {
		t.Fatal("artifact exposed plaintext")
	}
	if err := backups.Verify(bytes.NewReader(encrypted), "tenant-b", bytes.Repeat([]byte{0x21}, 32), backups.Options{}); !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("foreign tenant: %v", err)
	}
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	roots, err := files.NewTenantRoots(map[string]*files.Root{"tenant-a": root})
	if err != nil {
		t.Fatal(err)
	}
	if err := roots.RestoreEncryptedGzipTarNewForTenant(context.Background(), "tenant-a", "restored", bytes.NewReader(encrypted), bytes.Repeat([]byte{0x21}, 32), files.ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(rootPath, "restored", "index.html"))
	if err != nil || string(data) != "private site data" {
		t.Fatalf("restore: %q %v", data, err)
	}
	if err := run(args, nil, &stdout, &stderr); !errors.Is(err, os.ErrExist) {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestExportSiteArgumentsAndSourceOutputIsolation(t *testing.T) {
	for _, args := range [][]string{{}, {"--tenant", "a"}, {"--tenant=a", "--tenant=b", "--key-file=k", "--root=/r", "--site=s", "--output=/o", "--source-quiesced"}, {"--tenant=a", "--key-file=k", "--root=/r", "--site=s", "--output=/o", "--source-quiesced=true"}} {
		if _, err := parseExportSiteArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	site := filepath.Join(root, "site")
	if err := os.MkdirAll(site, 0700); err != nil {
		t.Fatal(err)
	}
	key := writeKeyFile(t, dir, "key", bytes.Repeat([]byte{1}, 32), 0600)
	args := []string{"--tenant", "tenant-a", "--key-file", key, "--root", root, "--site", "site", "--output", filepath.Join(site, "backup"), "--source-quiesced"}
	if err := runExportSite(args); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("source output accepted: %v", err)
	}
	if entries, err := os.ReadDir(site); err != nil || len(entries) != 0 {
		t.Fatalf("source changed: %v %v", entries, err)
	}
	private := filepath.Join(dir, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	insideKey := writeKeyFile(t, site, "inside.key", bytes.Repeat([]byte{2}, 32), 0600)
	args = []string{"--tenant", "tenant-a", "--key-file", insideKey, "--root", root, "--site", "site", "--output", filepath.Join(private, "backup"), "--source-quiesced"}
	if err := runExportSite(args); err == nil || !strings.Contains(err.Error(), "wrapping-key file must be outside") {
		t.Fatalf("source-contained key accepted: %v", err)
	}
	if entries, err := os.ReadDir(private); err != nil || len(entries) != 0 {
		t.Fatalf("unexpected output: %v %v", entries, err)
	}
}

func TestPrivateCiphertextPublicationOutcomes(t *testing.T) {
	dir := t.TempDir()
	// testing.TempDir's leaf is created with 0777 subject to umask, so make
	// the fixture satisfy the same private-directory contract as production.
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	producer := func(w io.Writer) error { _, err := w.Write([]byte("ciphertext")); return err }
	producerErr := errors.New("injected producer failure")
	if err := publishPrivateCiphertext(dir, "clean", func(w io.Writer) error { _, _ = w.Write([]byte("partial ciphertext")); return producerErr }, unix.Linkat, unix.Fsync); !errors.Is(err, producerErr) {
		t.Fatalf("producer failure = %v, want injected error", err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("failed producer left stage: %v %v", entries, err)
	}
	linkThenError := func(a int, b string, c int, d string, flags int) error {
		if err := unix.Linkat(a, b, c, d, flags); err != nil {
			return err
		}
		return unix.EIO
	}
	if err := publishPrivateCiphertext(dir, "uncertain", producer, linkThenError, unix.Fsync); !errors.Is(err, errExportPublishUnknown) {
		t.Fatalf("ambiguous link: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "uncertain"))
	if err != nil || string(data) != "ciphertext" {
		t.Fatalf("published target changed: %q %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tong-ciphertext-stage-") {
			info, err := entry.Info()
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("unsafe stage: %v %v", info, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("ambiguous link did not retain private stage")
	}
	if err := publishPrivateCiphertext(dir, "sync-failed", producer, unix.Linkat, func(int) error { return unix.EIO }); !errors.Is(err, errExportPublishedUncertain) {
		t.Fatalf("sync failure: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "sync-failed")); err != nil || string(data) != "ciphertext" {
		t.Fatalf("published output missing: %q %v", data, err)
	}
}

func TestExportSiteRejectsUnsafeSourceAndPrivateOutput(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			root := filepath.Join(dir, "root")
			site := filepath.Join(root, "site")
			private := filepath.Join(dir, "private")
			if err := os.MkdirAll(site, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(private, 0700); err != nil {
				t.Fatal(err)
			}
			if kind == "symlink" {
				if err := os.Symlink(filepath.Join(dir, "outside"), filepath.Join(site, "link")); err != nil {
					t.Fatal(err)
				}
			} else {
				file := filepath.Join(site, "one")
				if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Link(file, filepath.Join(site, "two")); err != nil {
					t.Fatal(err)
				}
			}
			key := writeKeyFile(t, private, "key", bytes.Repeat([]byte{5}, 32), 0600)
			dest := filepath.Join(private, "backup")
			args := []string{"--tenant", "tenant-a", "--key-file", key, "--root", root, "--site", "site", "--output", dest, "--source-quiesced"}
			if err := runExportSite(args); !errors.Is(err, files.ErrInvalidArchive) {
				t.Fatalf("unsafe source: %v", err)
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("unsafe source published: %v", err)
			}
			entries, err := os.ReadDir(private)
			if err != nil || len(entries) != 1 || entries[0].Name() != "key" {
				t.Fatalf("unexpected stage after rejection: %v %v", entries, err)
			}
		})
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := publishPrivateCiphertext(dir, "backup", func(io.Writer) error { return nil }, unix.Linkat, unix.Fsync); err == nil {
		t.Fatal("non-private output directory accepted")
	}
}

func TestExportSiteRejectsInvalidTenantAndRelativePathBeforeIO(t *testing.T) {
	for _, tc := range []struct{ tenant, site string }{{"bad%tenant", "site"}, {strings.Repeat("a", 129), "site"}, {"tenant-a", "../other"}, {"tenant-a", "/absolute"}, {"tenant-a", "site//nested"}, {"tenant-a", "site/.tong-archive-stage-foo"}, {"tenant-a", "site\\other"}} {
		args := []string{"--tenant", tc.tenant, "--key-file", "/missing/key", "--root", "/missing/root", "--site", tc.site, "--output", "/missing/output", "--source-quiesced"}
		if err := runExportSite(args); !errors.Is(err, errExportArguments) {
			t.Fatalf("tenant=%q site=%q: %v", tc.tenant, tc.site, err)
		}
	}
}
