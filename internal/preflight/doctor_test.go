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

func TestListeningPortParsingRejectsIncompleteRecords(t *testing.T) {
	const header = "  sl  local_address rem_address st\n"
	for _, row := range []string{
		"0:",
		"0: 00000000:0050",
		"0: 00000000:0050 00000000:0000",
	} {
		if ports, err := ParseListeningTCP(header + row + "\n"); err == nil || ports != nil {
			t.Fatalf("incomplete record %q reported ports=%v err=%v", row, ports, err)
		}
	}
	ports, err := ParseListeningTCP(header + "0: 00000000:0050 00000000:0000 0A 0:0\n\n")
	if err != nil || !ports[80] {
		t.Fatalf("valid listening row: ports=%v err=%v", ports, err)
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

func TestListeningParserKernelTCPHeaders(t *testing.T) {
	// tcp4_seq_show and tcp6_seq_show use different remote-address labels.
	// Keep the kernel spellings here rather than sharing a synthetic header.
	for _, tc := range []struct {
		name, header, address string
	}{
		{"tcp", "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n", "00000000"},
		{"tcp6", "  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n", "00000000000000000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ports, err := ParseListeningTCP(tc.header)
			if err != nil || len(ports) != 0 {
				t.Fatalf("empty kernel table: ports=%v err=%v", ports, err)
			}
			row := func(port, state string) string {
				return "0: " + tc.address + ":" + port + " " + tc.address + ":0000 " + state + " 0:0 00:0 0 0 0 0\n"
			}
			ports, err = ParseListeningTCP(tc.header + row("0050", "0A") + row("01BB", "0A") + row("24E3", "01"))
			if err != nil || len(ports) != 2 || !ports[80] || !ports[443] {
				t.Fatalf("kernel listeners: ports=%v err=%v", ports, err)
			}
			gate := EvaluateMutationGate(Snapshot{
				OSRelease: ubuntu, GOOS: "linux", Arch: "amd64",
				DiskKnown: true, DiskAvailableBytes: 1,
				PortsKnown: true, TCPListening: ports,
			}, MutationRequirements{MinimumDiskBytes: 1, RequiredTCPPorts: []uint16{80, 443}})
			if gate.Allowed || strings.Join(gate.Blockers, ",") != "tcp_port_in_use:80,tcp_port_in_use:443" {
				t.Fatalf("occupied web ports must block: %+v", gate)
			}
			if ports, err := ParseListeningTCP(tc.header + row("0050", "0A") + row("01BB", "ZZ")); err == nil || ports != nil {
				t.Fatalf("corrupt table must remain unknown: ports=%v err=%v", ports, err)
			}
		})
	}
	for _, header := range []string{
		"sl local_address unexpected_address st\n",
		"sl local_address remote_address state\n",
		"sl local_address remote_address\n",
	} {
		if ports, err := ParseListeningTCP(header); err == nil || ports != nil {
			t.Fatalf("invalid header %q accepted: ports=%v err=%v", header, ports, err)
		}
	}
}

func TestListeningParserFailsClosedOnMalformedNonListeningRows(t *testing.T) {
	const header = "sl local_address rem_address st\n"
	for _, row := range []string{
		"0: 00000000:0050 00000000:0000 ZZ",
		"0: 00000000:0050 00000000:0000 00",
		"0: 00000000:0050 00000000:0000 0D",
		"0: 00000000:0050 00000000:0000 1",
		"0: 00000000:0050 00000000:0000 +A",
		"0: invalid:0050 00000000:0000 01",
		"0: 0000000Z:0050 00000000:0000 01",
		"0: 00000000:50 00000000:0000 01",
		"0: 00000000:GGGG 00000000:0000 01",
		"0: 00000000:+050 00000000:0000 01",
	} {
		if ports, err := ParseListeningTCP(header + row); err == nil || ports != nil {
			t.Errorf("malformed row %q reported ports=%v err=%v", row, ports, err)
		}
	}
	if ports, err := ParseListeningTCP(header + "0: 00000000:0050 00000000:0000 0A\n1: 00000000:01BB 00000000:0000 ZZ\n"); err == nil || ports != nil {
		t.Fatalf("partial snapshot reported ports=%v err=%v", ports, err)
	}
	ipv6Header := strings.Replace(header, "rem_address", "remote_address", 1)
	ports, err := ParseListeningTCP(ipv6Header + "0: 00000000000000000000000000000000:01BB 00000000000000000000000000000000:0000 0A\n")
	if err != nil || !ports[443] {
		t.Fatalf("valid IPv6 listener: ports=%v err=%v", ports, err)
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

func TestMutationGateFailsClosedBeforeHostChanges(t *testing.T) {
	valid := Snapshot{
		OSRelease: ubuntu, GOOS: "linux", Arch: "amd64",
		DiskAvailableBytes: 12 * 1024 * 1024 * 1024, DiskKnown: true,
		TCPListening: map[uint16]bool{}, PortsKnown: true,
	}
	requirements := MutationRequirements{MinimumDiskBytes: 8 * 1024 * 1024 * 1024, RequiredTCPPorts: []uint16{80, 443}}
	if gate := EvaluateMutationGate(valid, requirements); !gate.Allowed || len(gate.Blockers) != 0 {
		t.Fatalf("valid candidate gate = %#v, want allowed", gate)
	}

	assertBlocked := func(name, want string, snapshot Snapshot, policy MutationRequirements) {
		t.Helper()
		gate := EvaluateMutationGate(snapshot, policy)
		if gate.Allowed {
			t.Errorf("%s gate allowed mutation: %#v", name, gate)
		}
		for _, blocker := range gate.Blockers {
			if blocker == want {
				return
			}
		}
		t.Errorf("%s blockers = %#v, want %q", name, gate.Blockers, want)
	}

	unsupported := valid
	unsupported.OSRelease = "ID=debian\nVERSION_ID=13\n"
	assertBlocked("unsupported OS", "unsupported_os", unsupported, requirements)

	lowDisk := valid
	lowDisk.DiskAvailableBytes = requirements.MinimumDiskBytes - 1
	assertBlocked("low disk", "disk_below_minimum", lowDisk, requirements)

	occupied := valid
	occupied.TCPListening = map[uint16]bool{443: true}
	assertBlocked("occupied port", "tcp_port_in_use:443", occupied, requirements)

	unknownDisk := valid
	unknownDisk.DiskKnown = false
	assertBlocked("unknown disk", "disk_unknown", unknownDisk, requirements)

	unknownPorts := valid
	unknownPorts.PortsKnown = false
	assertBlocked("unknown ports", "ports_unknown", unknownPorts, requirements)

	assertBlocked("missing policy", "disk_requirement_missing", valid, MutationRequirements{})
	assertBlocked("missing port requirements", "port_requirements_missing", valid,
		MutationRequirements{MinimumDiskBytes: requirements.MinimumDiskBytes})
	assertBlocked("zero port", "invalid_required_port", valid,
		MutationRequirements{MinimumDiskBytes: requirements.MinimumDiskBytes, RequiredTCPPorts: []uint16{0}})

	malformedOS := valid
	malformedOS.OSRelease = "not an os release"
	assertBlocked("malformed OS facts", "invalid_os_facts", malformedOS, requirements)
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
