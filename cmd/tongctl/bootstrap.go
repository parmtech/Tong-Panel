package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
)

// localOwnerBootstrap must remain an explicitly opted-in, interactive-operator
// invoked LOCAL command. No default account, password, API route or socket
// action may be introduced here. Password input comes ONLY from an anonymous
// pipe, never command-line arguments, environment variables or plaintext files.
func localOwnerBootstrap(args []string, passwordPipe *os.File, output io.Writer) error {
	if os.Getenv("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP") != "yes" {
		return errors.New("bootstrap disabled: explicit local operator opt-in required")
	}
	flags := flag.NewFlagSet("bootstrap-owner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var path, username, tenant string
	var passwordStdin bool
	flags.StringVar(&path, "db", "", "existing or fresh local SQLite metadata file")
	flags.StringVar(&username, "username", "", "initial owner account name")
	flags.StringVar(&tenant, "tenant", "", "initial tenant display name")
	flags.BoolVar(&passwordStdin, "password-stdin", false, "require password from a private pipe")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 0 ||
		path == "" || username == "" || tenant == "" || !passwordStdin {
		return errors.New("usage: tongctl bootstrap-owner --db PATH --username NAME --tenant NAME --password-stdin")
	}
	// A terminal could echo secret bytes and regular files leave unencrypted
	// credentials on disk. Require pipe semantics even in automation.
	if passwordPipe == nil {
		return errors.New("password must be supplied on an anonymous pipe")
	}
	info, err := passwordPipe.Stat()
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		return errors.New("password input must be a pipe, not a terminal or file")
	}
	// Bound input before allocating a password buffer; accept at most one
	// final LF delimiter, never embedded newline, carriage return or NUL.
	password, err := io.ReadAll(io.LimitReader(passwordPipe, 1026))
	if err != nil {
		return err
	}
	defer clear(password)
	if len(password) > 0 && password[len(password)-1] == '\n' {
		password = password[:len(password)-1]
	}
	if len(password) < 12 || len(password) > 1024 ||
		bytes.IndexAny(password, "\x00\r\n") >= 0 {
		return errors.New("invalid bootstrap password length or encoding")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(absPath)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	if err := requireSecureOwnedPath(parent, true); err != nil {
		return err
	}
	// Exclusive 0600 creation prevents a preexisting symlink from replacing
	// a new panel database. Existing paths must be regular, owned and private.
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		if err := file.Close(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrExist) {
		return err
	}
	if err := requireSecureOwnedPath(absPath, false); err != nil {
		return err
	}
	db, err := database.Open(context.Background(), absPath)
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := auth.BootstrapOwner(context.Background(), db, username, tenant, password)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "local owner created; tenant_id=%s identity_id=%s\n", result.TenantID, result.IdentityID)
	return err
}

func requireSecureOwnedPath(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if directory {
		if !info.IsDir() {
			return errors.New("database parent must be a directory")
		}
	} else if !info.Mode().IsRegular() {
		return errors.New("database file must be a regular file")
	}
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("database and parent must not be accessible by group or others")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("database and parent must be owned by the local operator")
	}
	return nil
}
