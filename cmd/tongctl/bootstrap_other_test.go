//go:build !linux

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapOwnerUnsupportedPlatformDoesNotCreateFiles(t *testing.T) {
	t.Setenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", "yes")
	directory := t.TempDir()
	parent := filepath.Join(directory, "not-created", "nested")
	databasePath := filepath.Join(parent, "panel.db")
	args := []string{"--db", databasePath, "--username", "first_owner", "--tenant", "Initial Tenant", "--password-stdin"}
	err := localOwnerBootstrap(args, passwordTestPipe(t, "unique first owner passphrase\n"), &bytes.Buffer{})
	if !errors.Is(err, errBootstrapUnsupportedPlatform) {
		t.Fatalf("unsupported platform error = %v, want %v", err, errBootstrapUnsupportedPlatform)
	}
	if _, err := os.Lstat(parent); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unsupported bootstrap created parent path: err=%v", err)
	}
	if _, err := os.Lstat(databasePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unsupported bootstrap created database: err=%v", err)
	}
}
