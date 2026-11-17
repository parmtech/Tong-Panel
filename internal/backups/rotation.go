package backups

import (
	"io"
)

// RotateWrappingKey decrypts a backup with oldWrappingKey and encrypts it
// with newWrappingKey, generating a fresh data key and nonce set. Plaintext is
// streamed through a bounded pipe and never written to disk by this function.
// dst may contain a partial envelope on error; callers must stage it privately
// and publish it only after RotateWrappingKey returns nil.
func RotateWrappingKey(dst io.Writer, src io.Reader, tenantID string, oldWrappingKey, newWrappingKey []byte, options Options) error {
	options, err := validateInputs(dst, src, tenantID, oldWrappingKey, options)
	if err != nil {
		return err
	}
	if len(newWrappingKey) != 32 {
		return ErrInvalidKey
	}

	plainReader, plainWriter := io.Pipe()
	decryptResult := make(chan error, 1)
	go func() {
		err := Decrypt(plainWriter, src, tenantID, oldWrappingKey, options)
		if err != nil {
			_ = plainWriter.CloseWithError(err)
		} else {
			_ = plainWriter.Close()
		}
		decryptResult <- err
	}()

	encryptErr := Encrypt(dst, plainReader, tenantID, newWrappingKey, options)
	if encryptErr != nil {
		_ = plainReader.CloseWithError(encryptErr)
	} else {
		_ = plainReader.Close()
	}
	decryptErr := <-decryptResult
	if encryptErr != nil {
		return encryptErr
	}
	return decryptErr
}

// RotateWrappingKeyFile stages a complete key rotation beside path and
// publishes it only after the source envelope is authenticated and the new
// envelope is fully written. The caller must choose a directory inaccessible
// to tenants and other untrusted local users.
func RotateWrappingKeyFile(path string, src io.Reader, tenantID string, oldWrappingKey, newWrappingKey []byte, options Options) error {
	return stageAndPublish(path, func(dst io.Writer) error {
		return RotateWrappingKey(dst, src, tenantID, oldWrappingKey, newWrappingKey, options)
	})
}
