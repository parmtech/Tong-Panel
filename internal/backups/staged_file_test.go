package backups

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEncryptFileAndDecryptFilePublishCompleteArtifacts(t *testing.T) {
	dir := t.TempDir()
	plain := bytes.Repeat([]byte("tenant backup payload"), 5000)
	cipherPath := filepath.Join(dir, "tenant-a.tpbk")
	plainPath := filepath.Join(dir, "restored.sql")
	if err := EncryptFile(cipherPath, bytes.NewReader(plain), "tenant-a", testWrappingKey, Options{}); err != nil {
		t.Fatalf("EncryptFile() error = %v", err)
	}
	sealed, err := os.ReadFile(cipherPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, plain) || bytes.Contains(sealed, []byte("tenant-a")) {
		t.Fatal("published envelope contains plaintext or tenant ID")
	}
	if err := DecryptFile(plainPath, bytes.NewReader(sealed), "tenant-a", testWrappingKey, Options{}); err != nil {
		t.Fatalf("DecryptFile() error = %v", err)
	}
	got, err := os.ReadFile(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("published restore does not match input")
	}
	assertNoStagedFiles(t, dir)
}

func TestDecryptFileDoesNotPublishUnauthenticatedPlaintext(t *testing.T) {
	dir := t.TempDir()
	plain := bytes.Repeat([]byte("sensitive-row,"), 9000)
	sealed := encryptBytes(t, "tenant-a", plain, Options{})
	sealed[len(sealed)-1] ^= 1
	destination := filepath.Join(dir, "restore.sql")
	err := DecryptFile(destination, bytes.NewReader(sealed), "tenant-a", testWrappingKey, Options{})
	if !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("DecryptFile() error = %v, want ErrInvalidBackup", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("failed restore destination exists (Lstat error = %v)", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestStagedFilesRefuseOverwriteAndLeaveExistingContents(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "existing.backup")
	if err := os.WriteFile(destination, []byte("preserve me"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := EncryptFile(destination, strings.NewReader("replacement"), "tenant-a", testWrappingKey, Options{})
	if !os.IsExist(err) {
		t.Fatalf("EncryptFile() error = %v, want existing destination denial", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "preserve me" {
		t.Fatalf("existing destination changed to %q", got)
	}
	assertNoStagedFiles(t, dir)
}

func TestStagedFilesCleanTemporaryOutputOnSourceFailure(t *testing.T) {
	dir := t.TempDir()
	wantErr := errors.New("injected source read failure")
	reader := failingReader{err: wantErr}
	err := EncryptFile(filepath.Join(dir, "failed.backup"), reader, "tenant-a", testWrappingKey, Options{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("EncryptFile() error = %v, want injected failure", err)
	}
	assertNoStagedFiles(t, dir)
}

func TestConcurrentStagedWritersPublishExactlyOneCompleteFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "winner.backup")
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		plain := bytes.Repeat([]byte{byte('A' + i)}, ChunkSize+i)
		wg.Add(1)
		go func(payload []byte) {
			defer wg.Done()
			<-start
			errs <- EncryptFile(destination, bytes.NewReader(payload), "tenant-race", testWrappingKey, Options{})
		}(plain)
	}
	close(start)
	wg.Wait()
	close(errs)
	winners := 0
	denied := 0
	for err := range errs {
		if err == nil {
			winners++
		} else if os.IsExist(err) {
			denied++
		} else {
			t.Fatalf("concurrent EncryptFile() error = %v", err)
		}
	}
	if winners != 1 || denied != 1 {
		t.Fatalf("published %d winners and denied %d writers; want one each", winners, denied)
	}
	sealed, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	var restored bytes.Buffer
	if err := Decrypt(&restored, bytes.NewReader(sealed), "tenant-race", testWrappingKey, Options{}); err != nil {
		t.Fatalf("published winner is not a complete envelope: %v", err)
	}
	if !bytes.Equal(restored.Bytes(), bytes.Repeat([]byte{'A'}, ChunkSize)) &&
		!bytes.Equal(restored.Bytes(), bytes.Repeat([]byte{'B'}, ChunkSize+1)) {
		t.Fatal("destination contains a partial or unexpected concurrent writer")
	}
	assertNoStagedFiles(t, dir)
}

func TestStagedFileRejectsDirectoryDestination(t *testing.T) {
	dir := t.TempDir()
	if err := EncryptFile(dir, bytes.NewReader(nil), "tenant-a", testWrappingKey, Options{}); err == nil {
		t.Fatal("EncryptFile() accepted a directory destination")
	}
}

func assertNoStagedFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), strings.TrimSuffix(stagedFilePattern, "*")) {
			t.Errorf("temporary staged file remains: %s", entry.Name())
		}
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

var _ io.Reader = failingReader{}
