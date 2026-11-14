//go:build !linux

package main

import (
	"errors"
	"io"
)

func runInspectRestoreLab([]string, io.Writer) error {
	return errors.New("inspect-restore-lab is supported only on Linux")
}
