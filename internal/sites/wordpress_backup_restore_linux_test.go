//go:build linux

package sites

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/files"
)

type wordpressBackupManifest struct {
	FormatVersion      int    `json:"format_version"`
	TenantID           string `json:"tenant_id"`
	CoreVersion        string `json:"core_version"`
	DatabaseName       string `json:"database_name"`
	DatabaseUser       string `json:"database_user"`
	DatabasePassword   string `json:"database_password"`
	TablePrefix        string `json:"table_prefix"`
	BlogTitle          string `json:"blog_title"`
	AdminUser          string `json:"admin_user"`
	DatabaseSHA256     string `json:"database_sha256"`
	ConfigSHA256       string `json:"config_sha256"`
	UploadMarkerSHA256 string `json:"upload_marker_sha256"`
}

// TestRestoreWordPressBackupForCleanGuest is enabled only on the disposable
// fresh guest. It authenticates/extracts with the bounded no-link file API,
// validates every manifest field and payload digest before creating a DB.
func TestRestoreWordPressBackupForCleanGuest(t *testing.T) {
	if os.Getenv("TONG_WORDPRESS_RESTORE_ROOT") == "" || os.Getenv("TONG_WORDPRESS_RESTORE_BUNDLE") == "" {
		t.Skip("set only by the disposable fresh-guest WordPress restore lab")
	}
	rootPath := os.Getenv("TONG_WORDPRESS_RESTORE_ROOT")
	if rootPath != "/srv/tong/tenants/tenant-a/sites/site-a/public" {
		t.Fatal("WordPress restore lab root is outside the fixed disposable tenant path")
	}
	bundlePath := os.Getenv("TONG_WORDPRESS_RESTORE_BUNDLE")
	bundleInfo, err := os.Lstat(bundlePath)
	if err != nil || !bundleInfo.Mode().IsRegular() || bundleInfo.Mode().Perm() != 0600 || bundleInfo.Size() < 1 || bundleInfo.Size() > files.MaxArchiveCompressedBytes {
		t.Fatal("decrypted WordPress bundle must be a private bounded regular file")
	}
	if _, err := os.Lstat(filepath.Join(rootPath, "wordpress")); !os.IsNotExist(err) {
		t.Fatal("fresh guest WordPress destination already exists")
	}
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatalf("open fixed fresh-guest document root: %v", err)
	}
	defer root.Close()
	bundle, err := os.Open(bundlePath)
	if err != nil {
		t.Fatal("open private authenticated WordPress bundle")
	}
	openedInfo, statErr := bundle.Stat()
	if statErr != nil || !os.SameFile(bundleInfo, openedInfo) {
		_ = bundle.Close()
		t.Fatal("private WordPress bundle changed while opening")
	}
	if err := root.ExtractGzipTarNewFrom("restore-stage", bundle, files.ArchiveLimits{}); err != nil {
		_ = bundle.Close()
		t.Fatalf("bounded WordPress bundle extraction failed: %v", err)
	}
	if err := bundle.Close(); err != nil {
		t.Fatal("close WordPress bundle")
	}
	stagePath := filepath.Join(rootPath, "restore-stage")
	defer os.RemoveAll(stagePath)
	if err := validateWordPressRestoreMembers(stagePath); err != nil {
		t.Fatalf("WordPress bundle member validation failed: %v", err)
	}
	manifestBytes, err := root.ReadFile("restore-stage/manifest.json", 64<<10)
	if err != nil {
		t.Fatal("read bounded WordPress restore manifest")
	}
	manifest, err := decodeWordPressBackupManifest(manifestBytes)
	if err != nil {
		t.Fatalf("WordPress restore manifest rejected: %v", err)
	}
	if err := validateWordPressBackupManifest(manifest); err != nil {
		t.Fatalf("WordPress restore manifest rejected: %v", err)
	}
	databasePath := filepath.Join(stagePath, "database.sql")
	configBytes, err := root.ReadFile("restore-stage/wp-config.php", 1<<20)
	if err != nil || string(configBytes[:min(len(configBytes), 5)]) != "<?php" {
		t.Fatal("WordPress bundle config is missing or invalid")
	}
	if err := checkWordPressRestoreDigest(databasePath, manifest.DatabaseSHA256); err != nil {
		t.Fatal("WordPress database dump digest does not match its manifest")
	}
	if got := digestBytes(configBytes); got != manifest.ConfigSHA256 {
		t.Fatal("WordPress configuration digest does not match its manifest")
	}
	uploadMarker, err := root.ReadFile("restore-stage/wp-content/uploads/tong-restore-marker.txt", 128)
	if err != nil || string(uploadMarker) != "private-wp-restore-upload-v1\n" || digestBytes(uploadMarker) != manifest.UploadMarkerSHA256 {
		t.Fatal("WordPress upload marker is missing or does not match its manifest")
	}
	writeWordPressRestoreCleanupIdentity(t, manifest)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := InstallPinnedWordPressCore(ctx, root, "core-stage"); err != nil {
		t.Fatalf("install pinned WordPress core on fresh guest: %v", err)
	}
	provisionWordPressRestoreDatabase(ctx, t, manifest)
	writeWordPressRestoreClientConfig(t, manifest)
	importWordPressRestoreDatabase(ctx, t, manifest.DatabaseName, databasePath)
	verifyWordPressRestoreRows(ctx, t, manifest)

	wordpressRoot := filepath.Join(rootPath, "core-stage", "wordpress")
	coreContent := filepath.Join(wordpressRoot, "wp-content")
	contentInfo, err := os.Lstat(coreContent)
	if err != nil || !contentInfo.IsDir() || contentInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatal("pinned core content path is not a regular directory")
	}
	if err := os.RemoveAll(coreContent); err != nil {
		t.Fatal("replace clean core content with restored tenant content")
	}
	if err := os.Rename(filepath.Join(stagePath, "wp-content"), coreContent); err != nil {
		t.Fatal("publish restored WordPress content in fresh guest site")
	}
	if err := root.WriteNew("core-stage/wordpress/wp-config.php", configBytes); err != nil {
		t.Fatalf("publish restored private wp-config.php: %v", err)
	}
	if err := os.Rename(wordpressRoot, filepath.Join(rootPath, "wordpress")); err != nil {
		t.Fatal("publish completed WordPress tree on fresh guest")
	}
	if err := os.Remove(filepath.Join(rootPath, "core-stage")); err != nil {
		t.Fatal("remove empty pinned-core staging directory")
	}
	if err := os.RemoveAll(stagePath); err != nil {
		t.Fatal("remove plaintext restore staging tree")
	}
	if err := os.Remove(bundlePath); err != nil {
		t.Fatal("remove decrypted WordPress bundle from fresh guest")
	}
}

