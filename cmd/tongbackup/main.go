// tongbackup is an experimental local operator utility. It is not wired to
// the panel API, scheduler, privileged agent, or production key management.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/parmtech/tong-panel/internal/backups"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "tongbackup:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: tongbackup encrypt|decrypt|rotate --tenant ID --output PATH --key-file PATH [--old-key-file PATH]")
	}
	operation := args[0]
	flags := flag.NewFlagSet("tongbackup "+operation, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	tenantID := flags.String("tenant", "", "tenant identifier to bind into the backup envelope")
	outputPath := flags.String("output", "", "new output path; existing paths are never overwritten")
	keyPath := flags.String("key-file", "", "externally managed 32-byte wrapping-key file")
	oldKeyPath := flags.String("old-key-file", "", "old wrapping-key file used by rotate")
	if err := flags.Parse(args[1:]); err != nil || len(flags.Args()) != 0 {
		return errors.New("invalid arguments; run tongbackup encrypt|decrypt|rotate with --tenant, --output and key-file flags")
	}
	if strings.TrimSpace(*tenantID) == "" || *outputPath == "" || *keyPath == "" {
		return errors.New("tenant, output and key-file are required")
	}
	newKey, err := readWrappingKey(*keyPath)
	if err != nil {
		return err
	}
	defer wipeKey(newKey)

	switch operation {
	case "encrypt":
		if *oldKeyPath != "" {
			return errors.New("old-key-file is only valid for rotate")
		}
		return backups.EncryptFile(*outputPath, stdin, *tenantID, newKey, backups.Options{})
	case "decrypt":
		if *oldKeyPath != "" {
			return errors.New("old-key-file is only valid for rotate")
		}
		return backups.DecryptFile(*outputPath, stdin, *tenantID, newKey, backups.Options{})
	case "rotate":
		if *oldKeyPath == "" {
			return errors.New("old-key-file is required for rotate")
		}
		oldKey, err := readWrappingKey(*oldKeyPath)
		if err != nil {
			return err
		}
		defer wipeKey(oldKey)
		return backups.RotateWrappingKeyFile(*outputPath, stdin, *tenantID, oldKey, newKey, backups.Options{})
	default:
		return errors.New("operation must be encrypt, decrypt, or rotate")
	}
}

func readWrappingKey(path string) ([]byte, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("wrapping-key file permission checks are supported only on Linux")
	}
	if path == "" {
		return nil, errors.New("wrapping-key file path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errors.New("cannot read wrapping-key file")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o400 == 0 || info.Mode().Perm()&0o177 != 0 {
		return nil, errors.New("wrapping-key file must be regular, owner-readable, and have no group, other, or execute permissions")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot read wrapping-key file")
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) || !openedInfo.Mode().IsRegular() {
		return nil, errors.New("wrapping-key file changed while opening")
	}
	key := make([]byte, 33)
	n, err := io.ReadFull(file, key)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		wipeKey(key)
		return nil, errors.New("cannot read wrapping-key file")
	}
	if n != 32 {
		wipeKey(key)
		return nil, errors.New("wrapping-key file must contain exactly 32 bytes")
	}
	return key[:32], nil
}

func wipeKey(key []byte) {
	for i := range key {
		key[i] = 0
	}
}
