// Package supportbundle builds a fixed allowlist of read-only diagnostic facts.
// It deliberately has no collectors, environment access, or free-form output.
package supportbundle

import (
	"bufio"
	"strings"
)

const Notice = "Development-only read-only support facts; no installation or support certification."

// Facts is internal input, not an export schema. Raw OS metadata and collector
// errors must never be marshalled directly into a support bundle.
type Facts struct {
	OSRelease, GOOS, Arch string
	CollectionFailed      bool
	MemoryKnown           bool
	MemAvailableKB        uint64
	DiskKnown             bool
	DiskAvailableBytes    uint64
	PortsKnown            bool
	TCP80Listening        bool
	TCP443Listening       bool
}

type Collection struct {
	Status string `json:"status"`
	Code   string `json:"code"`
}

type Platform struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
}

type Diagnostics struct {
	OS       string `json:"os"`
	Memory   string `json:"memory"`
	Disk     string `json:"disk"`
	WebPorts string `json:"web_ports"`
}

type Bundle struct {
	SchemaVersion int         `json:"schema_version"`
	Notice        string      `json:"notice"`
	Collection    Collection  `json:"collection"`
	Platform      Platform    `json:"platform"`
	Diagnostics   Diagnostics `json:"diagnostics"`
}

// Build never returns raw input strings, exact resource readings, listening
// addresses, arbitrary ports, paths or underlying error messages. Every output
// string is a literal from this package. Unknown facts remain unknown.
func Build(f Facts) Bundle {
	b := Bundle{
		SchemaVersion: 1, Notice: Notice,
		Collection:  Collection{Status: "complete", Code: "facts_collected"},
		Platform:    Platform{OS: "unknown", Arch: "unknown", Distribution: "unknown", Version: "unknown"},
		Diagnostics: Diagnostics{OS: "unknown", Memory: "unknown", Disk: "unknown", WebPorts: "unknown"},
	}
	switch f.GOOS {
	case "":
	case "linux":
		b.Platform.OS = "linux"
	default:
		b.Platform.OS = "other"
	}
	switch f.Arch {
	case "":
	case "amd64", "arm64":
		b.Platform.Arch = f.Arch // only these exact literals pass the allowlist
	default:
		b.Platform.Arch = "other"
	}
	b.Platform.Distribution, b.Platform.Version = distribution(f.OSRelease)
	if b.Platform.OS != "unknown" && b.Platform.Arch != "unknown" && b.Platform.Distribution != "unknown" {
		b.Diagnostics.OS = "outside_lab_targets"
		if b.Platform.OS == "linux" && b.Platform.Arch == "amd64" &&
			((b.Platform.Distribution == "ubuntu" && b.Platform.Version == "24.04") ||
				(b.Platform.Distribution == "debian" && b.Platform.Version == "12")) {
			b.Diagnostics.OS = "lab_candidate"
		}
	}
	if f.MemoryKnown {
		b.Diagnostics.Memory = "adequate"
		if f.MemAvailableKB < 1024*1024 {
			b.Diagnostics.Memory = "low"
		}
	}
	if f.DiskKnown {
		b.Diagnostics.Disk = "adequate"
		if f.DiskAvailableBytes < 8*1024*1024*1024 {
			b.Diagnostics.Disk = "low"
		}
	}
	if f.PortsKnown {
		b.Diagnostics.WebPorts = "not_seen"
		if f.TCP80Listening || f.TCP443Listening {
			b.Diagnostics.WebPorts = "in_use"
		}
	}
	if f.CollectionFailed || b.Diagnostics.OS == "unknown" || !f.MemoryKnown || !f.DiskKnown || !f.PortsKnown {
		b.Collection = Collection{Status: "partial", Code: "facts_incomplete"}
	}
	return b
}

// Only the recognised lab pairs retain a version. Unsupported versions and
// unrecognised IDs are replaced, not truncated or included as diagnostic text.
func distribution(raw string) (string, string) {
	if len(raw) == 0 || len(raw) > 64*1024 {
		return "unknown", "unknown"
	}
	var id, version string
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		key = strings.TrimSpace(key)
		if !ok || (key != "ID" && key != "VERSION_ID") {
			continue
		}
		if seen[key] {
			return "unknown", "unknown"
		}
		seen[key] = true
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") {
			if len(value) < 2 || value[len(value)-1] != value[0] {
				return "unknown", "unknown"
			}
			value = value[1 : len(value)-1]
		}
		if key == "ID" {
			id = strings.ToLower(value)
		} else {
			version = value
		}
	}
	if scanner.Err() != nil || id == "" || version == "" {
		return "unknown", "unknown"
	}
	switch id {
	case "ubuntu":
		if version == "24.04" {
			return "ubuntu", "24.04"
		}
		return "ubuntu", "other"
	case "debian":
		if version == "12" {
			return "debian", "12"
		}
		return "debian", "other"
	default:
		return "other", "other"
	}
}
