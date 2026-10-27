package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
)

func passwordTestPipe(t *testing.T, secret string) *os.File {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(secret)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}

func TestBootstrapOwnerCLIRequiresExplicitLocalOptInAndPrivatePipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	args := []string{"--db", path, "--username", "first_owner", "--tenant", "Initial Tenant", "--password-stdin"}
	t.Setenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", "")
	if err := localOwnerBootstrap(args, passwordTestPipe(t, "unique first owner passphrase\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("bootstrap accepted without local operator opt-in")
	}
	t.Setenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", "yes")
	file, err := os.CreateTemp(t.TempDir(), "plaintext-secret-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := localOwnerBootstrap(args, file, &bytes.Buffer{}); err == nil {
		t.Fatal("bootstrap accepted password in a regular file")
	}
	for _, bad := range [][]string{
		{"--db", path, "--username", "first_owner", "--tenant", "Initial Tenant"},
		append(append([]string{}, args...), "extra"),
		{"--password", "password-exposed-in-process-list"},
	} {
		if err := localOwnerBootstrap(bad, passwordTestPipe(t, "unique first owner passphrase\n"), &bytes.Buffer{}); err == nil {
			t.Fatalf("bad CLI flags accepted: %v", bad)
		}
	}
}

func TestBootstrapOwnerCLIIdempotentAuditAndNoPasswordOutput(t *testing.T) {
	t.Setenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", "yes")
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "panel.db")
	args := []string{"--db", path, "--username", "first_owner", "--tenant", "Initial Tenant", "--password-stdin"}
	secret := "extremely long private first owner secret"
	var output bytes.Buffer
	if err := localOwnerBootstrap(args, passwordTestPipe(t, secret+"\n"), &output); err != nil {
		t.Fatalf("local owner bootstrap failed: %v", err)
	}
	if strings.Contains(output.String(), secret) || !strings.Contains(output.String(), "local owner created;") {
		t.Fatalf("bootstrap leaked password or returned no nonsecret receipt: %s", output.String())
	}
	for _, path := range []string{filepath.Dir(path), path} {
		stat, err := os.Stat(path)
		if err != nil || stat.Mode().Perm()&0077 != 0 {
			t.Fatalf("bootstrap persisted public files: %s mode=%v err=%v", path, stat.Mode(), err)
		}
	}
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var phc string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE username='first_owner'").Scan(&phc); err != nil {
		t.Fatal(err)
	}
	if !auth.VerifyPassword([]byte(secret), phc) {
		t.Fatal("CLI owner hash did not verify")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='owner.bootstrap' AND actor_kind='user'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("owner audit missing: count=%d err=%v", count, err)
	}
	if err := localOwnerBootstrap(args, passwordTestPipe(t, secret+"\n"), &output); err == nil {
		t.Fatal("existing database allowed second bootstrap")
	}
}

func TestBootstrapOwnerCLIRejectsWorldReadableFileAndSymlink(t *testing.T) {
	t.Setenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", "yes")
	dir := t.TempDir()
	path := filepath.Join(dir, "panel.db")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"--db", path, "--username", "owner", "--tenant", "Org", "--password-stdin"}
	if err := localOwnerBootstrap(args, passwordTestPipe(t, "long bootstrap passphrase\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("world-readable database file accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real.db")
	if err := os.WriteFile(target, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := localOwnerBootstrap(args, passwordTestPipe(t, "long bootstrap passphrase\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("symlink database target accepted")
	}
}
