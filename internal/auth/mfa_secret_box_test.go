package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func newTestMFABox(t *testing.T) *MFASecretBox {
	t.Helper()
	key := bytes.Repeat([]byte{0x93}, 32)
	box, err := NewMFASecretBox("key_2026", key)
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestMFASecretBoxEncryptedRandomizedAndIdentityBound(t *testing.T) {
	box := newTestMFABox(t)
	secret := []byte("01234567890123456789")
	first, err := box.Seal("user-a", secret)
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.Seal("user-a", secret)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, string(secret)) {
		t.Fatal("TOTP seed not protected by fresh authenticated encryption")
	}
	opened, err := box.Open("user-a", first)
	if err != nil || !bytes.Equal(opened, secret) {
		t.Fatalf("identity-bound AEAD round trip failed: %v", err)
	}
	if _, err := box.Open("user-b", first); err == nil {
		t.Fatal("ciphertext could be swapped between identities")
	}
	other, _ := NewMFASecretBox("key_2026", bytes.Repeat([]byte{0x34}, 32))
	if _, err := other.Open("user-a", first); err == nil {
		t.Fatal("wrong key decrypted seed")
	}
	renamed, _ := NewMFASecretBox("key_2027", bytes.Repeat([]byte{0x93}, 32))
	if _, err := renamed.Open("user-a", first); err == nil {
		t.Fatal("changed key ID accepted without key migration")
	}
}

func TestMFASecretBoxTamperingInvalidFormatsAndBounds(t *testing.T) {
	box := newTestMFABox(t)
	secret := []byte("01234567890123456789")
	good, err := box.Seal("user-a", secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "v2:key_2026:x", "v1:key_2026:x",
		"v1:key_2026:", "v1:old:x", strings.Repeat("X", 400),
		good + "=", good + ":suffix"} {
		if _, err := box.Open("user-a", bad); err == nil {
			t.Fatalf("accepted invalid encrypted seed format: %q", bad)
		}
	}
	parts := strings.Split(good, ":")
	raw, _ := base64.RawURLEncoding.DecodeString(parts[2])
	raw[len(raw)-1] ^= 1
	tampered := "v1:" + parts[1] + ":" + base64.RawURLEncoding.EncodeToString(raw)
	if _, err := box.Open("user-a", tampered); err == nil {
		t.Fatal("AEAD tag modification was accepted")
	}
	raw[0] ^= 1
	tampered = "v1:" + parts[1] + ":" + base64.RawURLEncoding.EncodeToString(raw)
	if _, err := box.Open("user-a", tampered); err == nil {
		t.Fatal("AEAD nonce modification was accepted")
	}
	for _, tc := range []struct {
		identity string
		seed     []byte
	}{
		{"", secret}, {strings.Repeat("a", 129), secret},
		{"user\nb", secret}, {"user-a", []byte("short")},
		{"user-a", bytes.Repeat([]byte{8}, 65)},
	} {
		if _, err := box.Seal(tc.identity, tc.seed); err == nil {
			t.Fatalf("accepted invalid identity / TOTP seed: %q, %d", tc.identity, len(tc.seed))
		}
	}
	for _, id := range []string{"", "bad:name", "bad\nname", strings.Repeat("x", 41)} {
		if _, err := NewMFASecretBox(id, bytes.Repeat([]byte{9}, 32)); err == nil {
			t.Fatalf("invalid key generation accepted: %q", id)
		}
	}
	for _, key := range [][]byte{nil, bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{2}, 64)} {
		if _, err := NewMFASecretBox("valid", key); err == nil {
			t.Fatalf("accepted wrong AEAD key length %d", len(key))
		}
	}
	if _, err := box.Open("", good); err == nil {
		t.Fatal("missing identity accepted")
	}
}

func TestMFASecretBoxWorksWithIndependentCryptographicKey(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	box, err := NewMFASecretBox("operator_v1", key)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{20, 32, 64} {
		secret := make([]byte, size)
		if _, err := rand.Read(secret); err != nil {
			t.Fatal(err)
		}
		encrypted, err := box.Seal("account-2026", secret)
		if err != nil {
			t.Fatal(err)
		}
		opened, err := box.Open("account-2026", encrypted)
		if err != nil || !bytes.Equal(opened, secret) {
			t.Fatalf("round trip seed of %d bytes failed: %v", size, err)
		}
	}
}
