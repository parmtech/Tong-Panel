//go:build !linux

package preflight

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNonLinuxDiskFactsRemainUnknownAndMutationGateBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(path, []byte(ubuntu), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := DetectSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.DiskKnown || snapshot.DiskAvailableBytes != 0 {
		t.Fatalf("non-Linux disk facts = (%d, known=%t), want unknown", snapshot.DiskAvailableBytes, snapshot.DiskKnown)
	}

	report, err := Evaluate(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Candidate {
		t.Fatal("non-Linux runtime reported as an installation candidate")
	}

	gate := EvaluateMutationGate(snapshot, MutationRequirements{
		MinimumDiskBytes: 8 * 1024 * 1024 * 1024,
		RequiredTCPPorts: []uint16{80, 443},
	})
	if gate.Allowed {
		t.Fatalf("unknown non-Linux host facts allowed a future mutation: %#v", gate)
	}
	unsupportedOS, diskUnknown := false, false
	for _, blocker := range gate.Blockers {
		unsupportedOS = unsupportedOS || blocker == "unsupported_os"
		diskUnknown = diskUnknown || blocker == "disk_unknown"
	}
	if !unsupportedOS || !diskUnknown {
		t.Fatalf("gate blockers = %#v, want unsupported_os and disk_unknown", gate.Blockers)
	}
}
