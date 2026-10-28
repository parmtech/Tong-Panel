// Package preflight reports read-only Linux host diagnostics.
// A candidate host is NOT a certified/supported Tong Panel installation target.
package preflight

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"` // pass, warn, or block
	Detail string `json:"detail"`
}

type Report struct {
	OS, Version, Arch string
	Candidate         bool
	Warnings          []string
	Checks            []Check `json:"checks"`
}

func ParseOSRelease(raw string) (string, string, error) {
	fields := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		a, b, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(a)] = strings.Trim(strings.TrimSpace(b), "\"'")
	}
	if err := sc.Err(); err != nil {
		return "", "", err
	}
	id, v := strings.ToLower(fields["ID"]), fields["VERSION_ID"]
	if id == "" || v == "" {
		return "", "", errors.New("missing ID/VERSION_ID")
	}
	return id, v, nil
}

// ParseMemAvailableKB extracts the kernel's MemAvailable, not merely MemFree.
// Unknown or malformed readings must not be treated as zero memory.
func ParseMemAvailableKB(raw string) (uint64, error) {
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 3 && parts[0] == "MemAvailable:" && parts[2] == "kB" {
			v, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil || v == 0 {
				return 0, errors.New("invalid MemAvailable")
			}
			return v, nil
		}
	}
	return 0, errors.New("MemAvailable not found")
}

// ParseListeningTCP reads kernel /proc/net/tcp{,6} snapshots. It does not
// probe or bind host ports. A failed read is a warning, not an empty port list.
func ParseListeningTCP(raw string) (map[uint16]bool, error) {
	listening := make(map[uint16]bool)
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 || !strings.Contains(lines[0], "local_address") {
		return nil, errors.New("invalid proc tcp header")
	}
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		} // kernel race; ignore incomplete record
		if parts[3] != "0A" {
			continue
		} // TCP_LISTEN only
		_, portHex, ok := strings.Cut(parts[1], ":")
		if !ok {
			return nil, errors.New("invalid listening socket")
		}
		p, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid listening port: %w", err)
		}
		listening[uint16(p)] = true
	}
	return listening, nil
}

// Snapshot holds observable host facts. Separating collection from evaluation
// makes unsupported machines, collisions and low resources testable without
// mutating a VPS or needing root. These are *advisory* development thresholds.
type Snapshot struct {
	OSRelease          string
	GOOS, Arch         string
	MemAvailableKB     uint64
	MemoryKnown        bool
	DiskAvailableBytes uint64
	DiskKnown          bool
	TCPListening       map[uint16]bool
	PortsKnown         bool
}

func Evaluate(s Snapshot) (Report, error) {
	id, version, err := ParseOSRelease(s.OSRelease)
	if err != nil {
		return Report{}, err
	}
	r := Report{OS: id, Version: version, Arch: s.Arch, Warnings: []string{"Development-only diagnostic; no OS is certified for Tong Panel installation"}}
	r.Candidate = s.GOOS == "linux" && s.Arch == "amd64" && ((id == "ubuntu" && version == "24.04") || (id == "debian" && version == "12"))
	add := func(id, status, detail string) {
		r.Checks = append(r.Checks, Check{ID: id, Status: status, Detail: detail})
		if status != "pass" {
			r.Warnings = append(r.Warnings, detail)
		}
	}
	if r.Candidate {
		add("os", "pass", "OS is a planned lab candidate, not certified")
	} else {
		add("os", "block", fmt.Sprintf("OS %s %s / %s is outside initial lab targets", id, version, s.Arch))
	}
	if !s.MemoryKnown {
		add("memory", "warn", "Available RAM could not be determined")
	} else if s.MemAvailableKB < 1024*1024 {
		add("memory", "warn", "Available RAM is below the preliminary 1 GiB web-only development guideline")
	} else {
		add("memory", "pass", "Available RAM meets preliminary web-only development guideline")
	}
	if !s.DiskKnown {
		add("disk", "warn", "Available root filesystem space could not be determined")
	} else if s.DiskAvailableBytes < 8*1024*1024*1024 {
		add("disk", "warn", "Free root filesystem space is below the preliminary 8 GiB development guideline")
	} else {
		add("disk", "pass", "Available root filesystem space meets preliminary development guideline")
	}
	if !s.PortsKnown {
		add("ports", "warn", "Listening TCP ports could not be inspected; web-service conflicts unknown")
	} else if s.TCPListening[80] || s.TCPListening[443] {
		add("ports", "warn", "TCP 80 or 443 already has a listener; inspect web stack before installation")
	} else {
		add("ports", "pass", "No TCP listeners on 80/443 seen in /proc snapshot (not proof ports are accessible externally)")
	}
	return r, nil
}