func decodeWordPressBackupManifest(data []byte) (wordpressBackupManifest, error) {
	var manifest wordpressBackupManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return wordpressBackupManifest{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return wordpressBackupManifest{}, fmt.Errorf("trailing manifest data")
	}
	return manifest, nil
}

func validateWordPressBackupManifest(manifest wordpressBackupManifest) error {
	database := regexp.MustCompile(`^tp_wp_a_[0-9a-f]{16}$`)
	user := regexp.MustCompile(`^tu_wp_a_[0-9a-f]{16}$`)
	admin := regexp.MustCompile(`^tong_admin_a_[0-9a-f]{16}$`)
	password := regexp.MustCompile(`^[0-9a-f]{64}$`)
	digest := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if manifest.FormatVersion != 1 || manifest.TenantID != "tenant-a" || manifest.CoreVersion != wordpressReleaseVersion ||
		!database.MatchString(manifest.DatabaseName) || !user.MatchString(manifest.DatabaseUser) ||
		!password.MatchString(manifest.DatabasePassword) || !validWordPressTablePrefix(manifest.TablePrefix) ||
		manifest.BlogTitle != "Tong Tenant Alpha Lab" || !admin.MatchString(manifest.AdminUser) ||
		!digest.MatchString(manifest.DatabaseSHA256) || !digest.MatchString(manifest.ConfigSHA256) || !digest.MatchString(manifest.UploadMarkerSHA256) {
		return files.ErrInvalidArchive
	}
	return nil
}

