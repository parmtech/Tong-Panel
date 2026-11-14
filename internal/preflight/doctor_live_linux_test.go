//go:build linux

package preflight

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

// This uses a temporary unprivileged loopback listener and the local kernel's
// real proc tables. It does not bind a product port or change host services.
func TestLiveKernelTCPObservationBlocksOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	port := uint16(listener.Addr().(*net.TCPAddr).Port)
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		ports, err := ParseListeningTCP(string(data))
		if err != nil {
			t.Fatalf("parse real %s: %v", path, err)
		}
		if path == "/proc/net/tcp" && !ports[port] {
			t.Fatalf("real tcp table omitted loopback listener %d", port)
		}
	}
	snapshot, err := DetectSnapshot("/etc/os-release")
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.PortsKnown || !snapshot.TCPListening[port] {
		t.Fatalf("snapshot did not observe listener: known=%v, present=%v", snapshot.PortsKnown, snapshot.TCPListening[port])
	}
	gate := EvaluateMutationGate(snapshot, MutationRequirements{MinimumDiskBytes: 1, RequiredTCPPorts: []uint16{port}})
	want := fmt.Sprintf("tcp_port_in_use:%d", port)
	if gate.Allowed || !strings.Contains(","+strings.Join(gate.Blockers, ",")+",", ","+want+",") {
		t.Fatalf("occupied port must block: %+v", gate)
	}
}
