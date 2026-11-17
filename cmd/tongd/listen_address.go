package main

import (
	"errors"
	"net/netip"
)

const defaultDevelopmentListenAddress = "127.0.0.1:8090"

var errDevelopmentListenAddress = errors.New("invalid development listener configuration")

// parseDevelopmentListenAddress accepts an operator's literal IPv4 loopback
// address only. It never resolves a hostname, selects a wildcard, or falls back
// to the default after an invalid nonempty value.
func parseDevelopmentListenAddress(value string) (string, error) {
	if value == "" {
		return defaultDevelopmentListenAddress, nil
	}
	return validateDevelopmentListenAddress(value, false)
}

// Ephemeral ports remain available to internal lifecycle tests, not operator
// configuration. The same address-family/exposure policy applies in both paths.
func validateDevelopmentListenAddress(value string, allowEphemeral bool) (string, error) {
	address, err := netip.ParseAddrPort(value)
	if err != nil || !address.Addr().Is4() || !address.Addr().IsLoopback() ||
		(address.Port() == 0 && !allowEphemeral) {
		return "", errDevelopmentListenAddress
	}
	return address.String(), nil
}
