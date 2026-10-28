package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
)

func TestRunEncryptDecryptAndRotateFromExternalKeyFiles(t *testing.T) {
	requireLinux(t)
	dir := t.TempDir()
	keyA := writeKeyFile(t, dir, "key-a", bytes.Repeat([]byte{0x31}, 32), 0o600)
	keyB := writeKeyFile(t, dir, "key-b", bytes.Repeat([]byte{0x93}, 32), 0o400)
	plain := bytes.Repeat([]byte("mariadb tenant dump fixture\n"), 3000)
	backupPath := filepath.Join(dir, "tenant-a.tpbk")
	if err := run([]string{"encrypt", "--tenant", "tenant-a", "--key-file", keyA, "--output", backupPath}, bytes.NewReader(plain), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("encrypt command error = %v", err)
	}
	sealed, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, plain) || bytes.Contains(sealed, []byte("tenant-a")) {
		t.Fatal("backup artifact exposed input or tenant identifier")
	}
	rotatedPath := filepath.Join(dir, "tenant-a-rotated.tpbk")
	if err := run([]string{"rotate", "--tenant", "tenant-a", "--old-key-file", keyA, "--key-file", keyB, "--output", rotatedPath}, bytes.NewReader(sealed), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("rotate command error = %v", err)
	}
	rotated, err := os.ReadFile(rotatedPath)
	if err != nil {
		t.Fatal(err)
	}
	restoredPath := filepath.Join(dir, "restored.sql")
	if err := run([]string{"decrypt", "--tenant", "tenant-a", "--key-file", keyB, "--output", restoredPath}, bytes.NewReader(rotated), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("decrypt command error = %v", err)
	}
	restored, err := os.ReadFile(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, plain) {
		t.Fatal("decrypt command changed backup contents")
	}
}

func TestRunRejectsInsecureOrMalformedKeyFilesWithoutPublishing(t *testing.T) {
	requireLinux(t)
	dir := t.TempDir()
	unsafeKey := writeKeyFile(t, dir, "world-readable.key", bytes.Repeat([]byte{0x41}, 32), 0o644)
	destination := filepath.Join(dir, "must-not-exist.tpbk")
	var stderr bytes.Buffer
	err := run([]string{"encrypt", "--tenant", "tenant-a", "--key-file", unsafeKey, "--output", destination}, strings.NewReader("secret"), &bytes.Buffer{}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "permissions") {
		t.Fatalf("run() error = %v, want insecure key permission denial", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("insecure key created output (Lstat error = %v)", err)
	}
	if strings.Contains(stderr.String(), string(bytes.Repeat([]byte{0x41}, 32))) {
		t.Fatal("stderr exposed key material")
	}
	shortKey := writeKeyFile(t, dir, "short.key", []byte("too short"), 0o600)
	err = run([]string{"encrypt", "--tenant", "tenant-a", "--key-file", shortKey, "--output", destination}, strings.NewReader("secret"), &bytes.Buffer{}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "exactly 32 bytes") {
		t.Fatalf("run() error = %v, want key length denial", err)
	}
}

func TestRunDoesNotOverwriteAndDoesNotPublishTamperedRestore(t *testing.T) {
	requireLinux(t)
	dir := t.TempDir()
	key := writeKeyFile(t, dir, "key", bytes.Repeat([]byte{0x55}, 32), 0o600)
	path := filepath.Join(dir, "existing.backup")
	if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"encrypt", "--tenant", "tenant-a", "--key-file", key, "--output", path}, strings.NewReader("replacement"), &bytes.Buffer{}, &bytes.Buffer{})
	if !os.IsExist(err) {
		t.Fatalf("encrypt overwrite error = %v, want existing-target denial", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "preserve" {
		t.Fatalf("existing target changed; content=%q err=%v", contents, err)
	}

	bad := filepath.Join(dir, "bad.tpbk")
	if err := run([]string{"encrypt", "--tenant", "tenant-a", "--key-file", key, "--output", bad}, strings.NewReader("contents"), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	sealed, err := os.ReadFile(bad)
	if err != nil {
		t.Fatal(err)
	}
	sealed[len(sealed)-1] ^= 1
	restored := filepath.Join(dir, "must-not-appear.sql")
	err = run([]string{"decrypt", "--tenant", "tenant-a", "--key-file", key, "--output", restored}, bytes.NewReader(sealed), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, backups.ErrInvalidBackup) {
		t.Fatalf("tampered restore error = %v, want ErrInvalidBackup", err)
	}
	if _, err := os.Lstat(restored); !os.IsNotExist(err) {
		t.Fatalf("tampered restore published plaintext (Lstat error = %v)", err)
	}
}

func requireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("tongbackup key-file permission checks are Linux-only")
	}
}

func writeKeyFile(t *testing.T, dir, name string, key []byte, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, key, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}
