package backups

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"sync"
	"testing"
)

var testWrappingKey = bytes.Repeat([]byte{0x5a}, 32)

func encryptBytes(t *testing.T, tenant string, plaintext []byte, options Options) []byte {
	t.Helper()
	var encrypted bytes.Buffer
	if err := Encrypt(&encrypted, bytes.NewReader(plaintext), tenant, testWrappingKey, options); err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	return encrypted.Bytes()
}

func TestEnvelopeRoundTripChunkBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, ChunkSize - 1, ChunkSize, ChunkSize + 1, 3*ChunkSize + 17} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			plaintext := bytes.Repeat([]byte("p"), size)
			sealed := encryptBytes(t, "tenant-a", plaintext, Options{})
			if bytes.Contains(sealed, []byte("tenant-a")) || bytes.Contains(sealed, plaintext) && size > 32 {
				t.Fatal("envelope exposed tenant metadata or plaintext")
			}
			var restored bytes.Buffer
			if err := Decrypt(&restored, bytes.NewReader(sealed), "tenant-a", testWrappingKey, Options{}); err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			if !bytes.Equal(restored.Bytes(), plaintext) {
				t.Fatalf("restored %d bytes; want %d", restored.Len(), size)
			}
		})
	}
}

func TestEnvelopeBindsTenantAndWrappingKey(t *testing.T) {
	sealed := encryptBytes(t, "tenant-a", []byte("customer backup contents"), Options{})
	for name, test := range map[string]struct {
		name string
		key  []byte
	}{
		"wrong tenant": {name: "tenant-b", key: testWrappingKey},
		"wrong key":    {name: "tenant-a", key: bytes.Repeat([]byte{0x7b}, 32)},
	} {
		t.Run(name, func(t *testing.T) {
			var restored bytes.Buffer
			if err := Decrypt(&restored, bytes.NewReader(sealed), test.name, test.key, Options{}); !errors.Is(err, ErrInvalidBackup) {
				t.Fatalf("Decrypt() error = %v, want ErrInvalidBackup", err)
			}
			if restored.Len() != 0 {
				t.Fatalf("wrong tenant/key wrote %d plaintext bytes", restored.Len())
			}
		})
	}
}

func TestEnvelopeRejectsTamperingTruncationAndTrailingBytes(t *testing.T) {
	sealed := encryptBytes(t, "tenant-a", bytes.Repeat([]byte("secret-data-"), 9000), Options{})
	firstChunk := headerSize + 4 + 7
	finalTag := len(sealed) - 1
	cases := map[string]func([]byte){
		"format header": func(v []byte) { v[0] ^= 0x80 },
		"wrapped data key": func(v []byte) { v[24] ^= 0x01 },
		"chunk ciphertext": func(v []byte) { v[firstChunk] ^= 0x01 },
		"final authenticator": func(v []byte) { v[finalTag] ^= 0x01 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			copyOfEnvelope := append([]byte(nil), sealed...)
			mutate(copyOfEnvelope)
			var restored bytes.Buffer
			if err := Decrypt(&restored, bytes.NewReader(copyOfEnvelope), "tenant-a", testWrappingKey, Options{}); !errors.Is(err, ErrInvalidBackup) {
				t.Fatalf("Decrypt() error = %v, want ErrInvalidBackup", err)
			}
		})
	}
	if err := Decrypt(io.Discard, bytes.NewReader(sealed[:len(sealed)-1]), "tenant-a", testWrappingKey, Options{}); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("truncated envelope error = %v, want ErrInvalidBackup", err)
	}
	withTrailing := append(append([]byte(nil), sealed...), 0x00)
	if err := Decrypt(io.Discard, bytes.NewReader(withTrailing), "tenant-a", testWrappingKey, Options{}); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("trailing data error = %v, want ErrInvalidBackup", err)
	}
}

func TestEnvelopeEnforcesConfiguredAndHardLimits(t *testing.T) {
	if err := Encrypt(io.Discard, bytes.NewReader([]byte("123456")), "tenant-a", testWrappingKey, Options{MaxPlaintextBytes: 5}); !errors.Is(err, ErrBackupLimit) {
		t.Fatalf("Encrypt() error = %v, want ErrBackupLimit", err)
	}
	sealed := encryptBytes(t, "tenant-a", []byte("123456"), Options{})
	if err := Decrypt(io.Discard, bytes.NewReader(sealed), "tenant-a", testWrappingKey, Options{MaxPlaintextBytes: 5}); !errors.Is(err, ErrBackupLimit) {
		t.Fatalf("Decrypt() error = %v, want ErrBackupLimit", err)
	}
	for _, limit := range []int64{-1, MaxPlaintextBytes + 1} {
		if err := Encrypt(io.Discard, bytes.NewReader(nil), "tenant-a", testWrappingKey, Options{MaxPlaintextBytes: limit}); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("limit %d error = %v, want ErrInvalidOptions", limit, err)
		}
	}
}

func TestEnvelopeRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		tenant string
		key    []byte
		want   error
	}{
		{name: "empty tenant", tenant: "", key: testWrappingKey, want: ErrInvalidTenant},
		{name: "tenant control byte", tenant: "tenant\nA", key: testWrappingKey, want: ErrInvalidTenant},
		{name: "short key", tenant: "tenant-a", key: []byte("short"), want: ErrInvalidKey},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := Encrypt(io.Discard, bytes.NewReader(nil), tc.tenant, tc.key, Options{}); !errors.Is(err, tc.want) {
				t.Fatalf("Encrypt() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestConcurrentEnvelopeOperationsUseIndependentKeysAndNonces(t *testing.T) {
	const workers = 12
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			plain := bytes.Repeat([]byte{byte(i + 1)}, ChunkSize+i)
			var sealed bytes.Buffer
			if err := Encrypt(&sealed, bytes.NewReader(plain), "tenant-race", testWrappingKey, Options{}); err != nil {
				errs <- err
				return
			}
			var restored bytes.Buffer
			if err := Decrypt(&restored, bytes.NewReader(sealed.Bytes()), "tenant-race", testWrappingKey, Options{}); err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(restored.Bytes(), plain) {
				errs <- errors.New("concurrent round trip changed backup contents")
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

