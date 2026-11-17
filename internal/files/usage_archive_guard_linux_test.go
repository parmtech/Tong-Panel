//go:build linux

package files

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestMeasureUsageRejectsEveryReservedArchiveStage(t *testing.T) {
	for _, kind := range []string{"private", "empty", "locked", "permissive", "malformed", "symlink", "regular"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := newTestRoot(t)
			if err := os.WriteFile(filepath.Join(dir, "visible"), []byte("abc"), 0600); err != nil {
				t.Fatal(err)
			}
			name := archiveStagePrefix + strings.Repeat("a", 32)
			if kind == "malformed" {
				name = archiveStagePrefix + "not-a-stage-id"
			}
			stage := filepath.Join(dir, name)
			switch kind {
			case "symlink":
				if err := os.Symlink(t.TempDir(), stage); err != nil {
					t.Fatal(err)
				}
			case "regular":
				if err := os.WriteFile(stage, []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.Mkdir(stage, 0700); err != nil {
					t.Fatal(err)
				}
				if kind != "empty" {
					if err := os.WriteFile(filepath.Join(stage, "payload"), []byte("unresolved"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "permissive" {
					if err := os.Chmod(stage, 0755); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "locked" {
					fd, err := unix.Open(stage, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
					if err != nil {
						t.Fatal(err)
					}
					defer unix.Close(fd)
					if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
						t.Fatal(err)
					}
				}
			}
			usage, err := root.MeasureUsage(100, 8)
			if !errors.Is(err, ErrUsageSnapshotDirty) || usage != 0 {
				t.Fatalf("unresolved %s stage produced usage=%d err=%v", kind, usage, err)
			}
		})
	}
}

func TestMeasureUsageRejectsRetainedUnknownPublication(t *testing.T) {
	root, dir := newTestRoot(t)
	archive := gzipTarForTest(t, archiveTestEntry{header: tar.Header{Name: "asset", Typeflag: tar.TypeReg}, data: []byte("preserve")})
	var result ArchivePublishResult
	err := root.extractGzipTarNewWithResult(context.Background(), "site", archive, ArchiveLimits{}, false, unix.Fsync,
		func(int, string, int, string, uint) error { return unix.EIO }, &result)
	if !errors.Is(err, ErrPublicationOutcomeUnknown) || result.State != ArchivePublicationUnknown {
		t.Fatalf("expected unresolved publication: %+v err=%v", result, err)
	}
	usage, err := root.MeasureUsage(100, 8)
	if !errors.Is(err, ErrUsageSnapshotDirty) || usage != 0 {
		t.Fatalf("retained publication produced trusted usage=%d err=%v", usage, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("retained stage changed: %v err=%v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name(), "asset"))
	if err != nil || string(data) != "preserve" {
		t.Fatalf("usage scan altered recovery evidence: %q err=%v", data, err)
	}
}
