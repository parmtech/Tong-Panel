package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	mfaSecretMinBytes = 20
	mfaSecretMaxBytes = 64
	mfaMaxCiphertextBytes = 256
)

// MFASecretBox encrypts TOTP seeds using a caller-provided, OFF-DATABASE
// 256-bit key. It never creates a default key, stores keys in SQLite or
// exposes a browser route. Provisioning and rotation require operator review.
type MFASecretBox struct {
	keyID string
	aead cipher.AEAD
}

// NewMFASecretBox accepts only a 32-byte key and an ASCII stable key ID.
// The caller must load the key from a reviewed out-of-database key source.
func NewMFASecretBox(keyID string, key []byte) (*MFASecretBox, error) {
	if len(key) != 32 || len(keyID) < 1 || len(keyID) > 40 {
		return nil, errors.New("invalid MFA encryption key")
	}
	for _, c := range keyID {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') &&
			!(c >= '0' && c <= '9') && c != '-' && c != '_' {
			return nil, errors.New("invalid MFA encryption key ID")
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &MFASecretBox{keyID: keyID, aead: aead}, nil
}

func validMFAIdentity(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

func mfaAssociatedData(identity, keyID string) []byte {
	// Explicit domain separation and length-prefix the identity to guard
	// against concatenation ambiguity and future key reuse.
	return []byte(fmt.Sprintf("TongPanel.TOTPSeed.v1:%d:%s:%s", len(identity), identity, keyID))
}

// Seal generates a fresh 96-bit nonce, binds ciphertext to identity and key
// version and returns a versioned, bounded text envelope suitable for SQLite.
// The caller remains responsible for redacting plaintext and ciphertext.
func (b *MFASecretBox) Seal(identity string, secret []byte) (string, error) {
	if b == nil || b.aead == nil || !validMFAIdentity(identity) ||
		len(secret) < mfaSecretMinBytes || len(secret) > mfaSecretMaxBytes {
		return "", errors.New("invalid TOTP secret or identity")
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	encrypted := b.aead.Seal(nil, nonce, secret, mfaAssociatedData(identity, b.keyID))
	payload := append(nonce, encrypted...)
	return "v1:" + b.keyID + ":" + base64.RawURLEncoding.EncodeToString(payload), nil
}

// Open authenticates the seed against both identity and key ID, rejecting
// cross-user swaps, tampering, truncated input and unknown key generations.
func (b *MFASecretBox) Open(identity, envelope string) ([]byte, error) {
	if b == nil || b.aead == nil || !validMFAIdentity(identity) ||
		len(envelope) > mfaMaxCiphertextBytes {
		return nil, errors.New("invalid encrypted TOTP secret")
	}
	parts := strings.Split(envelope, ":")
	if len(parts) != 3 || parts[0] != "v1" || parts[1] != b.keyID {
		return nil, errors.New("unsupported MFA secret key or envelope")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(raw) < b.aead.NonceSize()+b.aead.Overhead()+mfaSecretMinBytes ||
		len(raw) > b.aead.NonceSize()+b.aead.Overhead()+mfaSecretMaxBytes ||
		base64.RawURLEncoding.EncodeToString(raw) != parts[2] {
		return nil, errors.New("invalid encrypted TOTP payload")
	}
	secret, err := b.aead.Open(nil, raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():],
		mfaAssociatedData(identity, b.keyID))
	if err != nil || len(secret) < mfaSecretMinBytes || len(secret) > mfaSecretMaxBytes {
		return nil, errors.New("MFA secret authentication failed")
	}
	return secret, nil
}