// MutationRequirements makes installation-specific blocking thresholds explicit.
// Callers must use an owner-reviewed policy; this package does not certify OS support.
type MutationRequirements struct {
	MinimumDiskBytes  uint64
	RequiredTCPPorts []uint16
}

// MutationGate is a fail-closed decision over a read-only Snapshot. Allowed only
// means the supplied facts meet the supplied policy; it does not perform or
// authorize a host mutation.
type MutationGate struct {
	Allowed  bool     `json:"allowed"`
	Blockers []string `json:"blockers"`
}

// EvaluateMutationGate reports blockers that must be resolved before a future
// installer or host mutator acts. Missing facts and incomplete policies fail
// closed. The function is pure and never changes host state.
func EvaluateMutationGate(s Snapshot, requirements MutationRequirements) MutationGate {
	gate := MutationGate{Blockers: []string{}}
	report, err := Evaluate(s)
	if err != nil {
		gate.Blockers = append(gate.Blockers, "invalid_os_facts")
	} else if !report.Candidate {
		gate.Blockers = append(gate.Blockers, "unsupported_os")
	}

	if requirements.MinimumDiskBytes == 0 {
		gate.Blockers = append(gate.Blockers, "disk_requirement_missing")
	} else if !s.DiskKnown {
		gate.Blockers = append(gate.Blockers, "disk_unknown")
	} else if s.DiskAvailableBytes < requirements.MinimumDiskBytes {
		gate.Blockers = append(gate.Blockers, "disk_below_minimum")
	}

	if len(requirements.RequiredTCPPorts) == 0 {
		gate.Blockers = append(gate.Blockers, "port_requirements_missing")
	} else if !s.PortsKnown {
		gate.Blockers = append(gate.Blockers, "ports_unknown")
	} else {
		for _, port := range requirements.RequiredTCPPorts {
			if port == 0 {
				gate.Blockers = append(gate.Blockers, "invalid_required_port")
				continue
			}
			if s.TCPListening[port] {
				gate.Blockers = append(gate.Blockers, fmt.Sprintf("tcp_port_in_use:%d", port))
			}
		}
	}

	gate.Allowed = len(gate.Blockers) == 0
	return gate
}

// Detect returns a best-effort, READ-ONLY host report. It never binds ports,
// changes packages, writes files, modifies firewall rules or probes credentials.
func Detect(path string) (Report, error) {
	osRelease, err := os.ReadFile(path)
	if err != nil {
		return Report{}, err
	}
	s := Snapshot{OSRelease: string(osRelease), GOOS: runtime.GOOS, Arch: runtime.GOARCH}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		if kb, err := ParseMemAvailableKB(string(data)); err == nil {
			s.MemAvailableKB, s.MemoryKnown = kb, true
		}
	}
	if runtime.GOOS == "linux" {
		var st syscall.Statfs_t
		if err := syscall.Statfs("/", &st); err == nil {
			s.DiskAvailableBytes = st.Bavail * uint64(st.Bsize)
			s.DiskKnown = true
		}
	}
	listeners := map[uint16]bool{}
	seen := 0
	for _, proc := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		b, err := os.ReadFile(proc)
		if err != nil {
			continue
		}
		ports, err := ParseListeningTCP(string(b))
		if err != nil {
			continue
		}
		for port := range ports {
			listeners[port] = true
		}
		seen++
	}
	s.TCPListening, s.PortsKnown = listeners, seen == 2
	return Evaluate(s)
}
