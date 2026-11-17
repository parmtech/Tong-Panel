package main

import (
	"errors"
	"strings"
	"testing"
)

func TestDevelopmentListenAddressDefaultsAndLiteralLoopback(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"", "127.0.0.1:8090"},
		{"127.0.0.1:18090", "127.0.0.1:18090"},
		{"127.0.0.2:8090", "127.0.0.2:8090"},
		{"127.255.255.255:65535", "127.255.255.255:65535"},
		{"127.1.2.3:1", "127.1.2.3:1"},
		{"127.0.0.1:08090", "127.0.0.1:8090"},
	} {
		got, err := parseDevelopmentListenAddress(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("literal loopback/default decision: got=%q error=%v", got, err)
		}
	}
}

func TestDevelopmentListenAddressRejectsAmbiguousOrExposedAddresses(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"missing-host", ":8090"}, {"wildcard", "0.0.0.0:8090"},
		{"star", "*:8090"}, {"nonloopback", "192.0.2.1:8090"},
		{"hostname", "localhost:8090"}, {"hostname-with-dot", "localhost.:8090"},
		{"IPv6-loopback", "[::1]:8090"}, {"IPv6-wildcard", "[::]:8090"},
		{"mapped-IPv6", "[::ffff:127.0.0.1]:8090"},
		{"IPv6-zone", "[::1%lo]:8090"}, {"IPv4-brackets", "[127.0.0.1]:8090"},
		{"abbreviated-IP", "127.1:8090"}, {"integer-IP", "2130706433:8090"},
		{"hex-IP", "0x7f000001:8090"}, {"leading-zero-IP", "127.000.000.001:8090"},
		{"missing-port", "127.0.0.1"}, {"empty-port", "127.0.0.1:"},
		{"zero-port", "127.0.0.1:0"}, {"large-port", "127.0.0.1:65536"},
		{"negative-port", "127.0.0.1:-1"}, {"service-port", "127.0.0.1:http"},
		{"space-prefix", " 127.0.0.1:8090"}, {"space-suffix", "127.0.0.1:8090 "},
		{"URL", "http://127.0.0.1:8090"}, {"path", "127.0.0.1:8090/private"},
		{"credential-URL", "http://secret-bind-canary@127.0.0.1:8090"},
		{"control", "127.0.0.1:8090\nsecret-bind-canary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address, err := parseDevelopmentListenAddress(tc.value)
			if address != "" || !errors.Is(err, errDevelopmentListenAddress) || strings.Contains(err.Error(), "secret-bind-canary") {
				t.Fatal("invalid bind was accepted, fell back, or disclosed input")
			}
		})
	}
}

func TestDevelopmentListenAddressEphemeralPortsAreInternalOnly(t *testing.T) {
	if _, err := parseDevelopmentListenAddress("127.0.0.1:0"); !errors.Is(err, errDevelopmentListenAddress) {
		t.Fatal("operator configuration accepted an ephemeral port")
	}
	if value, err := validateDevelopmentListenAddress("127.0.0.1:0", true); err != nil || value != "127.0.0.1:0" {
		t.Fatal("internal lifecycle tests lost ephemeral-port support")
	}
	for _, value := range []string{":0", "0.0.0.0:0", "[::1]:0", "[::ffff:127.0.0.1]:0"} {
		if _, err := validateDevelopmentListenAddress(value, true); !errors.Is(err, errDevelopmentListenAddress) {
			t.Fatal("ephemeral mode bypassed the exposure/family restriction")
		}
	}
}
