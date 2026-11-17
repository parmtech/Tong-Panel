//go:build linux

package files

import (
	"bytes"
	"context"
	"io"

	"github.com/parmtech/tong-panel/internal/backups"
	"golang.org/x/sys/unix"
)

// RestoreEncryptedGzipTarNew authenticates the entire tenant-bound backup before
// extracting its gzip-tar payload into a new path beneath the pinned root.
// The caller must bind tenantID, root, current authorization and quota; this
// helper does not do so or restore databases/services. Its compressed plaintext
// bound is the archive limit (at most 32 MiB). It holds plaintext in memory and
// clears its buffer on return, but Go cannot promise all copies are erased.
//
// Cancellation is checked during reads and immediately before publication; a
// blocked reader cannot be interrupted by this helper, and cancellation may race
// the atomic rename. If ErrPublishedDurabilityUnconfirmed is returned the
// destination exists but durability is uncertain. ErrPublicationOutcomeUnknown
// means an attempted rename may or may not have published; do not assume rollback
// or release reservations without reconciliation. Other cleanup failures may
// leave private staging debris. Caller must protect the source from replacement
// and require current authority when integrating this primitive.
func (r *Root) RestoreEncryptedGzipTarNew(ctx context.Context, relative string, src io.Reader, tenantID string, key []byte, limits ArchiveLimits) error {
	_, err := r.RestoreEncryptedGzipTarNewWithResult(ctx, relative, src, tenantID, key, limits)
	return err
}

// RestoreEncryptedGzipTarNewWithResult returns publication state and logical
// regular-file bytes, including when post-rename durability is uncertain. It
// neither reserves nor settles quota and does not report physical disk usage.
func (r *Root) RestoreEncryptedGzipTarNewWithResult(ctx context.Context, relative string, src io.Reader, tenantID string, key []byte, limits ArchiveLimits) (ArchivePublishResult, error) {
	var result ArchivePublishResult
	err := r.restoreEncryptedGzipTarNewWithResult(ctx, relative, src, tenantID, key, limits, &result)
	return result, err
}

func (r *Root) restoreEncryptedGzipTarNewWithResult(ctx context.Context, relative string, src io.Reader, tenantID string, key []byte, limits ArchiveLimits, result *ArchivePublishResult) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil {
		return ErrClosed
	}
	if !validRelativePath(relative) {
		return ErrInvalidPath
	}
	if src == nil {
		return backups.ErrInvalidBackup
	}
	normalized, err := limits.withDefaults()
	if err != nil {
		return err
	}
	r.mu.RLock()
	open := r.open
	r.mu.RUnlock()
	if !open {
		return ErrClosed
	}
	var archive bytes.Buffer
	defer func() { clear(archive.Bytes()) }()
	err = backups.Decrypt(&archive, contextBackupReader{ctx: ctx, src: src}, tenantID, key, backups.Options{MaxPlaintextBytes: normalized.CompressedBytes})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.extractGzipTarNewWithResult(ctx, relative, archive.Bytes(), normalized, false, unix.Fsync, unix.Renameat2, result)
}

type contextBackupReader struct {
	ctx context.Context
	src io.Reader
}

func (r contextBackupReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.src.Read(p)
	if e := r.ctx.Err(); e != nil {
		return n, e
	}
	return n, err
}
