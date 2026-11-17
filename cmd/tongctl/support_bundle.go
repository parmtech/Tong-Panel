package main

import (
	"encoding/json"
	"errors"
	"io"
	"runtime"

	"github.com/parmtech/tong-panel/internal/preflight"
	"github.com/parmtech/tong-panel/internal/supportbundle"
)

var (
	errSupportBundleUsage  = errors.New("usage: tongctl support-bundle")
	errSupportBundleOutput = errors.New("support-bundle: output unavailable")
)

func collectSupportSnapshot() (preflight.Snapshot, error) {
	snapshot, err := preflight.DetectSnapshot("/etc/os-release")
	// An unavailable OS-release file must not discard the executable's known
	// platform. No environment variable or user-provided path is consulted.
	if snapshot.GOOS == "" {
		snapshot.GOOS = runtime.GOOS
	}
	if snapshot.Arch == "" {
		snapshot.Arch = runtime.GOARCH
	}
	return snapshot, err
}

// runSupportBundle accepts no flags or attachment paths. A partial facts-only
// export is still a successful export; it is never a health/installation gate.
func runSupportBundle(args []string, output io.Writer, collect func() (preflight.Snapshot, error)) error {
	if len(args) != 0 {
		return errSupportBundleUsage
	}
	snapshot, collectErr := collect()
	bundle := supportbundle.Build(supportbundle.Facts{
		OSRelease: snapshot.OSRelease, GOOS: snapshot.GOOS, Arch: snapshot.Arch,
		CollectionFailed: collectErr != nil,
		MemoryKnown:      snapshot.MemoryKnown, MemAvailableKB: snapshot.MemAvailableKB,
		DiskKnown: snapshot.DiskKnown, DiskAvailableBytes: snapshot.DiskAvailableBytes,
		PortsKnown: snapshot.PortsKnown, TCP80Listening: snapshot.TCPListening[80], TCP443Listening: snapshot.TCPListening[443],
	})
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return errSupportBundleOutput
	}
	encoded = append(encoded, '\n')
	n, err := output.Write(encoded)
	if err != nil || n != len(encoded) {
		return errSupportBundleOutput
	}
	return nil
}
