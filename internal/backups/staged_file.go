package backups

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const stagedFilePattern = ".tong-backup-stage-*"

// EncryptFile writes an encrypted envelope to a private temporary file beside
// path, syncs and closes it, then publishes it with an atomic no-overwrite
// hard link. The caller must choose a directory inaccessible to tenants and
// other untrusted local users. The destination is never visible until the
// complete envelope has been written successfully.
func EncryptFile(path string, src io.Reader, tenantID string, wrappingKey []byte, options Options) error {
	return stageAndPublish(path, func(dst io.Writer) error {
		return Encrypt(dst, src, tenantID, wrappingKey, options)
	})
}

// DecryptFile authenticates an entire envelope into a private temporary file
// and publishes plaintext only after all records, the final marker, and EOF
// have been validated. The caller must choose a directory inaccessible to
// tenants and other untrusted local users. Existing destinations are never
// replaced.
func DecryptFile(path string, src io.Reader, tenantID string, wrappingKey []byte, options Options) error {
	return stageAndPublish(path, func(dst io.Writer) error {
		return Decrypt(dst, src, tenantID, wrappingKey, options)
	})
}

func stageAndPublish(path string, write func(io.Writer) error) error {
	if path == "" || write == nil {
		return ErrInvalidBackup
	}
	path = filepath.Clean(path)
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return ErrInvalidBackup
	}
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("backup staging parent must be a real directory")
	}

	tmp, err := os.CreateTemp(dir, stagedFilePattern)
	if err != nil {
		return err
	}
	tempPath := tmp.Name()
	defer os.Remove(tempPath)

	writeErr := write(tmp)
	if writeErr == nil {
		writeErr = tmp.Sync()
	}
	closeErr := tmp.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}

	// A same-directory hard link is atomic and fails if path already exists,
	// including when it names a symlink. This avoids a check-then-rename race
	// and prevents silently replacing an earlier backup or restore target.
	if err := os.Link(tempPath, path); err != nil {
		return err
	}
	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("published backup file but could not remove private staging link: %w", err)
	}
	return nil
}