func validateWordPressRestoreMembers(root string) error {
	allowedTop := map[string]bool{"manifest.json": true, "database.sql": true, "wp-config.php": true, "wp-content": true}
	var total int64
	entries := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return files.ErrInvalidArchive
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return files.ErrInvalidArchive
		}
		relative = filepath.ToSlash(relative)
		first := strings.SplitN(relative, "/", 2)[0]
		if !allowedTop[first] || entry.Type()&os.ModeSymlink != 0 {
			return files.ErrInvalidArchive
		}
		info, err := entry.Info()
		if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
			return files.ErrInvalidArchive
		}
		entries++
		if entries > files.MaxArchiveEntries {
			return files.ErrArchiveLimit
		}
		if info.Mode().IsRegular() {
			if info.Size() < 0 || info.Size() > files.MaxArchiveFileBytes {
				return files.ErrArchiveLimit
			}
			total += info.Size()
			if total > files.MaxArchiveTotalBytes {
				return files.ErrArchiveLimit
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, required := range []string{"manifest.json", "database.sql", "wp-config.php", "wp-content/uploads/tong-restore-marker.txt"} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(required))); err != nil {
			return files.ErrInvalidArchive
		}
	}
	return nil
}

func checkWordPressRestoreDigest(path, expected string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > files.MaxArchiveFileBytes {
		return files.ErrInvalidArchive
	}
	file, err := os.Open(path)
	if err != nil {
		return files.ErrInvalidArchive
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, io.LimitReader(file, files.MaxArchiveFileBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || hex.EncodeToString(hash.Sum(nil)) != expected {
		return files.ErrInvalidArchive
	}
	return nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func writeWordPressRestoreCleanupIdentity(t *testing.T, manifest wordpressBackupManifest) {
	t.Helper()
	path := "/home/ci/wordpress-restore-identifiers.json"
	data, err := json.Marshal(struct {
		Database string `json:"database"`
		User     string `json:"user"`
	}{manifest.DatabaseName, manifest.DatabaseUser})
	if err != nil {
		t.Fatal("could not stage private cleanup identifiers")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("could not stage fresh-guest database cleanup identifiers")
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		t.Fatal("could not write fresh-guest cleanup identifiers")
	}
	if err := file.Close(); err != nil {
		t.Fatal("could not close fresh-guest cleanup identifiers")
	}
}

func provisionWordPressRestoreDatabase(ctx context.Context, t *testing.T, manifest wordpressBackupManifest) {
	t.Helper()
	statement := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4;\nCREATE USER '%s'@'localhost' IDENTIFIED BY '%s';\nGRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost';\n",
		manifest.DatabaseName, manifest.DatabaseUser, manifest.DatabasePassword, manifest.DatabaseName, manifest.DatabaseUser)
	command := exec.CommandContext(ctx, "sudo", "mariadb", "--protocol=socket", "-uroot")
	command.Stdin = strings.NewReader(statement)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		t.Fatal("fresh-guest WordPress database provisioning failed")
	}
}

func writeWordPressRestoreClientConfig(t *testing.T, manifest wordpressBackupManifest) {
	t.Helper()
	path := "/home/ci/wp-restore-tenant.cnf"
	content := fmt.Sprintf("[client]\nuser=%s\npassword=%s\nhost=localhost\nprotocol=socket\n", manifest.DatabaseUser, manifest.DatabasePassword)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("could not stage owner-only restored database client credentials")
	}
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		t.Fatal("could not write owner-only restored database client credentials")
	}
	if err := file.Close(); err != nil {
		t.Fatal("could not close owner-only restored database client credentials")
	}
}

func importWordPressRestoreDatabase(ctx context.Context, t *testing.T, database, path string) {
	t.Helper()
	input, err := os.Open(path)
	if err != nil {
		t.Fatal("open private WordPress SQL dump")
	}
	defer input.Close()
	command := exec.CommandContext(ctx, "mariadb", "--defaults-extra-file=/home/ci/wp-restore-tenant.cnf", database)
	command.Stdin = input
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		t.Fatal("fresh-guest WordPress database import failed")
	}
}

