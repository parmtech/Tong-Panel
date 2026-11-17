package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/preflight"
	"github.com/parmtech/tong-panel/internal/supportbundle"
)

func TestRunSupportBundleMasksCollectorErrorsAndRetainsPartialFacts(t *testing.T) {
	const sentinel = "never-export-error-path-or-password"
	t.Setenv("TONG_DEV_TOKEN", sentinel)
	var output bytes.Buffer
	err := runSupportBundle(nil, &output, func() (preflight.Snapshot, error) {
		return preflight.Snapshot{OSRelease: "ID=" + sentinel + "\nVERSION_ID=" + sentinel,
			GOOS: "linux", Arch: "amd64", MemoryKnown: true, MemAvailableKB: 2 * 1024 * 1024,
			TCPListening: map[uint16]bool{80: true, 2222: true}}, errors.New(sentinel)
	})
	if err != nil {
		t.Fatal("partial support export should succeed")
	}
	if strings.Contains(output.String(), sentinel) || strings.Contains(output.String(), "2222") || len(output.Bytes()) > 4096 {
		t.Fatal("support export escaped its allowlist")
	}
	var bundle supportbundle.Bundle
	if err := json.Unmarshal(output.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Collection.Status != "partial" || bundle.Diagnostics.Memory != "adequate" ||
		bundle.Diagnostics.Disk != "unknown" || bundle.Diagnostics.WebPorts != "unknown" {
		t.Fatal("missing collection status or known partial facts")
	}
}

func TestRunSupportBundleMapsOnlyHTTPAndHTTPSPortObservations(t *testing.T) {
	tests := []struct {
		name           string
		portsKnown     bool
		listeners      map[uint16]bool
		wantWebPorts   string
		wantCollection supportbundle.Collection
	}{
		{
			name:           "HTTP listener",
			portsKnown:     true,
			listeners:      map[uint16]bool{80: true},
			wantWebPorts:   "in_use",
			wantCollection: supportbundle.Collection{Status: "complete", Code: "facts_collected"},
		},
		{
			name:           "HTTPS listener",
			portsKnown:     true,
			listeners:      map[uint16]bool{443: true},
			wantWebPorts:   "in_use",
			wantCollection: supportbundle.Collection{Status: "complete", Code: "facts_collected"},
		},
		{
			name:           "unlisted listener is ignored",
			portsKnown:     true,
			listeners:      map[uint16]bool{2222: true},
			wantWebPorts:   "not_seen",
			wantCollection: supportbundle.Collection{Status: "complete", Code: "facts_collected"},
		},
		{
			name:           "unknown observation stays unknown",
			listeners:      map[uint16]bool{80: true, 443: true},
			wantWebPorts:   "unknown",
			wantCollection: supportbundle.Collection{Status: "partial", Code: "facts_incomplete"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := runSupportBundle(nil, &output, func() (preflight.Snapshot, error) {
				return preflight.Snapshot{
					OSRelease: "ID=debian\nVERSION_ID=12\n", GOOS: "linux", Arch: "amd64",
					MemoryKnown: true, MemAvailableKB: 2 * 1024 * 1024,
					DiskKnown: true, DiskAvailableBytes: 12 * 1024 * 1024 * 1024,
					PortsKnown: test.portsKnown, TCPListening: test.listeners,
				}, nil
			})
			if err != nil {
				t.Fatal("support export failed")
			}
			var bundle supportbundle.Bundle
			if err := json.Unmarshal(output.Bytes(), &bundle); err != nil {
				t.Fatal(err)
			}
			if bundle.Diagnostics.WebPorts != test.wantWebPorts || bundle.Collection != test.wantCollection {
				t.Fatalf("web ports = %q, collection = %#v; want %q, %#v",
					bundle.Diagnostics.WebPorts, bundle.Collection, test.wantWebPorts, test.wantCollection)
			}
		})
	}
}

func TestRunSupportBundleRejectsArgumentsBeforeCollecting(t *testing.T) {
	const sentinel = "credential-in-invalid-argument"
	var output bytes.Buffer
	err := runSupportBundle([]string{"--path=" + sentinel}, &output, func() (preflight.Snapshot, error) {
		t.Fatal("invalid arguments triggered collection")
		return preflight.Snapshot{}, nil
	})
	if !errors.Is(err, errSupportBundleUsage) || strings.Contains(err.Error(), sentinel) || output.Len() != 0 {
		t.Fatal("invalid support arguments must return generic usage without output")
	}
}

type supportFailingWriter struct{}

func (supportFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("secret-output-path")
}

type supportShortWriter struct{}

func (supportShortWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func TestRunSupportBundleReportsWriterFailureWithoutItsMessage(t *testing.T) {
	for _, writer := range []io.Writer{supportFailingWriter{}, supportShortWriter{}} {
		err := runSupportBundle(nil, writer, func() (preflight.Snapshot, error) { return preflight.Snapshot{}, nil })
		if !errors.Is(err, errSupportBundleOutput) || strings.Contains(err.Error(), "secret-output-path") {
			t.Fatal("output failure must be generic and detect short writes")
		}
	}
}
