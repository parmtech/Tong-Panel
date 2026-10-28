package backups

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRotateWrappingKeyStreamsToNewKey(t *testing.T) {
	oldKey := bytes.Repeat([]byte{0x3c}, 32)
	newKey := bytes.Repeat([]byte{0xc3}, 32)
	plain := bytes.Repeat([]byte("rotated tenant backup"), ChunkSize)
	oldEnvelope := encryptWithKey(t, "tenant-a", oldKey, plain)
	var rotated bytes.Buffer
	if err := RotateWrappingKey(&rotated, bytes.NewReader(oldEnvelope), "tenant-a", oldKey, newKey, Options{}); err != nil {
		t.Fatalf("RotateWrappingKey() error = %v", err)
	}
	var restored bytes.Buffer
	if err := Decrypt(&restored, bytes.NewReader(rotated.Bytes()), "tenant-a", newKey, Options{}); err != nil {
		t.Fatalf("Decrypt() with new wrapping key error = %v", err)
	}
	if !bytes.Equal(restored.Bytes(), plain) {
		t.Fatal("rotation changed backup plaintext")
	}
	if err := Decrypt(&bytes.Buffer{}, bytes.NewReader(rotated.Bytes()), "tenant-a", oldKey, Options{}); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("old wrapping key accepted rotated backup: %v", err)
	}
}

func TestRotateWrappingKeyPropagatesInvalidEnvelopeWithoutPublishing(t *testing.T) {
	oldKey := bytes.Repeat([]byte{0x3c}, 32)
	newKey := bytes.Repeat([]byte{0xc3}, 32)
	envelope := encryptWithKey(t, "tenant-a", oldKey, bytes.Repeat([]byte("secret"), ChunkSize))
	envelope[len(envelope)-1] ^= 0x01
	var rotated bytes.Buffer
	err := RotateWrappingKey(&rotated, bytes.NewReader(envelope), "tenant-a", oldKey, newKey, Options{})
	if !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("RotateWrappingKey() error = %v, want ErrInvalidBackup", err)
	}
	if rotated.Len() > 0 {
		if err := Decrypt(io.Discard, bytes.NewReader(rotated.Bytes()), "tenant-a", newKey, Options{}); err == nil {
			t.Fatal("partial rotation output unexpectedly forms a valid envelope")
		}
	}
}

func TestRotateWrappingKeyRejectsInvalidNewKeyBeforeWriting(t *testing.T) {
	oldKey := bytes.Repeat([]byte{0x3c}, 32)
	var rotated bytes.Buffer
	err := RotateWrappingKey(&rotated, bytes.NewReader(nil), "tenant-a", oldKey, []byte("short"), Options{})
	if !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("RotateWrappingKey() error = %v, want ErrInvalidKey", err)
	}
	if rotated.Len() != 0 {
		t.Fatal("invalid new key wrote an envelope header")
	}
}

func TestRotateWrappingKeyFilePublishesOnlyAuthenticatedRotation(t *testing.T) {
	dir := t.TempDir()
	oldKey := bytes.Repeat([]byte{0x3c}, 32)
	newKey := bytes.Repeat([]byte{0xc3}, 32)
	plain := bytes.Repeat([]byte("file rotation payload"), 2000)
	oldEnvelope := encryptWithKey(t, "tenant-a", oldKey, plain)
	path := filepath.Join(dir, "rotated.backup")
	if err := RotateWrappingKeyFile(path, bytes.NewReader(oldEnvelope), "tenant-a", oldKey, newKey, Options{}); err != nil {
		t.Fatalf("RotateWrappingKeyFile() error = %v", err)
	}
	rotated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var restored bytes.Buffer
	if err := Decrypt(&restored, bytes.NewReader(rotated), "tenant-a", newKey, Options{}); err != nil {
		t.Fatalf("rotated file did not authenticate with new key: %v", err)
	}
	if !bytes.Equal(restored.Bytes(), plain) {
		t.Fatal("file rotation changed plaintext")
	}
	assertNoStagedFiles(t, dir)
}

func TestRotateWrappingKeyFileWithTamperingPublishesNothing(t *testing.T) {
	dir := t.TempDir()
	oldKey := bytes.Repeat([]byte{0x3c}, 32)
	newKey := bytes.Repeat([]byte{0xc3}, 32)
	bad := encryptWithKey(t, "tenant-a", oldKey, bytes.Repeat([]byte("private"), ChunkSize))
	bad[len(bad)-1] ^= 1
	path := filepath.Join(dir, "should-not-exist.backup")
	err := RotateWrappingKeyFile(path, bytes.NewReader(bad), "tenant-a", oldKey, newKey, Options{})
	if !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("RotateWrappingKeyFile() error = %v, want ErrInvalidBackup", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("tampered rotation destination exists (Lstat error = %v)", err)
	}
	assertNoStagedFiles(t, dir)
}

func encryptWithKey(t *testing.T, tenant string, key, plain []byte) []byte {
	t.Helper()
	var envelope bytes.Buffer
	if err := Encrypt(&envelope, bytes.NewReader(plain), tenant, key, Options{}); err != nil {
		t.Fatal(err)
	}
	return envelope.Bytes()
}
