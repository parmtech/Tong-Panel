//go:build linux

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

var errLabInventory = errors.New("retained-stage inventory unavailable; do not infer publication or cleanup safety")

func parseInspectRestoreLabArgs(args []string) (string, error) {
	var root string
	ack := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return "", errLabArguments
		}
		name, value, equals := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if name == "disposable-only" {
			if ack || equals {
				return "", errLabArguments
			}
			ack = true
			continue
		}
		if name != "root" || root != "" {
			return "", errLabArguments
		}
		if !equals {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return "", errLabArguments
			}
			value = args[i]
		}
		if value == "" || len(value) > 4096 || strings.ContainsRune(value, 0) {
			return "", errLabArguments
		}
		root = value
	}
	if !ack || root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", errLabArguments
	}
	return root, nil
}

// runInspectRestoreLab reads metadata only. It neither detects actual
// disposability nor determines which stages may be deleted/reconciled.
func runInspectRestoreLab(args []string, output io.Writer) error {
	rootPath, err := parseInspectRestoreLabArgs(args)
	if err != nil {
		return err
	}
	if os.Geteuid() == 0 || os.Getenv("TONG_DISPOSABLE_SITE_RESTORE") != "1" || output == nil {
		return errLabBoundary
	}
	base := filepath.Base(rootPath)
	if !strings.HasPrefix(base, "tong-restore-lab-") || len(base) > 80 || base == "tong-restore-lab-" {
		return errLabBoundary
	}
	for _, c := range []byte(strings.TrimPrefix(base, "tong-restore-lab-")) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errLabBoundary
		}
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(rootPath))
	if err != nil {
		return errLabBoundary
	}
	tmp, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		return errLabBoundary
	}
	rel, err := filepath.Rel(tmp, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errLabBoundary
	}
	parentFD, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return errLabBoundary
	}
	defer unix.Close(parentFD)
	var parentStat unix.Stat_t
	if err := unix.Fstat(parentFD, &parentStat); err != nil || parentStat.Mode&unix.S_IFMT != unix.S_IFDIR || parentStat.Uid != uint32(unix.Geteuid()) || parentStat.Mode&0077 != 0 || parentStat.Mode&0700 != 0700 {
		return errLabBoundary
	}
	var before unix.Stat_t
	if err := unix.Fstatat(parentFD, base, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil || !privateLabDirectory(before) {
		return errLabBoundary
	}
	// The trusted parent must remain stable; OpenRoot pins the selected root but
	// does not independently authorize its ownership or resolve mount aliases.
	root, err := files.OpenRoot(filepath.Join(parent, base))
	if err != nil {
		return errLabBoundary
	}
	defer root.Close()
	var current unix.Stat_t
	if err := unix.Fstatat(parentFD, base, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || current.Dev != before.Dev || current.Ino != before.Ino || !privateLabDirectory(current) {
		return errLabBoundary
	}
	count, err := root.CountRetainedArchiveStages(context.Background())
	if err != nil {
		return &redactedLabError{message: errLabInventory.Error(), causes: []error{errLabInventory, err}}
	}
	// Recheck the visible name after the pinned scan; this is detection only and
	// cannot replace the trusted/stable-parent precondition.
	if err := unix.Fstatat(parentFD, base, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil || current.Dev != before.Dev || current.Ino != before.Ino || !privateLabDirectory(current) {
		return errLabBoundary
	}
	text := "Archive stages observed: " + strconv.Itoa(count) + "\nSnapshot includes active stages; count does not establish publication or permit deletion or quota release.\n"
	n, err := io.WriteString(output, text)
	if err != nil {
		return &redactedLabError{message: "could not write inventory result", causes: []error{err}}
	}
	if n != len(text) {
		return io.ErrShortWrite
	}
	return nil
}
func privateLabDirectory(st unix.Stat_t) bool {
	return st.Mode&unix.S_IFMT == unix.S_IFDIR && st.Uid == uint32(unix.Geteuid()) && st.Mode&0077 == 0 && st.Mode&0700 == 0700
}
