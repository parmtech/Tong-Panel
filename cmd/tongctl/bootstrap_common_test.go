package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
