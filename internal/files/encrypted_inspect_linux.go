//go:build linux

package files

import (
	"bytes"
	"context"
	"io"

	"github.com/parmtech/tong-panel/internal/backups"
)

// VerifyEncryptedGzipTar authenticates a tenant-bound encrypted stream and
// validates its entire gzip-tar using the same parser and bounds as extraction,
// without creating a directory or writing plaintext. The returned count is
// logical regular-file bytes, not disk usage or authorization. This does not
// prove a different stream will restore, bind a tenant to a root, inspect file
// semantics, reserve quota, or authorize a later operation. Plaintext archive
// data is buffered in memory within the compressed cap; clearing is best effort.
// Cancellation is checked during input reads and before/after archive parsing;
// a blocked reader or an already-running parser cannot be forcibly interrupted.
func VerifyEncryptedGzipTar(ctx context.Context, src io.Reader, tenantID string, key []byte, limits ArchiveLimits) (int64, error) {
	if ctx == nil {
		return 0, ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if src == nil {
		return 0, backups.ErrInvalidBackup
	}
	normalized, err := limits.withDefaults()
	if err != nil {
		return 0, err
	}
	var archive bytes.Buffer
	defer func() { clear(archive.Bytes()) }()
	if err := backups.Decrypt(&archive, contextBackupReader{ctx: ctx, src: src}, tenantID, key, backups.Options{MaxPlaintextBytes: normalized.CompressedBytes}); err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var logicalBytes int64
	if err := parseGzipTar(-1, archive.Bytes(), normalized, &logicalBytes, true); err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return logicalBytes, nil
}
