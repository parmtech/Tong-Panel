// Package backups provides a versioned streaming encryption envelope for
// tenant backup payloads. It does not select, store, rotate, or recover keys.
package backups

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
)

const (
	FormatVersion          byte  = 1
	ChunkSize                    = 64 << 10
	DefaultMaxPlaintextBytes     = int64(64 << 30)
	MaxPlaintextBytes            = int64(64 << 30)
	wrappedKeySize               = 32 + 16
	headerSize                   = 8 + 1 + 12 + wrappedKeySize + 8
)

var (
	ErrInvalidKey     = errors.New("backup wrapping key must be 32 bytes")
	ErrInvalidTenant  = errors.New("invalid backup tenant identifier")
	ErrInvalidOptions = errors.New("invalid backup limits")
	ErrBackupLimit    = errors.New("backup exceeds configured plaintext limit")
	ErrInvalidBackup  = errors.New("invalid or unauthenticated backup envelope")
)

var magic = [8]byte{'T', 'P', 'B', 'K', 'P', '0', '0', '1'}

// Options can lower the hard plaintext bound. A zero limit selects the
// package default. Larger limits are rejected to bound decompression/restore
// staging requirements at the caller.
type Options struct {
	MaxPlaintextBytes int64
}

func (o Options) normalized() (Options, error) {
	if o.MaxPlaintextBytes == 0 {
		o.MaxPlaintextBytes = DefaultMaxPlaintextBytes
	}
	if o.MaxPlaintextBytes < 1 || o.MaxPlaintextBytes > MaxPlaintextBytes {
		return Options{}, ErrInvalidOptions
	}
	return o, nil
}

func validateInputs(dst io.Writer, src io.Reader, tenantID string, wrappingKey []byte, options Options) (Options, error) {
	if dst == nil || src == nil {
		return Options{}, ErrInvalidBackup
	}
	if len(wrappingKey) != 32 {
		return Options{}, ErrInvalidKey
	}
	if len(tenantID) == 0 || len(tenantID) > 128 {
		return Options{}, ErrInvalidTenant
	}
	for _, c := range []byte(tenantID) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return Options{}, ErrInvalidTenant
		}
	}
	return options.normalized()
}

