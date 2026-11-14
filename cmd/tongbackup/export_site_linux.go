//go:build linux

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

var (
	errExportArguments          = errors.New("export-site requires exactly one --tenant, --key-file, --root, --site, --output and --source-quiesced")
	errExportPublishUnknown     = errors.New("ciphertext publication outcome unknown; private stage retained for reconciliation")
	errExportPublishedUncertain = errors.New("ciphertext published but cleanup or durability is uncertain; reconcile before retry")
	errExportCleanupUncertain   = errors.New("export failed and private staging cleanup is uncertain; reconcile before retry")
)

type exportSiteArgs struct{ tenant, key, root, site, output string }

// Parsing is deliberately separate from I/O; duplicate, missing, unrecognized,
// positional or valued acknowledgement flags are rejected before key reads.
func parseExportSiteArgs(args []string) (exportSiteArgs, error) {
	values := map[string]string{}
	ack := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return exportSiteArgs{}, errExportArguments
		}
		name, value, hasEquals := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if name == "source-quiesced" {
			if ack || hasEquals {
				return exportSiteArgs{}, errExportArguments
			}
			ack = true
			continue
		}
		switch name {
		case "tenant", "key-file", "root", "site", "output":
		default:
			return exportSiteArgs{}, errExportArguments
		}
		if _, exists := values[name]; exists {
			return exportSiteArgs{}, errExportArguments
		}
		if !hasEquals {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return exportSiteArgs{}, errExportArguments
			}
			value = args[i]
		}
		if value == "" || len(value) > 4096 || strings.ContainsRune(value, 0) {
			return exportSiteArgs{}, errExportArguments
		}
		values[name] = value
	}
	if !ack || len(values) != 5 {
		return exportSiteArgs{}, errExportArguments
	}
	if len(values["tenant"]) > 128 {
		return exportSiteArgs{}, errExportArguments
	}
	for _, c := range []byte(values["tenant"]) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return exportSiteArgs{}, errExportArguments
		}
	}
	site := values["site"]
	if path.IsAbs(site) || path.Clean(site) != site || strings.Contains(site, "\\") {
		return exportSiteArgs{}, errExportArguments
	}
	for _, part := range strings.Split(site, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".tong-stage-") || strings.HasPrefix(part, ".tong-archive-stage-") || strings.HasPrefix(part, ".tong-ciphertext-stage-") {
			return exportSiteArgs{}, errExportArguments
		}
	}
	for _, name := range []string{"root", "output", "key-file"} {
		if !filepath.IsAbs(values[name]) || filepath.Clean(values[name]) != values[name] {
			return exportSiteArgs{}, errExportArguments
		}
	}
	return exportSiteArgs{tenant: values["tenant"], key: values["key-file"], root: values["root"], site: values["site"], output: values["output"]}, nil
}

func runExportSite(args []string) error {
	config, err := parseExportSiteArgs(args)
	if err != nil {
		return err
	}
	rootInfo, err := os.Lstat(config.root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("source root unavailable")
	}
	// Both directory trees must have trusted, stable parent paths. Resolving here
	// is not protection against an untrusted actor renaming those parent paths.
	rootPath, err := filepath.EvalSymlinks(config.root)
	if err != nil {
		return errors.New("source root unavailable")
	}
	outputDir, err := filepath.EvalSymlinks(filepath.Dir(config.output))
	if err != nil {
		return errors.New("private output directory unavailable")
	}
	relative, err := filepath.Rel(rootPath, outputDir)
	if err != nil || relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("output directory must be outside the entire source root")
	}
	keyPath, err := filepath.EvalSymlinks(config.key)
	if err != nil {
		return errors.New("wrapping-key file unavailable")
	}
	keyRelative, err := filepath.Rel(rootPath, keyPath)
	if err != nil || keyRelative == "." || (keyRelative != ".." && !strings.HasPrefix(keyRelative, ".."+string(filepath.Separator))) {
		return errors.New("wrapping-key file must be outside the entire source root")
	}
	root, err := files.OpenRoot(config.root)
	if err != nil {
		return errors.New("source root unavailable")
	}
	defer root.Close()
	roots, err := files.NewTenantRoots(map[string]*files.Root{config.tenant: root})
	if err != nil {
		return errors.New("invalid tenant root")
	}
	key, err := readWrappingKey(config.key)
	if err != nil {
		return err
	}
	defer wipeKey(key)
	return publishPrivateCiphertext(outputDir, filepath.Base(config.output), func(dst io.Writer) error {
		return roots.ExportEncryptedGzipTarForTenant(context.Background(), config.tenant, config.site, dst, key, files.ArchiveLimits{})
	}, unix.Linkat, unix.Fsync)
}

// publishPrivateCiphertext pins a private, owner-controlled output directory;
// rename-like link errors retain the private stage because the destination may
// already have been published. It never deletes an ambiguous destination.
func publishPrivateCiphertext(directory, base string, produce func(io.Writer) error, link func(int, string, int, string, int) error, syncDir func(int) error) (retErr error) {
	if base == "" || base == "." || base == ".." || strings.ContainsAny(base, "/\x00") || strings.HasPrefix(base, ".tong-ciphertext-stage-") || produce == nil || link == nil || syncDir == nil {
		return errExportArguments
	}
	fd, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return errors.New("private output directory unavailable")
	}
	defer unix.Close(fd)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFDIR || stat.Uid != uint32(unix.Geteuid()) || stat.Mode&0077 != 0 || stat.Mode&0700 != 0700 {
		return errors.New("output directory must be owner-controlled and private")
	}
	var existing unix.Stat_t
	if err := unix.Fstatat(fd, base, &existing, unix.AT_SYMLINK_NOFOLLOW); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, unix.ENOENT) {
		return errors.New("cannot inspect output destination")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("cannot prepare private output")
	}
	stage := ".tong-ciphertext-stage-" + hex.EncodeToString(nonce[:])
	raw, err := unix.Openat(fd, stage, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return errors.New("cannot prepare private output")
	}
	file := os.NewFile(uintptr(raw), "private-ciphertext-stage")
	if file == nil {
		_ = unix.Close(raw)
		_ = unix.Unlinkat(fd, stage, 0)
		return errors.New("cannot prepare private output")
	}
	keep := false
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !keep {
			if cleanupErr := unix.Unlinkat(fd, stage, 0); cleanupErr != nil && !errors.Is(cleanupErr, unix.ENOENT) {
				retErr = fmt.Errorf("%w: %w", errExportCleanupUncertain, retErr)
			}
		}
	}()
	if err := produce(file); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return errors.New("cannot sync private ciphertext")
	}
	if err := file.Close(); err != nil {
		closed = true
		return errors.New("cannot close private ciphertext")
	}
	closed = true
	// Regardless of errno, a failed link may already have taken effect on a
	// remote filesystem. Keep the stage; reconciliation must inspect both names.
	if err := link(fd, stage, fd, base, 0); err != nil {
		keep = true
		return errExportPublishUnknown
	}
	keep = true
	if err := unix.Unlinkat(fd, stage, 0); err != nil {
		return errExportPublishedUncertain
	}
	if err := syncDir(fd); err != nil {
		return errExportPublishedUncertain
	}
	return nil
}
