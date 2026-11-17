//go:build !linux

package main

import (
	"errors"
	"io"
)

func verifySiteArchive(io.Reader, string, []byte) error {
	return errors.New("verify-site is supported only on Linux")
}