// Encrypt writes a versioned, chunk-authenticated envelope. A fresh random
// data-encryption key is generated for each backup and wrapped with the
// caller-supplied 256-bit key. Tenant identity is authenticated as associated
// data and is not serialized into the archive.
//
// On any error, dst may contain a partial envelope. The caller must write to a
// private temporary object and publish it only after Encrypt returns nil.
func Encrypt(dst io.Writer, src io.Reader, tenantID string, wrappingKey []byte, options Options) error {
	options, err := validateInputs(dst, src, tenantID, wrappingKey, options)
	if err != nil {
		return err
	}

	dataKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return err
	}
	defer wipe(dataKey)

	var wrapNonce [12]byte
	var noncePrefix [8]byte
	if _, err := io.ReadFull(rand.Reader, wrapNonce[:]); err != nil {
		return err
	}
	if _, err := io.ReadFull(rand.Reader, noncePrefix[:]); err != nil {
		return err
	}

	wrapAEAD, err := newAEAD(wrappingKey)
	if err != nil {
		return err
	}
	wrappedKey := wrapAEAD.Seal(nil, wrapNonce[:], dataKey, associatedData(tenantID, recordWrappedKey, 0))
	var header [headerSize]byte
	copy(header[:8], magic[:])
	header[8] = FormatVersion
	copy(header[9:21], wrapNonce[:])
	copy(header[21:21+wrappedKeySize], wrappedKey)
	copy(header[21+wrappedKeySize:], noncePrefix[:])
	if err := writeFull(dst, header[:]); err != nil {
		return err
	}

	dataAEAD, err := newAEAD(dataKey)
	if err != nil {
		return err
	}
	buf := make([]byte, ChunkSize)
	var total int64
	var sequence uint32
	for {
		n, readErr := io.ReadFull(src, buf)
		if n > 0 {
			if int64(n) > options.MaxPlaintextBytes-total {
				return ErrBackupLimit
			}
			if sequence == ^uint32(0) {
				return ErrBackupLimit
			}
			sealed := dataAEAD.Seal(nil, chunkNonce(noncePrefix[:], sequence), buf[:n], associatedData(tenantID, recordData, sequence))
			if err := writeRecord(dst, sealed); err != nil {
				return err
			}
			total += int64(n)
			sequence++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	finalTag := dataAEAD.Seal(nil, chunkNonce(noncePrefix[:], sequence), nil, associatedData(tenantID, recordFinal, sequence))
	if err := writeFull(dst, []byte{0, 0, 0, 0}); err != nil {
		return err
	}
	return writeFull(dst, finalTag)
}

// Decrypt authenticates and streams plaintext from an envelope. It verifies
// tenant binding, every chunk, completeness, limits and the absence of trailing
// data. On error, dst may contain a verified plaintext prefix; callers must
// stage it privately and publish only after Decrypt returns nil.
func Decrypt(dst io.Writer, src io.Reader, tenantID string, wrappingKey []byte, options Options) error {
	options, err := validateInputs(dst, src, tenantID, wrappingKey, options)
	if err != nil {
		return err
	}

	var header [headerSize]byte
	if _, err := io.ReadFull(src, header[:]); err != nil {
		return ErrInvalidBackup
	}
	if string(header[:8]) != string(magic[:]) || header[8] != FormatVersion {
		return ErrInvalidBackup
	}

	wrapAEAD, err := newAEAD(wrappingKey)
	if err != nil {
		return err
	}
	dataKey, err := wrapAEAD.Open(nil, header[9:21], header[21:21+wrappedKeySize], associatedData(tenantID, recordWrappedKey, 0))
	if err != nil || len(dataKey) != 32 {
		return ErrInvalidBackup
	}
	defer wipe(dataKey)

	var noncePrefix [8]byte
	copy(noncePrefix[:], header[21+wrappedKeySize:])
	dataAEAD, err := newAEAD(dataKey)
	if err != nil {
		return ErrInvalidBackup
	}

	var total int64
	var sequence uint32
	var pending []byte
	for {
		var lengthBytes [4]byte
		if _, err := io.ReadFull(src, lengthBytes[:]); err != nil {
			return ErrInvalidBackup
		}
		length := binary.BigEndian.Uint32(lengthBytes[:])
		if length == 0 {
			finalTag := make([]byte, dataAEAD.Overhead())
			if _, err := io.ReadFull(src, finalTag); err != nil {
				return ErrInvalidBackup
			}
			plain, err := dataAEAD.Open(nil, chunkNonce(noncePrefix[:], sequence), finalTag, associatedData(tenantID, recordFinal, sequence))
			if err != nil || len(plain) != 0 {
				return ErrInvalidBackup
			}
			if err := ensureEOF(src); err != nil {
				return err
			}
			if len(pending) > 0 {
				if err := writeFull(dst, pending); err != nil {
					return err
				}
			}
			return nil
		}
		if length <= uint32(dataAEAD.Overhead()) || length > ChunkSize+uint32(dataAEAD.Overhead()) {
			return ErrInvalidBackup
		}
		sealed := make([]byte, int(length))
		if _, err := io.ReadFull(src, sealed); err != nil {
			return ErrInvalidBackup
		}
		plain, err := dataAEAD.Open(nil, chunkNonce(noncePrefix[:], sequence), sealed, associatedData(tenantID, recordData, sequence))
		if err != nil || len(plain) == 0 {
			return ErrInvalidBackup
		}
		if int64(len(plain)) > options.MaxPlaintextBytes-total {
			return ErrBackupLimit
		}
		if len(pending) > 0 {
			if len(pending) != ChunkSize {
				return ErrInvalidBackup
			}
			if err := writeFull(dst, pending); err != nil {
				return err
			}
		}
		pending = plain
		total += int64(len(plain))
		if sequence == ^uint32(0) {
			return ErrBackupLimit
		}
		sequence++
	}
}

const (
	recordWrappedKey byte = 1
	recordData       byte = 2
	recordFinal      byte = 3
)

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func associatedData(tenantID string, recordType byte, sequence uint32) []byte {
	data := make([]byte, 0, 8+1+1+2+len(tenantID)+4)
	data = append(data, magic[:]...)
	data = append(data, FormatVersion, recordType)
	var size [2]byte
	binary.BigEndian.PutUint16(size[:], uint16(len(tenantID)))
	data = append(data, size[:]...)
	data = append(data, tenantID...)
	if recordType == recordData || recordType == recordFinal {
		var seq [4]byte
		binary.BigEndian.PutUint32(seq[:], sequence)
		data = append(data, seq[:]...)
	}
	return data
}

func chunkNonce(prefix []byte, sequence uint32) []byte {
	nonce := make([]byte, 12)
	copy(nonce[:8], prefix)
	binary.BigEndian.PutUint32(nonce[8:], sequence)
	return nonce
}

func writeRecord(dst io.Writer, sealed []byte) error {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(sealed)))
	if err := writeFull(dst, length[:]); err != nil {
		return err
	}
	return writeFull(dst, sealed)
}

func writeFull(dst io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := dst.Write(data)
		if n < 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func ensureEOF(src io.Reader) error {
	var extra [1]byte
	n, err := src.Read(extra[:])
	if n != 0 || err == nil {
		return ErrInvalidBackup
	}
	if err != io.EOF {
		return ErrInvalidBackup
	}
	return nil
}

func wipe(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

