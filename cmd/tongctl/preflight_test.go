package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/preflight"
)

func preflightSnapshot() preflight.Snapshot {
	return preflight.Snapshot{
		OSRelease: "ID=ubuntu\nVERSION_ID=\"24.04\"\n", GOOS: "linux", Arch: "amd64",
		DiskAvailableBytes: 12 * 1024 * 1024 * 1024, DiskKnown: true,
		TCPListening: map[uint16]bool{}, PortsKnown: true,
	}
}

func TestInstallPreflightReportsReadOnlyThresholdMatch(t *testing.T) {
	var output bytes.Buffer
	allowed, err := runInstallPreflight([]string{"--min-disk-bytes", "8589934592", "--required-tcp-ports", "80,443"}, preflightSnapshot(), &output)
	if err != nil || !allowed {
		t.Fatalf("preflight allowed=%t error=%v output=%s", allowed, err, output.String())
	}
	var result installPreflightResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Gate.Allowed || len(result.Gate.Blockers) != 0 || !strings.Contains(result.Notice, "not installation authorization") {
		t.Fatalf("preflight result %#v", result)
	}
}

func TestInstallPreflightFailsClosedOnUnsupportedOrUnknownFacts(t *testing.T) {
	tests := []struct {
		name     string
		snapshot preflight.Snapshot
		want     string
	}{
		{name: "unsupported OS", snapshot: func() preflight.Snapshot {
			s := preflightSnapshot()
			s.OSRelease = "ID=debian\nVERSION_ID=13\n"
			return s
		}(), want: "unsupported_os"},
		{name: "low disk", snapshot: func() preflight.Snapshot { s := preflightSnapshot(); s.DiskAvailableBytes = 1; return s }(), want: "disk_below_minimum"},
		{name: "unknown disk", snapshot: func() preflight.Snapshot { s := preflightSnapshot(); s.DiskKnown = false; return s }(), want: "disk_unknown"},
		{name: "unknown listeners", snapshot: func() preflight.Snapshot { s := preflightSnapshot(); s.PortsKnown = false; return s }(), want: "ports_unknown"},
		{name: "occupied port", snapshot: func() preflight.Snapshot { s := preflightSnapshot(); s.TCPListening[443] = true; return s }(), want: "tcp_port_in_use:443"},
		{name: "malformed OS", snapshot: func() preflight.Snapshot { s := preflightSnapshot(); s.OSRelease = "invalid"; return s }(), want: "invalid_os_facts"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			allowed, err := runInstallPreflight([]string{"--min-disk-bytes", "8", "--required-tcp-ports", "80,443"}, test.snapshot, &output)
			if err != nil || allowed {
				t.Fatalf("preflight allowed=%t error=%v output=%s", allowed, err, output.String())
			}
			var result installPreflightResult
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			for _, blocker := range result.Gate.Blockers {
				if blocker == test.want {
					return
				}
			}
			t.Fatalf("blockers %#v do not include %q", result.Gate.Blockers, test.want)
		})
	}
}

func TestInstallPreflightRejectsIncompleteAndAmbiguousPolicy(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"--required-tcp-ports", "80"},
		{"--min-disk-bytes", "1"},
		{"--min-disk-bytes", "1", "--required-tcp-ports", "0"},
		{"--min-disk-bytes", "1", "--required-tcp-ports", "80,80"},
		{"--min-disk-bytes", "1", "--required-tcp-ports", "bad"},
		{"--min-disk-bytes", "1", "--required-tcp-ports", "80", "extra"},
	} {
		var output bytes.Buffer
		if allowed, err := runInstallPreflight(args, preflightSnapshot(), &output); err == nil || allowed || output.Len() != 0 {
			t.Fatalf("ambiguous policy args %q: allowed=%t error=%v output=%q", args, allowed, err, output.String())
		}
	}
}
