package preflight

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectSnapshotRetainsPartialFactsWhenOSReleaseUnavailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-os-release")
	snapshot, err := DetectSnapshot(path)
	if !os.IsNotExist(err) {
		t.Fatal("OS-release read failure must be preserved")
	}
	if snapshot.OSRelease != "" || snapshot.GOOS != runtime.GOOS || snapshot.Arch != runtime.GOARCH {
		t.Fatal("OS-release failure discarded known runtime facts")
	}
	if _, err := Detect(path); !os.IsNotExist(err) {
		t.Fatal("doctor must retain its existing OS-release error behaviour")
	}
}
