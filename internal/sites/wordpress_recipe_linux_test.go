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
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/files"
)

type wordPressRoundTripper func(*http.Request) (*http.Response, error)

func (f wordPressRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func wordpressTestArchive(t *testing.T, name string) []byte {
	t.Helper()
	return wordpressTestArchiveType(t, name, tar.TypeReg)
}

func wordpressTestArchiveType(t *testing.T, name string, entryType byte) []byte {
	t.Helper()
	var output bytes.Buffer
	compressed := gzip.NewWriter(&output)
	archive := tar.NewWriter(compressed)
	header := &tar.Header{Name: name, Mode: 0600, Size: 4, Typeflag: entryType}
	if entryType == tar.TypeSymlink {
		header.Size = 0
		header.Linkname = "../../outside.txt"
	}
	if entryType == tar.TypeDir {
		header.Size = 0
	}
	if err := archive.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if entryType == tar.TypeReg {
		if _, err := archive.Write([]byte("core")); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func wordpressFixtureRelease(archive []byte) wordPressRelease {
	digest := sha256.Sum256(archive)
	return wordPressRelease{
		url:    "https://downloads.wordpress.org/release/wordpress-fixture.tar.gz",
		sha256: hex.EncodeToString(digest[:]),
	}
}

func wordpressFixtureTransport(archive []byte) http.RoundTripper {
	return wordPressRoundTripper(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(archive)),
			ContentLength: int64(len(archive)), Request: request,
		}, nil
	})
}

func TestPinnedWordPressReleaseManifest(t *testing.T) {
	parsed, err := url.Parse(wordpressReleaseURL)
	if err != nil || wordpressReleaseVersion == "" || parsed.Scheme != "https" ||
		parsed.Host != "downloads.wordpress.org" || parsed.Path != "/release/wordpress-"+wordpressReleaseVersion+".tar.gz" {
		t.Fatalf("pinned release URL is not canonical: %q (%v)", wordpressReleaseURL, err)
	}
	digest, err := hex.DecodeString(wordpressReleaseSHA256)
	if err != nil || len(digest) != sha256.Size {
		t.Fatalf("pinned release SHA-256 is invalid: %v", err)
	}
}

func TestWordPressCoreInstallVerifiesAndAtomicallyExtracts(t *testing.T) {
	archive := wordpressTestArchive(t, "wordpress/index.php")
	release := wordpressFixtureRelease(archive)
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := installWordPressCore(context.Background(), root, "site-a", release, wordpressFixtureTransport(archive)); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(rootPath, "site-a", "wordpress", "index.php"))
	if err != nil || string(content) != "core" {
		t.Fatalf("extracted core = %q, %v", content, err)
	}
	info, err := os.Stat(filepath.Join(rootPath, "site-a", "wordpress", "index.php"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("extracted file mode = %v, %v", info, err)
	}
}

func TestWordPressCoreInstallRejectsTamperingWithoutPublishing(t *testing.T) {
	archive := wordpressTestArchive(t, "wordpress/index.php")
	release := wordpressFixtureRelease(archive)
	release.sha256 = strings.Repeat("0", 64)
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := installWordPressCore(context.Background(), root, "site-a", release, wordpressFixtureTransport(archive)); !errors.Is(err, ErrInvalidWordPressRelease) {
		t.Fatalf("tampered archive error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(rootPath, "site-a")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tampered archive published output: %v", err)
	}
}

func TestWordPressCoreInstallRejectsUnsafeArchivesWithoutPublishing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeflag byte
	}{{"../outside.txt", tar.TypeReg}, {"wordpress/../../outside.txt", tar.TypeReg}, {"wordpress/link", tar.TypeSymlink}} {
		t.Run(tc.name, func(t *testing.T) {
			archive := wordpressTestArchiveType(t, tc.name, tc.typeflag)
			release := wordpressFixtureRelease(archive)
			rootPath := t.TempDir()
			root, err := files.OpenRoot(rootPath)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := installWordPressCore(context.Background(), root, "site-a", release, wordpressFixtureTransport(archive)); err == nil {
				t.Fatal("unsafe archive was accepted")
			}
			if _, err := os.Lstat(filepath.Join(rootPath, "site-a")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unsafe archive published output: %v", err)
			}
		})
	}
}

func TestWordPressCoreInstallRejectsUntrustedURLsStatusAndRedirects(t *testing.T) {
	archive := wordpressTestArchive(t, "wordpress/index.php")
	release := wordpressFixtureRelease(archive)
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	cases := []struct {
		name      string
		url       string
		transport http.RoundTripper
	}{
		{"plain HTTP", "http://downloads.wordpress.org/release/wordpress-fixture.tar.gz", wordpressFixtureTransport(archive)},
		{"untrusted host", "https://attacker.invalid/release/wordpress-fixture.tar.gz", wordpressFixtureTransport(archive)},
		{"path traversal version", "https://downloads.wordpress.org/release/wordpress-../payload.tar.gz", wordpressFixtureTransport(archive)},
		{"redirect status", release.url, wordPressRoundTripper(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://attacker.invalid/payload"}}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
		})},
		{"unexpected final URL", release.url, wordPressRoundTripper(func(request *http.Request) (*http.Response, error) {
			other := request.Clone(request.Context())
			other.URL = &url.URL{Scheme: "https", Host: "attacker.invalid", Path: "/payload"}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(archive)), Request: other}, nil
		})},
		{"response too large", release.url, wordPressRoundTripper(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, ContentLength: wordpressArchiveLimit + 1, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
		})},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := release
			candidate.url = tc.url
			if err := installWordPressCore(context.Background(), root, "site-"+string(rune('a'+index)), candidate, tc.transport); err == nil {
				t.Fatal("untrusted release response was accepted")
			}
		})
	}
}

func TestWordPressCoreInstallHonorsCancellation(t *testing.T) {
	archive := wordpressTestArchive(t, "wordpress/index.php")
	root, err := files.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	transport := wordPressRoundTripper(func(request *http.Request) (*http.Response, error) {
		return nil, request.Context().Err()
	})
	if err := installWordPressCore(ctx, root, "site-a", wordpressFixtureRelease(archive), transport); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled download error = %v", err)
	}
}

func TestConcurrentWordPressCoreInstallsPublishOnlyOneTree(t *testing.T) {
	archive := wordpressTestArchive(t, "wordpress/index.php")
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	release := wordpressFixtureRelease(archive)
	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wait sync.WaitGroup
	for range attempts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results <- installWordPressCore(context.Background(), root, "site-a", release, wordpressFixtureTransport(archive))
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	successes, alreadyExists := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, files.ErrAlreadyExists):
			alreadyExists++
		default:
			t.Errorf("concurrent install error = %v", err)
		}
	}
	if successes != 1 || alreadyExists != attempts-1 {
		t.Fatalf("successful=%d already-exists=%d, want 1 and %d", successes, alreadyExists, attempts-1)
	}
}

func TestPinnedWordPressCoreReleaseOnCleanGuest(t *testing.T) {
	if os.Getenv("TONG_TEST_PINNED_WORDPRESS_RELEASE") != "1" {
		t.Skip("set only by the disposable clean-VM workflow")
	}
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := InstallPinnedWordPressCore(context.Background(), root, "site-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootPath, "site-a", "wordpress", "wp-includes", "version.php")); err != nil {
		t.Fatalf("pinned WordPress core was not installed: %v", err)
	}
}
