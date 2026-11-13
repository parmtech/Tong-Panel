//go:build linux

package files

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRetainedArchiveInventoryReadsMetadataOnly(t *testing.T) {
	root, dir := newTestRoot(t)
	stage := archiveStagePrefix + strings.Repeat("a", 32)
	if err := os.Mkdir(filepath.Join(dir, stage), 0700); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, stage, "private-marker")
	if err := os.WriteFile(secret, []byte("do-not-read"), 0000); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ordinary"), []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	stageFD, err := unix.Open(filepath.Join(dir, stage), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(stageFD)
	if err := unix.Flock(stageFD, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer unix.Flock(stageFD, unix.LOCK_UN)
	count, err := root.CountRetainedArchiveStages(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if info, err := os.Stat(secret); err != nil || info.Mode().Perm() != 0000 {
		t.Fatalf("stage data changed: %v %v", info, err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "ordinary")); err != nil || string(data) != "untouched" {
		t.Fatalf("ordinary data changed: %q %v", data, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if count, err := root.CountRetainedArchiveStages(ctx); !errors.Is(err, context.Canceled) || count != 0 {
		t.Fatalf("cancel count=%d err=%v", count, err)
	}
}
func TestRetainedArchiveInventoryRejectsUnsafeEntries(t *testing.T) {
	for _, kind := range []string{"invalid-name", "symlink", "file", "permissive"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := newTestRoot(t)
			name := archiveStagePrefix + strings.Repeat("b", 32)
			target := filepath.Join(dir, name)
			switch kind {
			case "invalid-name":
				target = filepath.Join(dir, archiveStagePrefix+"malformed")
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(t.TempDir(), target); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(target, []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
			case "permissive":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(target, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if count, err := root.CountRetainedArchiveStages(context.Background()); !errors.Is(err, ErrUsageSnapshotDirty) || count != 0 {
				t.Fatalf("unsafe %s count=%d err=%v", kind, count, err)
			}
			if _, err := os.Lstat(target); err != nil {
				t.Fatalf("inventory removed stage: %v", err)
			}
		})
	}
	st := unix.Stat_t{Mode: unix.S_IFDIR | 0700, Uid: uint32(unix.Geteuid()) + 1}
	if privateArchiveStage(st) {
		t.Fatal("unowned stage accepted")
	}
}
func TestRetainedArchiveInventoryBoundAndTenantScope(t *testing.T) {
	a, dir := newTestRoot(t)
	b, _ := newTestRoot(t)
	roots, err := NewTenantRoots(map[string]*Root{"a": a, "b": b})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, archiveStagePrefix+strings.Repeat("c", 32)), 0700); err != nil {
		t.Fatal(err)
	}
	if count, err := roots.CountRetainedArchiveStagesForTenant(context.Background(), "a"); err != nil || count != 1 {
		t.Fatalf("tenant a %d %v", count, err)
	}
	if count, err := roots.CountRetainedArchiveStagesForTenant(context.Background(), "b"); err != nil || count != 0 {
		t.Fatalf("tenant b %d %v", count, err)
	}
	if _, err := roots.CountRetainedArchiveStagesForTenant(context.Background(), "missing"); !errors.Is(err, ErrTenantDenied) {
		t.Fatalf("missing tenant %v", err)
	}
	for i := 1; i <= MaxArchiveStageInventoryEntries; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("x%05d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if count, err := a.CountRetainedArchiveStages(context.Background()); !errors.Is(err, ErrUsageScanLimit) || count != 0 {
		t.Fatalf("over limit count=%d err=%v", count, err)
	}
}
