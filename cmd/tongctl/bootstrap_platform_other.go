//go:build !linux

package main

import (
	"errors"
	"os"
)

var errBootstrapUnsupportedPlatform = errors.New("local owner bootstrap is supported only on Linux")

func requireBootstrapPlatform() error { return errBootstrapUnsupportedPlatform }

func requireSecureOwner(os.FileInfo) error { return errBootstrapUnsupportedPlatform }
