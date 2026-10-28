//go:build linux

package api

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxFileMutationBarrierSubprocessHelper(t *testing.T) {
	if os.Getenv("TONG_FILE_BARRIER_HELPER") != "1" {
		return
	}
	if err := os.WriteFile(os.Getenv("TONG_FILE_BARRIER_STARTED"), []byte("started"), 0600); err != nil {
		t.Fatal(err)
	}
	barrier, err := NewLinuxFileMutationBarrier(os.Getenv("TONG_FILE_BARRIER_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	defer barrier.Close()
	release, err := barrier.Acquire(context.Background(), "tenant-a", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("TONG_FILE_BARRIER_SIGNAL"), []byte("acquired"), 0600); err != nil {
		t.Fatal(err)
	}
	release()
}

func TestLinuxFileMutationBarrierCoordinatesAcrossProcesses(t *testing.T) {
	barrier, directory := newTestLinuxFileMutationBarrier(t)
	maintenanceRelease, err := barrier.Acquire(context.Background(), "tenant-a", true)
	if err != nil {
		t.Fatal(err)
	}
	defer maintenanceRelease()
	command := exec.Command(os.Args[0], "-test.v", "-test.run=^TestLinuxFileMutationBarrierSubprocessHelper$")
	command.Env = append(os.Environ(), "TONG_FILE_BARRIER_HELPER=1", "TONG_FILE_BARRIER_DIR="+directory)
	signal := filepath.Join(directory, "acquired")
	started := filepath.Join(directory, "started")
	command.Env = append(command.Env, "TONG_FILE_BARRIER_SIGNAL="+signal, "TONG_FILE_BARRIER_STARTED="+started)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if !waitForFile(started, time.Second) {
		_ = command.Process.Kill()
		t.Fatal("mutation helper did not start")
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(signal); err == nil {
		_ = command.Process.Kill()
		t.Fatal("other process entered mutation during exclusive maintenance")
	}
	maintenanceRelease()
	if !waitForFile(signal, time.Second) {
		_ = command.Process.Kill()
		t.Fatal("mutation process did not acquire after maintenance released")
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("mutation helper failed: %v", err)
	}
}

func waitForFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestLinuxFileMutationBarrierDrainsMutationsBeforeMaintenance(t *testing.T) {
	apiBarrier, barrierDir := newTestLinuxFileMutationBarrier(t)
	maintenanceBarrier, err := NewLinuxFileMutationBarrier(barrierDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = maintenanceBarrier.Close() })
	mutationRelease, err := apiBarrier.Acquire(context.Background(), "tenant-a", false)
	if err != nil {
		t.Fatal(err)
	}
	defer mutationRelease()
	maintenanceCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	maintenanceResult := make(chan error, 1)
	go func() {
		maintenanceResult <- WithExclusiveFileMaintenance(maintenanceCtx, maintenanceBarrier, "tenant-a", func() error {
			return nil
		})
	}()
	select {
	case err := <-maintenanceResult:
		t.Fatalf("exclusive maintenance passed an active mutation: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	mutationRelease()
	select {
	case err := <-maintenanceResult:
		if err != nil {
			t.Fatalf("maintenance after mutation drain failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("maintenance did not acquire after mutation drained")
	}
}

func TestLinuxFileMutationBarrierIsTenantScopedAndCancelable(t *testing.T) {
	barrier, _ := newTestLinuxFileMutationBarrier(t)
	releaseA, err := barrier.Acquire(context.Background(), "tenant-a", true)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseA()
	releaseB, err := barrier.Acquire(context.Background(), "tenant-b", false)
	if err != nil {
		t.Fatalf("tenant A blocked tenant B: %v", err)
	}
	releaseB()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := barrier.Acquire(ctx, "tenant-a", false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting mutation cancellation error = %v", err)
	}
}

func TestLinuxFileMutationBarrierCloseKeepsActiveLeaseUntilRelease(t *testing.T) {
	barrier, directory := newTestLinuxFileMutationBarrier(t)
	release, err := barrier.Acquire(context.Background(), "tenant-a", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := barrier.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := barrier.Acquire(context.Background(), "tenant-a", false); err == nil {
		t.Fatal("closed barrier accepted a new mutation")
	}
	peer, err := NewLinuxFileMutationBarrier(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := peer.Acquire(ctx, "tenant-a", true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("active lease did not survive barrier close, error=%v", err)
	}
	release()
	maintenanceRelease, err := peer.Acquire(context.Background(), "tenant-a", true)
	if err != nil {
		t.Fatalf("maintenance after lease release failed: %v", err)
	}
	maintenanceRelease()
}

func newTestLinuxFileMutationBarrier(t *testing.T) (*LinuxFileMutationBarrier, string) {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	barrier, err := NewLinuxFileMutationBarrier(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = barrier.Close() })
	return barrier, directory
}

func TestNewLinuxFileMutationBarrierRejectsUnsafeDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "locks")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLinuxFileMutationBarrier(directory); err == nil {
		t.Fatal("accepted a group/other-writable lock directory")
	}
}
