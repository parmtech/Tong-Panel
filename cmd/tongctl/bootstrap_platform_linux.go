//go:build linux

package main

import (
	"errors"
	"os"
	"syscall"
)

func requireBootstrapPlatform() error { return nil }

func requireSecureOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("database and parent must be owned by the local operator")
	}
	return nil
}
