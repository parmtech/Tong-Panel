package backups

import (
	"bytes"
	"errors"
	"testing"
)

func TestVerifyAuthenticatesCompleteTenantBoundStream(t *testing.T) {
	key := bytes.Repeat([]byte{0x37}, 32)
	plaintext := bytes.Repeat([]byte("fixture-data"), 15000)
	var encrypted bytes.Buffer
	if err := Encrypt(&encrypted, bytes.NewReader(plaintext), "tenant-a", key, Options{}); err != nil {
		t.Fatal(err)
	}
	sealed := encrypted.Bytes()
	if err := Verify(bytes.NewReader(sealed), "tenant-a", key, Options{}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	tampered := bytes.Clone(sealed)
	tampered[len(tampered)/2] ^= 1
	finalTag := bytes.Clone(sealed)
	finalTag[len(finalTag)-1] ^= 1
	cases := []struct {
		name   string
		data   []byte
		tenant string
		key    []byte
	}{
		{"wrong tenant", sealed, "tenant-b", key}, {"wrong key", sealed, "tenant-a", bytes.Repeat([]byte{0x38}, 32)},
		{"middle corruption", tampered, "tenant-a", key}, {"final tag", finalTag, "tenant-a", key},
		{"truncated", sealed[:len(sealed)-1], "tenant-a", key}, {"trailing bytes", append(bytes.Clone(sealed), 1), "tenant-a", key},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Verify(bytes.NewReader(tc.data), tc.tenant, tc.key, Options{}); !errors.Is(err, ErrInvalidBackup) {
				t.Fatalf("verify: %v", err)
			}
		})
	}
	if err := Verify(bytes.NewReader(sealed), "tenant-a", key, Options{MaxPlaintextBytes: 1}); !errors.Is(err, ErrBackupLimit) {
		t.Fatalf("limit: %v", err)
	}
	if err := Verify(nil, "tenant-a", key, Options{}); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("nil stream: %v", err)
	}
}
