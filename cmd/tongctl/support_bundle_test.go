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
	if bundle.Collection.Status != "partial" || bundle.Diagnostics.Memory != "adequate" || bundle.Diagnostics.WebPorts != "unknown" {
		t.Fatal("missing collection status or known partial facts")
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
