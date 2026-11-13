//go:build !linux

package main

import (
	"errors"
	"io"
)

func runRestoreSiteLab([]string, io.Reader) error {
	return errors.New("restore-site-lab is supported only on Linux")
}
