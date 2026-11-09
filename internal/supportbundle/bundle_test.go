package supportbundle

import (
	"encoding/json"
	"strings"
	"testing"
)

func completeFacts() Facts {
	return Facts{
		OSRelease: "ID=ubuntu\nVERSION_ID=\"24.04\"\n", GOOS: "linux", Arch: "amd64",
		MemoryKnown: true, MemAvailableKB: 2 * 1024 * 1024,
		DiskKnown: true, DiskAvailableBytes: 12 * 1024 * 1024 * 1024,
		PortsKnown: true,
	}
}

func TestBuildRetainsOnlyApprovedFactsAndIsDeterministic(t *testing.T) {
	facts := completeFacts()
	facts.OSRelease += "PRETTY_NAME=private-hostname\nSECRET=private-token\n"
	first, err := json.Marshal(Build(facts))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(Build(facts))
	if string(first) != string(second) || len(first) > 4096 {
		t.Fatal("support export must be deterministic and bounded")
	}
	for _, forbidden := range []string{"private-hostname", "private-token", "PRETTY_NAME", "SECRET", "MemAvailableKB", "2147483648", "12884901888"} {
		if strings.Contains(string(first), forbidden) {
			t.Fatal("support export included non-allowlisted input")
		}
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(first, &object); err != nil || len(object) != 5 {
		t.Fatal("unexpected top-level export schema")
	}
	for _, key := range []string{"schema_version", "notice", "collection", "platform", "diagnostics"} {
		if _, exists := object[key]; !exists {
			t.Fatal("missing expected schema field")
		}
	}
	bundle := Build(facts)
	if bundle.SchemaVersion != 1 || bundle.Notice != Notice || bundle.Collection.Status != "complete" || bundle.Collection.Code != "facts_collected" ||
		bundle.Platform != (Platform{OS: "linux", Arch: "amd64", Distribution: "ubuntu", Version: "24.04"}) ||
		bundle.Diagnostics != (Diagnostics{OS: "lab_candidate", Memory: "adequate", Disk: "adequate", WebPorts: "not_seen"}) {
		t.Fatal("unexpected normalised support facts")
	}
}

func TestBuildDoesNotExportInjectedIdentifiersOrEnvironment(t *testing.T) {
	const sentinel = "credential-canary-never-export"
	t.Setenv("TONG_DEV_TOKEN", sentinel)
	facts := completeFacts()
	facts.OSRelease = "ID=" + sentinel + "\nVERSION_ID=" + sentinel + "\nNAME=" + sentinel + "\n"
	facts.GOOS, facts.Arch = sentinel, sentinel
	encoded, err := json.Marshal(Build(facts))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), sentinel) {
		t.Fatal("credential canary escaped the support allowlist")
	}
	if Build(facts).Platform != (Platform{OS: "other", Arch: "other", Distribution: "other", Version: "other"}) {
		t.Fatal("unrecognised strings must become static other markers")
	}
}

func TestDistributionPairsMalformedAndOversizeFacts(t *testing.T) {
	for _, test := range []struct {
		name, raw, distribution, version, status string
	}{
		{"debian", "ID='debian'\nVERSION_ID=12", "debian", "12", "lab_candidate"},
		{"unsupported version", "ID=ubuntu\nVERSION_ID=private-token", "ubuntu", "other", "outside_lab_targets"},
		{"crossed version", "ID=debian\nVERSION_ID=24.04", "debian", "other", "outside_lab_targets"},
		{"unrecognised", "ID=private-token\nVERSION_ID=12", "other", "other", "outside_lab_targets"},
		{"missing", "", "unknown", "unknown", "unknown"},
		{"malformed", "ID=ubuntu", "unknown", "unknown", "unknown"},
		{"duplicate", "ID=ubuntu\nID=debian\nVERSION_ID=12", "unknown", "unknown", "unknown"},
		{"unclosed quote", "ID=ubuntu\nVERSION_ID=\"24.04", "unknown", "unknown", "unknown"},
		{"oversize", strings.Repeat("x", 64*1024+1), "unknown", "unknown", "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := completeFacts()
			facts.OSRelease = test.raw
			bundle := Build(facts)
			if bundle.Platform.Distribution != test.distribution || bundle.Platform.Version != test.version || bundle.Diagnostics.OS != test.status {
				t.Fatal("unexpected allowlisted OS classification")
			}
		})
	}
}

func TestBuildKeepsUnknownAndPartialFactsDistinctFromAdequate(t *testing.T) {
	unknown := Build(Facts{})
	if unknown.Collection != (Collection{Status: "partial", Code: "facts_incomplete"}) ||
		unknown.Diagnostics != (Diagnostics{OS: "unknown", Memory: "unknown", Disk: "unknown", WebPorts: "unknown"}) {
		t.Fatal("unknown host facts must not look adequate")
	}
	facts := completeFacts()
	facts.CollectionFailed = true
	facts.MemoryKnown = false
	facts.MemAvailableKB = 99 * 1024 * 1024 // ignored while unknown
	facts.DiskAvailableBytes = 1
	facts.TCP443Listening = true
	bundle := Build(facts)
	if bundle.Collection.Status != "partial" || bundle.Diagnostics.Memory != "unknown" || bundle.Diagnostics.Disk != "low" || bundle.Diagnostics.WebPorts != "in_use" {
		t.Fatal("partial known readings must survive without inventing missing facts")
	}
	facts.MemoryKnown = true
	facts.MemAvailableKB = 1
	facts.PortsKnown = false
	bundle = Build(facts)
	if bundle.Diagnostics.Memory != "low" || bundle.Diagnostics.WebPorts != "unknown" {
		t.Fatal("unknown port collection must not claim absence of a listener")
	}
}

func TestBuildDoesNotCertifyOtherRuntimeTargets(t *testing.T) {
	for _, change := range []func(*Facts){
		func(f *Facts) { f.Arch = "arm64" },
		func(f *Facts) { f.GOOS = "darwin" },
	} {
		facts := completeFacts()
		change(&facts)
		if Build(facts).Diagnostics.OS != "outside_lab_targets" {
			t.Fatal("other runtime targets must not become lab candidates")
		}
	}
}
