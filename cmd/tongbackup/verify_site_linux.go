//go:build linux

package main

import (
	"context"
	"github.com/parmtech/tong-panel/internal/files"
	"io"
)

func verifySiteArchive(src io.Reader, tenant string, key []byte) error {
	_, err := files.VerifyEncryptedGzipTar(context.Background(), src, tenant, key, files.ArchiveLimits{})
	return err
}
