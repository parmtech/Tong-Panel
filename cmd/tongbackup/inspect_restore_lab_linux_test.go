//go:build linux

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type shortInventoryWriter struct{}

func (shortInventoryWriter) Write([]byte) (int, error) { return 0, nil }

func TestInspectRestoreLabNonRootOptInAndNoPayloadRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("lab command refuses root")
	}
	t.Setenv("TONG_DISPOSABLE_SITE_RESTORE", "1")
	parent := t.TempDir()
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "tong-restore-lab-case")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, ".tong-archive-stage-"+strings.Repeat("a", 32))
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(stage, "private-marker")
	if err := os.WriteFile(marker, []byte("private-content"), 0000); err != nil {
		t.Fatal(err)
	}
	args := []string{"--root", root, "--disposable-only"}
	var out bytes.Buffer
	if err := runInspectRestoreLab(args, &out); err != nil {
		t.Fatal(err)
	}
	expected := "Archive stages observed: 1\nSnapshot includes active stages; count does not establish publication or permit deletion or quota release.\n"
	if out.String() != expected || strings.Contains(out.String(), "private-marker") {
		t.Fatalf("unexpected output %q", out.String())
	}
	if _, err := os.Lstat(marker); err != nil {
		t.Fatalf("inventory changed stage: %v", err)
	}
	if err := runInspectRestoreLab(args, shortInventoryWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short writer: %v", err)
	}
	t.Setenv("TONG_DISPOSABLE_SITE_RESTORE", "")
	out.Reset()
	if err := runInspectRestoreLab(args, &out); !errors.Is(err, errLabBoundary) || out.Len() != 0 {
		t.Fatalf("missing opt-in: %v", err)
	}
}
func TestInspectRestoreLabRejectsUnsafeRootsAndStages(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("lab command refuses root")
	}
	t.Setenv("TONG_DISPOSABLE_SITE_RESTORE", "1")
	parent := t.TempDir()
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "tong-restore-lab-case")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	args := []string{"--root", root, "--disposable-only"}
	invalid := filepath.Join(root, ".tong-archive-stage-malformed")
	if err := os.Mkdir(invalid, 0700); err != nil {
		t.Fatal(err)
	}
	if err := runInspectRestoreLab(args, &out); !errors.Is(err, errLabInventory) || out.Len() != 0 {
		t.Fatalf("unsafe stage: %v %q", err, out.String())
	}
	if _, err := os.Lstat(invalid); err != nil {
		t.Fatalf("unsafe stage removed: %v", err)
	}
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := runInspectRestoreLab(args, &out); !errors.Is(err, errLabBoundary) {
		t.Fatalf("permissive root accepted: %v", err)
	}
	for _, bad := range [][]string{{}, {"--root", root}, {"--root", root, "--root", root, "--disposable-only"}, {"--root", root, "--disposable-only=true"}} {
		if _, err := parseInspectRestoreLabArgs(bad); !errors.Is(err, errLabArguments) {
			t.Fatalf("bad args %v: %v", bad, err)
		}
	}
}
