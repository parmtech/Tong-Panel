//go:build linux

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/backups"
	"github.com/parmtech/tong-panel/internal/files"
)

func labEncryptedArchive(t *testing.T, key []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	body := []byte("private site marker")
	if err := tw.WriteHeader(&tar.Header{Name: "nested/index.html", Mode: 0600, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := backups.Encrypt(&out, bytes.NewReader(archive.Bytes()), "tenant-a", key, backups.Options{}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func labArgs(parent, key, name, tenant string) []string {
	return []string{"--tenant", tenant, "--key-file", key, "--root", filepath.Join(parent, "tong-restore-lab-"+name), "--destination", "site", "--disposable-only"}
}

type restoreLabObservedReader struct{ reads int }

func (r *restoreLabObservedReader) Read([]byte) (int, error) {
	r.reads++
	return 0, errors.New("must not read")
}
func TestRestoreSiteLabRequiresStrictArgsAndExplicitOptIn(t *testing.T) {
	for _, args := range [][]string{{}, {"--tenant", "a"}, {"--tenant=a", "--tenant=b", "--key-file=/k", "--root=/r", "--destination=site", "--disposable-only"}, {"--tenant=a", "--key-file=/k", "--root=/r", "--destination=../x", "--disposable-only"}, {"--tenant=a", "--key-file=/k", "--root=/r", "--destination=site", "--disposable-only=true"}} {
		if _, err := parseRestoreSiteLabArgs(args); !errors.Is(err, errLabArguments) {
			t.Fatalf("accepted %v: %v", args, err)
		}
	}
	t.Setenv("TONG_DISPOSABLE_SITE_RESTORE", "")
	input := &restoreLabObservedReader{}
	if err := runRestoreSiteLab(labArgs("/tmp", "/tmp/key", "absent", "tenant-a"), input); !errors.Is(err, errLabBoundary) || input.reads != 0 {
		t.Fatalf("missing opt-in: %v", err)
	}
}
func TestRestoreSiteLabFreshRootRoundTripAndDenials(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("lab command refuses root")
	}
	t.Setenv("TONG_DISPOSABLE_SITE_RESTORE", "1")
	parent := t.TempDir()
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	keyBytes := bytes.Repeat([]byte{0x73}, 32)
	key := writeKeyFile(t, parent, "key", keyBytes, 0600)
	sealed := labEncryptedArchive(t, keyBytes)
	args := labArgs(parent, key, "success", "tenant-a")
	if err := runRestoreSiteLab(args, bytes.NewReader(sealed)); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "tong-restore-lab-success")
	body, err := os.ReadFile(filepath.Join(root, "site", "nested", "index.html"))
	if err != nil || string(body) != "private site marker" {
		t.Fatalf("restored: %q %v", body, err)
	}
	if err := runRestoreSiteLab(args, bytes.NewReader(sealed)); !errors.Is(err, errLabBoundary) {
		t.Fatalf("existing root accepted: %v", err)
	}
	body, err = os.ReadFile(filepath.Join(root, "site", "nested", "index.html"))
	if err != nil || string(body) != "private site marker" {
		t.Fatal("existing lab modified")
	}
	for _, tc := range []struct {
		name, tenant string
		data         []byte
	}{{"wrong-tenant", "tenant-b", sealed}, {"truncated", "tenant-a", sealed[:len(sealed)-1]}} {
		args := labArgs(parent, key, tc.name, tc.tenant)
		if err := runRestoreSiteLab(args, bytes.NewReader(tc.data)); !errors.Is(err, backups.ErrInvalidBackup) {
			t.Fatalf("%s: %v", tc.name, err)
		}
		root := filepath.Join(parent, "tong-restore-lab-"+tc.name)
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("%s changed root: %v %v", tc.name, entries, err)
		}
	}
}
func TestRestoreSiteLabOutcomePreservesUncertainty(t *testing.T) {
	failure := &os.PathError{Op: "open", Path: "private-filename-and-secret-marker", Err: os.ErrPermission}
	dir := t.TempDir()
	evidence := filepath.Join(dir, "private-stage-evidence")
	if err := os.WriteFile(evidence, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreSiteLabOutcome(files.ArchivePublishResult{State: files.ArchivePublicationUnknown, LogicalBytes: 10}, failure); !errors.Is(err, errLabOutcomeUnknown) || !errors.Is(err, failure) {
		t.Fatalf("unknown: %v", err)
	}
	if err := restoreSiteLabOutcome(files.ArchivePublishResult{State: files.ArchivePublished, LogicalBytes: 10}, failure); !errors.Is(err, errLabPublishedUncertain) || !errors.Is(err, failure) {
		t.Fatalf("published uncertain: %v", err)
	}
	if err := restoreSiteLabOutcome(files.ArchivePublishResult{State: files.ArchiveNotPublished}, failure); !errors.Is(err, failure) {
		t.Fatalf("not published: %v", err)
	}
	if err := restoreSiteLabOutcome(files.ArchivePublishResult{State: files.ArchivePublished}, nil); err != nil {
		t.Fatalf("published success: %v", err)
	}
	for _, state := range []files.ArchivePublicationState{files.ArchiveNotPublished, files.ArchivePublished, files.ArchivePublicationUnknown} {
		err := restoreSiteLabOutcome(files.ArchivePublishResult{State: state}, failure)
		var pathError *os.PathError
		if !errors.Is(err, os.ErrPermission) || !errors.As(err, &pathError) || strings.Contains(err.Error(), "private-filename-and-secret-marker") {
			t.Fatalf("unredacted error for state %v: %v", state, err)
		}
	}
	if data, err := os.ReadFile(evidence); err != nil || string(data) != "retain" {
		t.Fatalf("evidence changed: %q %v", data, err)
	}
}