func verifyWordPressRestoreRows(ctx context.Context, t *testing.T, manifest wordpressBackupManifest) {
	t.Helper()
	query := fmt.Sprintf("SELECT option_value FROM `%soptions` WHERE option_name='blogname'; SELECT user_login FROM `%susers` WHERE ID=1; SELECT option_value FROM `%soptions` WHERE option_name='tong_restore_marker'; SELECT COUNT(*) FROM `%sposts` WHERE post_status='publish';",
		manifest.TablePrefix, manifest.TablePrefix, manifest.TablePrefix, manifest.TablePrefix)
	command := exec.CommandContext(ctx, "mariadb", "--defaults-extra-file=/home/ci/wp-restore-tenant.cnf", "--batch", "--skip-column-names", manifest.DatabaseName, "-e", query)
	output, err := command.Output()
	if err != nil {
		t.Fatal("restored WordPress database rows could not be queried")
	}
	rows := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(rows) != 4 || rows[0] != manifest.BlogTitle || rows[1] != manifest.AdminUser || rows[2] != "private-wp-restore-db-v1" {
		t.Fatal("restored WordPress title, administrator, post, or private marker row is missing")
	}
	posts, err := strconv.Atoi(rows[3])
	if err != nil || posts < 1 {
		t.Fatal("restored WordPress published content rows are missing")
	}
}

func TestValidateWordPressBackupManifestRejectsUnsafeFields(t *testing.T) {
	valid := wordpressBackupManifest{
		FormatVersion: 1, TenantID: "tenant-a", CoreVersion: wordpressReleaseVersion,
		DatabaseName: "tp_wp_a_0123456789abcdef", DatabaseUser: "tu_wp_a_0123456789abcdef", DatabasePassword: strings.Repeat("a", 64),
		TablePrefix: "tenant_a_", BlogTitle: "Tong Tenant Alpha Lab", AdminUser: "tong_admin_a_0123456789abcdef",
		DatabaseSHA256: strings.Repeat("b", 64), ConfigSHA256: strings.Repeat("c", 64), UploadMarkerSHA256: strings.Repeat("d", 64),
	}
	if err := validateWordPressBackupManifest(valid); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	cases := []struct {
		name string
		edit func(*wordpressBackupManifest)
	}{
		{"wrong tenant", func(m *wordpressBackupManifest) { m.TenantID = "tenant-b" }},
		{"unexpected core", func(m *wordpressBackupManifest) { m.CoreVersion = "9.9.9" }},
		{"unsafe database name", func(m *wordpressBackupManifest) { m.DatabaseName = "db`;DROP DATABASE x;--" }},
		{"unsafe database user", func(m *wordpressBackupManifest) { m.DatabaseUser = "user'@'%'" }},
		{"unsafe password", func(m *wordpressBackupManifest) { m.DatabasePassword = "secret' OR 1=1 --" }},
		{"unsafe table prefix", func(m *wordpressBackupManifest) { m.TablePrefix = "1bad;" }},
		{"invalid digest", func(m *wordpressBackupManifest) { m.DatabaseSHA256 = "not-a-digest" }},
		{"wrong title", func(m *wordpressBackupManifest) { m.BlogTitle = "other site" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := valid
			tc.edit(&manifest)
			if err := validateWordPressBackupManifest(manifest); err == nil {
				t.Fatal("unsafe backup manifest was accepted")
			}
		})
	}
	if _, err := decodeWordPressBackupManifest([]byte(`{"format_version":1,"unexpected":true}`)); err == nil {
		t.Fatal("manifest decoder accepted an unknown field")
	}
}

func TestValidateWordPressRestoreMembersRejectsUnexpectedFilesAndLinks(t *testing.T) {
	makeValidStage := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		for _, name := range []string{"manifest.json", "database.sql", "wp-config.php", "wp-content/uploads/tong-restore-marker.txt"} {
			path := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	t.Run("valid member allowlist", func(t *testing.T) {
		if err := validateWordPressRestoreMembers(makeValidStage(t)); err != nil {
			t.Fatalf("valid stage rejected: %v", err)
		}
	})
	t.Run("unexpected top-level path", func(t *testing.T) {
		root := makeValidStage(t)
		if err := os.WriteFile(filepath.Join(root, "outside.txt"), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateWordPressRestoreMembers(root); err == nil {
			t.Fatal("unexpected restore path was accepted")
		}
	})
	t.Run("symlink member", func(t *testing.T) {
		root := makeValidStage(t)
		if err := os.Symlink(filepath.Join(root, "manifest.json"), filepath.Join(root, "wp-content", "uploads", "link")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if err := validateWordPressRestoreMembers(root); err == nil {
			t.Fatal("symlink restore member was accepted")
		}
	})
}
