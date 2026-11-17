//go:build linux

package sites

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/parmtech/tong-panel/internal/files"
)

const (
	wordpressReleaseVersion = "7.1.4"
	wordpressReleaseURL     = "https://downloads.wordpress.org/release/wordpress-7.1.4.tar.gz"
	wordpressReleaseSHA256  = "881e53a4681ff2cd911651b11cccdede5ae477ddf0765af5d6a1a8b219a1d835"
	wordpressArchiveLimit   = files.MaxVerifiedArchiveCompressedBytes
)

// The SHA-256 pin was calculated from the official 7.1.4 tarball after its
// SHA-1 matched the checksum published by WordPress.org:
// https://wordpress.org/wordpress-7.1.4.tar.gz.sha1
// That archive has 4,250 tar entries, so the verified-artifact extractor uses
// an 8,192-entry bound while ordinary user archives remain capped at 2,048.

var ErrInvalidWordPressRelease = errors.New("invalid or unauthenticated WordPress release archive")

type wordPressRelease struct {
	url    string
	sha256 string
}

// InstallPinnedWordPressCore downloads the exact upstream release pinned in
// this source tree and atomically extracts it below an already-pinned tenant
// root. The relative destination must not exist. This helper never configures
// PHP, a database, credentials, a web server, or host services.
func InstallPinnedWordPressCore(ctx context.Context, root *files.Root, relative string) error {
	transport := http.DefaultTransport
	return installWordPressCore(ctx, root, relative, wordPressRelease{
		url: wordpressReleaseURL, sha256: wordpressReleaseSHA256,
	}, transport)
}

func installWordPressCore(ctx context.Context, root *files.Root, relative string, release wordPressRelease, transport http.RoundTripper) error {
	if ctx == nil || root == nil || transport == nil || relative == "" {
		return ErrInvalidWordPressRelease
	}
	parsed, err := url.Parse(release.url)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "downloads.wordpress.org" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		!strings.HasPrefix(parsed.Path, "/release/wordpress-") || !strings.HasSuffix(parsed.Path, ".tar.gz") {
		return ErrInvalidWordPressRelease
	}
	version := strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/release/wordpress-"), ".tar.gz")
	if version == "" || version[0] == '.' || version[len(version)-1] == '.' || strings.Contains(version, "..") {
		return ErrInvalidWordPressRelease
	}
	for _, character := range version {
		if (character < '0' || character > '9') && character != '.' {
			return ErrInvalidWordPressRelease
		}
	}
	digest, err := hex.DecodeString(release.sha256)
	if err != nil || len(digest) != sha256.Size {
		return ErrInvalidWordPressRelease
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ErrInvalidWordPressRelease
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download WordPress release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Request == nil || response.Request.URL == nil ||
		response.Request.URL.String() != parsed.String() ||
		(response.ContentLength >= 0 && response.ContentLength > wordpressArchiveLimit) {
		return ErrInvalidWordPressRelease
	}
	archive, err := io.ReadAll(io.LimitReader(response.Body, wordpressArchiveLimit+1))
	if err != nil {
		return fmt.Errorf("read WordPress release: %w", err)
	}
	if int64(len(archive)) > wordpressArchiveLimit {
		return files.ErrArchiveLimit
	}
	actual := sha256.Sum256(archive)
	if !bytes.Equal(actual[:], digest) {
		return ErrInvalidWordPressRelease
	}
	if err := validateWordPressArchive(bytes.NewReader(archive)); err != nil {
		return err
	}
	return root.ExtractVerifiedGzipTarNew(relative, archive, [sha256.Size]byte(digest))
}

// validateWordPressArchive enforces the canonical single top-level directory
// and rejects links/special files before the tenant-root extractor publishes
// anything. The extractor independently enforces path confinement, size and
// entry-count limits, and only permits regular files/directories.
func validateWordPressArchive(source io.Reader) error {
	gz, err := gzip.NewReader(source)
	if err != nil {
		return ErrInvalidWordPressRelease
	}
	defer gz.Close()
	bounded := &io.LimitedReader{R: gz, N: files.MaxVerifiedArchiveExpandedBytes + 1}
	reader := tar.NewReader(bounded)
	entries := 0
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			if _, err := io.Copy(io.Discard, bounded); err != nil {
				return ErrInvalidWordPressRelease
			}
			if bounded.N == 0 {
				return files.ErrArchiveLimit
			}
			return nil
		}
		if err != nil {
			if bounded.N == 0 {
				return files.ErrArchiveLimit
			}
			return ErrInvalidWordPressRelease
		}
		entries++
		if entries > files.MaxVerifiedArchiveEntries ||
			!(header.Typeflag == tar.TypeDir || header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA) {
			return ErrInvalidWordPressRelease
		}
		name := strings.TrimSuffix(header.Name, "/")
		if name != "wordpress" && !strings.HasPrefix(name, "wordpress/") {
			return ErrInvalidWordPressRelease
		}
		if strings.Contains(header.Name, "\\") || header.Size < 0 || header.Size > files.MaxVerifiedArchiveFileBytes ||
			(header.Typeflag == tar.TypeDir && header.Size != 0) {
			return ErrInvalidWordPressRelease
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return ErrInvalidWordPressRelease
		}
	}
}
