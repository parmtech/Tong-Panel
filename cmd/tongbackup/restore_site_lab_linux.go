//go:build linux

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/parmtech/tong-panel/internal/files"
	"golang.org/x/sys/unix"
)

var (
	errLabArguments          = errors.New("restore-site-lab requires exactly one --tenant, --key-file, --root, --destination and --disposable-only")
	errLabBoundary           = errors.New("restore-site-lab requires non-root disposable-lab opt-in and a fresh private temporary root")
	errLabOutcomeUnknown     = errors.New("restore publication outcome unknown; do not retry or clean up without reconciliation")
	errLabPublishedUncertain = errors.New("restore published but durability is uncertain; reconcile before retry")
	errLabRootOutcomeUnknown = errors.New("lab root creation outcome unknown; inspect before retry")
	errLabRootCreatedFailure = errors.New("lab root was created but could not be opened; inspect before retry")
)

type restoreSiteLabArgs struct{ tenant, key, root, destination string }

// The command must not print underlying filesystem errors, which can include
// archive member names. Preserve the error chain for programmatic inspection.
type redactedLabError struct {
	message string
	causes  []error
}

func (e *redactedLabError) Error() string   { return e.message }
func (e *redactedLabError) Unwrap() []error { return e.causes }

func parseRestoreSiteLabArgs(args []string) (restoreSiteLabArgs, error) {
	values := map[string]string{}
	ack := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return restoreSiteLabArgs{}, errLabArguments
		}
		name, value, equals := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if name == "disposable-only" {
			if ack || equals {
				return restoreSiteLabArgs{}, errLabArguments
			}
			ack = true
			continue
		}
		switch name {
		case "tenant", "key-file", "root", "destination":
		default:
			return restoreSiteLabArgs{}, errLabArguments
		}
		if _, ok := values[name]; ok {
			return restoreSiteLabArgs{}, errLabArguments
		}
		if !equals {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return restoreSiteLabArgs{}, errLabArguments
			}
			value = args[i]
		}
		if value == "" || len(value) > 4096 || strings.ContainsRune(value, 0) {
			return restoreSiteLabArgs{}, errLabArguments
		}
		values[name] = value
	}
	if !ack || len(values) != 4 {
		return restoreSiteLabArgs{}, errLabArguments
	}
	tenant := values["tenant"]
	if len(tenant) > 128 {
		return restoreSiteLabArgs{}, errLabArguments
	}
	for _, c := range []byte(tenant) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return restoreSiteLabArgs{}, errLabArguments
		}
	}
	name := values["destination"]
	if len(name) > 63 {
		return restoreSiteLabArgs{}, errLabArguments
	}
	for _, c := range []byte(name) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return restoreSiteLabArgs{}, errLabArguments
		}
	}
	for _, key := range []string{"root", "key-file"} {
		if !filepath.IsAbs(values[key]) || filepath.Clean(values[key]) != values[key] {
			return restoreSiteLabArgs{}, errLabArguments
		}
	}
	return restoreSiteLabArgs{tenant: tenant, key: values["key-file"], root: values["root"], destination: name}, nil
}

// runRestoreSiteLab is a disposable non-root fixture, not a production restore.
// Path separation is checked canonically but cannot establish absence of bind
// mount aliases or prevent a trusted ancestor being replaced concurrently.
func runRestoreSiteLab(args []string, src io.Reader) error {
	config, err := parseRestoreSiteLabArgs(args)
	if err != nil {
		return err
	}
	if os.Geteuid() == 0 || os.Getenv("TONG_DISPOSABLE_SITE_RESTORE") != "1" || src == nil {
		return errLabBoundary
	}
	base := filepath.Base(config.root)
	if !strings.HasPrefix(base, "tong-restore-lab-") || len(base) > 80 || base == "tong-restore-lab-" {
		return errLabBoundary
	}
	for _, c := range []byte(strings.TrimPrefix(base, "tong-restore-lab-")) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errLabBoundary
		}
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(config.root))
	if err != nil {
		return errLabBoundary
	}
	tmpCanonical, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		return errLabBoundary
	}
	rel, err := filepath.Rel(tmpCanonical, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errLabBoundary
	}
	// Parent paths and mount topology must remain trusted/stable throughout.
	// A path underneath /tmp is only a boundary, not proof of disposability.
	parentFD, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return errLabBoundary
	}
	defer unix.Close(parentFD)
	var st unix.Stat_t
	if err := unix.Fstat(parentFD, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(unix.Geteuid()) || st.Mode&0077 != 0 || st.Mode&0700 != 0700 {
		return errLabBoundary
	}
	keyCanonical, err := filepath.EvalSymlinks(config.key)
	if err != nil {
		return errors.New("wrapping-key file unavailable")
	}
	rootCanonical := filepath.Join(parent, base)
	keyRel, err := filepath.Rel(rootCanonical, keyCanonical)
	if err != nil || keyRel == "." || (keyRel != ".." && !strings.HasPrefix(keyRel, ".."+string(filepath.Separator))) {
		return errors.New("wrapping-key file must be outside the lab root")
	}
	key, err := readWrappingKey(config.key)
	if err != nil {
		return err
	}
	defer wipeKey(key)
	// This command creates a fresh root atomically and never removes it. Any
	// failed or uncertain outcome remains available for supervised inspection.
	var existing unix.Stat_t
	if err := unix.Fstatat(parentFD, base, &existing, unix.AT_SYMLINK_NOFOLLOW); err == nil {
		return errLabBoundary
	} else if !errors.Is(err, unix.ENOENT) {
		return errLabBoundary
	}
	if err := unix.Mkdirat(parentFD, base, 0700); err != nil {
		return errLabRootOutcomeUnknown
	}
	root, err := files.OpenRoot(rootCanonical)
	if err != nil {
		return errLabRootCreatedFailure
	}
	defer root.Close()
	roots, err := files.NewTenantRoots(map[string]*files.Root{config.tenant: root})
	if err != nil {
		return errLabRootCreatedFailure
	}
	result, err := roots.RestoreEncryptedGzipTarNewForTenantWithResult(context.Background(), config.tenant, config.destination, src, key, files.ArchiveLimits{})
	return restoreSiteLabOutcome(result, err)
}

func restoreSiteLabOutcome(result files.ArchivePublishResult, err error) error {
	switch result.State {
	case files.ArchiveNotPublished:
		if err == nil {
			return errors.New("restore returned an inconsistent outcome")
		}
		return &redactedLabError{message: "restore destination not published; private staging debris may remain", causes: []error{err}}
	case files.ArchivePublished:
		if err != nil {
			return &redactedLabError{message: errLabPublishedUncertain.Error(), causes: []error{errLabPublishedUncertain, err}}
		}
		return nil
	case files.ArchivePublicationUnknown:
		if err == nil {
			return errLabOutcomeUnknown
		}
		return &redactedLabError{message: errLabOutcomeUnknown.Error(), causes: []error{errLabOutcomeUnknown, err}}
	default:
		return errLabOutcomeUnknown
	}
}
