package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ubuntu = "NAME=Ubuntu\nID=ubuntu\nVERSION_ID=\"24.04\"\n"

func TestParseOSRelease(t *testing.T) {
	id, v, err := ParseOSRelease(ubuntu)
	if err != nil || id != "ubuntu" || v != "24.04" {
		t.Fatalf("got %s %s %v", id, v, err)
	}
	if _, _, err = ParseOSRelease("invalid"); err == nil {
		t.Fatal("missing os fields accepted")
	}
}

func TestParseAvailableMemory(t *testing.T) {
	kb, err := ParseMemAvailableKB("MemTotal: 4194304 kB\nMemFree: 1024 kB\nMemAvailable: 2048000 kB\n")
	if err != nil || kb != 2048000 {
		t.Fatalf("value %d: %v", kb, err)
	}
	for _, raw := range []string{"MemFree: 100 kB", "MemAvailable: 0 kB", "MemAvailable: -5 kB", "MemAvailable: 100 MB"} {
		if _, err := ParseMemAvailableKB(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestReadOnlyListeningPortParsing(t *testing.T) {
	lines := "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n" +
		"   0: 00000000:0050 00000000:0000 0A 0:0 00:0 0 0 0 0\n" +
		"   1: 00000000:01BB 00000000:0000 01 0:0 00:0 0 0 0 0\n"
	ports, err := ParseListeningTCP(lines)
	if err != nil || !ports[80] || ports[443] {
		t.Fatalf("ports=%v error=%v", ports, err)
	}
	if _, err := ParseListeningTCP("bad-data"); err == nil {
		t.Fatal("accepted invalid proc header")
	}
}

func TestEvaluationMatrixAndUnknownReadings(t *testing.T) {
	base := Snapshot{OSRelease: ubuntu, GOOS: "linux", Arch: "amd64", MemAvailableKB: 2 * 1024 * 1024,
		MemoryKnown: true, DiskAvailableBytes: 15 * 1024 * 1024 * 1024, DiskKnown: true,
		PortsKnown: true, TCPListening: map[uint16]bool{}}
	r, err := Evaluate(base)
	if err != nil || !r.Candidate || len(r.Checks) != 4 || len(r.Warnings) != 1 {
		t.Fatalf("candidate %#v %v", r, err)
	}
	base.TCPListening[443] = true
	r, _ = Evaluate(base)
	if r.Checks[3].Status != "warn" {
		t.Fatal("occupied HTTPS port not reported")
	}
	base.OSRelease = "ID=debian\nVERSION_ID=13\n"
	r, _ = Evaluate(base)
	if r.Candidate || r.Checks[0].Status != "block" {
		t.Fatal("unsupported OS accepted")
	}
	base.OSRelease = ubuntu
	base.MemoryKnown, base.DiskKnown, base.PortsKnown = false, false, false
	r, _ = Evaluate(base)
	for _, c := range r.Checks[1:] {
		if c.Status != "warn" {
			t.Fatalf("unknown check should warn: %v", c)
		}
	}
}

func TestDoctorDoesNotModifyOSFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(path, []byte(ubuntu), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Detect(path)
	if err != nil || len(r.Checks) != 4 || !strings.Contains(strings.Join(r.Warnings, " "), "Development-only") {
		t.Fatalf("bad doctor result %#v %v", r, err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != ubuntu {
		t.Fatal("doctor modified OS fixture")
	}
}
