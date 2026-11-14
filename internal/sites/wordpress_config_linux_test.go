//go:build linux

package sites

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/files"
)

func TestWriteWordPressDatabaseConfigNewWritesPrivateEscapedConfig(t *testing.T) {
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	secret := "pa$word\\' ); file_put_contents('/tmp/tong-pwned', 'yes'); //"
	config := WordPressDatabaseConfig{
		Name: "tenant_db", User: "tenant_user", Password: secret,
		Host: "localhost:3306",
	}
	if err := WriteWordPressDatabaseConfigNew(root, "wp-config.php", config); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(rootPath, "wp-config.php")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %v, %v; want 0600", info, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	expected := "define( 'DB_PASSWORD', " + phpSingleQuoted(secret) + " );"
	if !strings.Contains(text, expected) {
		t.Fatalf("password was not encoded as a single PHP string literal: %s", text)
	}
	if strings.Contains(text, "$table_prefix = '1") {
		t.Fatal("numeric-leading table prefix was emitted")
	}
	for _, name := range []string{"AUTH_KEY", "SECURE_AUTH_KEY", "LOGGED_IN_KEY", "NONCE_KEY", "AUTH_SALT", "SECURE_AUTH_SALT", "LOGGED_IN_SALT", "NONCE_SALT"} {
		if !strings.Contains(text, "define( '"+name+"', '") {
			t.Errorf("generated configuration missing %s", name)
		}
	}
	if err := WriteWordPressDatabaseConfigNew(root, "wp-config.php", config); !errors.Is(err, files.ErrAlreadyExists) {
		t.Fatalf("second write error = %v, want ErrAlreadyExists", err)
	}
}

func TestWriteWordPressDatabaseConfigNewRejectsInvalidInputWithoutPublishing(t *testing.T) {
	cases := []struct {
		name   string
		config WordPressDatabaseConfig
	}{
		{"empty name", WordPressDatabaseConfig{User: "u", Password: "p", Host: "localhost"}},
		{"empty user", WordPressDatabaseConfig{Name: "db", Password: "p", Host: "localhost"}},
		{"empty password", WordPressDatabaseConfig{Name: "db", User: "u", Host: "localhost"}},
		{"empty host", WordPressDatabaseConfig{Name: "db", User: "u", Password: "p"}},
		{"control character", WordPressDatabaseConfig{Name: "db\nbad", User: "u", Password: "p", Host: "localhost"}},
		{"invalid utf8", WordPressDatabaseConfig{Name: "db", User: "u", Password: string([]byte{0xff}), Host: "localhost"}},
		{"oversized", WordPressDatabaseConfig{Name: "db", User: "u", Password: strings.Repeat("x", maxWordPressConfigValueBytes+1), Host: "localhost"}},
		{"invalid table prefix", WordPressDatabaseConfig{Name: "db", User: "u", Password: "p", Host: "localhost", TablePrefix: "1bad;"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rootPath := t.TempDir()
			root, err := files.OpenRoot(rootPath)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := WriteWordPressDatabaseConfigNew(root, "wp-config.php", tc.config); !errors.Is(err, ErrInvalidWordPressConfig) {
				t.Fatalf("invalid configuration error = %v", err)
			}
			if _, err := os.Lstat(filepath.Join(rootPath, "wp-config.php")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid configuration published a file: %v", err)
			}
		})
	}
}

func TestConcurrentWordPressConfigWritesPublishOnlyOneFile(t *testing.T) {
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	config := WordPressDatabaseConfig{Name: "db", User: "user", Password: "secret", Host: "localhost"}

	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wait sync.WaitGroup
	for range attempts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results <- WriteWordPressDatabaseConfigNew(root, "wp-config.php", config)
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	successes, exists := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, files.ErrAlreadyExists):
			exists++
		default:
			t.Errorf("concurrent config write error = %v", err)
		}
	}
	if successes != 1 || exists != attempts-1 {
		t.Fatalf("success=%d already-exists=%d; want 1 and %d", successes, exists, attempts-1)
	}
}

func TestWordPressDatabaseConfigParsesWithPHP(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP CLI is installed only in the disposable hosting guest")
	}
	rootPath := t.TempDir()
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	markerPath := filepath.Join(rootPath, "unexpected-execution")
	config := WordPressDatabaseConfig{
		Name:        "tenant'db\\$name",
		User:        "tenant\\'user",
		Password:    `literal $value \\' ); file_put_contents(` + phpSingleQuoted(markerPath) + `, 'unexpected'); //`,
		Host:        "db\\'host:3306",
		TablePrefix: "tenant_7_",
	}
	if err := WriteWordPressDatabaseConfigNew(root, "wp-config.php", config); err != nil {
		t.Fatal(err)
	}
	wpSettings := `<?php
echo json_encode([
    'name' => DB_NAME,
    'user' => DB_USER,
    'password' => DB_PASSWORD,
    'host' => DB_HOST,
    'prefix' => $table_prefix,
    'charset' => DB_CHARSET,
    'collate' => DB_COLLATE,
    'debug' => WP_DEBUG,
    'salts' => [AUTH_KEY, SECURE_AUTH_KEY, LOGGED_IN_KEY, NONCE_KEY, AUTH_SALT, SECURE_AUTH_SALT, LOGGED_IN_SALT, NONCE_SALT],
], JSON_THROW_ON_ERROR);
`
	if err := os.WriteFile(filepath.Join(rootPath, "wp-settings.php"), []byte(wpSettings), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(php, filepath.Join(rootPath, "wp-config.php")).Output()
	if err != nil {
		t.Fatalf("PHP rejected or failed to execute generated WordPress config against the harmless settings stub: %v", err)
	}
	var got struct {
		Name     string   `json:"name"`
		User     string   `json:"user"`
		Password string   `json:"password"`
		Host     string   `json:"host"`
		Prefix   string   `json:"prefix"`
		Charset  string   `json:"charset"`
		Collate  string   `json:"collate"`
		Debug    bool     `json:"debug"`
		Salts    []string `json:"salts"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("generated configuration did not pass values to the settings stub as valid JSON: %v", err)
	}
	if got.Name != config.Name || got.User != config.User || got.Password != config.Password ||
		got.Host != config.Host || got.Prefix != config.TablePrefix || got.Charset != "utf8mb4" ||
		got.Collate != "" || got.Debug {
		t.Fatal("generated configuration values did not round-trip exactly through PHP")
	}
	if _, err := os.Lstat(markerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("database configuration value executed PHP code instead of remaining literal data")
	}
	if len(got.Salts) != 8 {
		t.Fatal("generated configuration did not contain all eight authentication keys and salts")
	}
	seen := make(map[string]struct{}, len(got.Salts))
	for _, salt := range got.Salts {
		decoded, err := base64.RawURLEncoding.DecodeString(salt)
		if err != nil || len(decoded) != 64 || base64.RawURLEncoding.EncodeToString(decoded) != salt {
			t.Fatal("generated authentication key or salt is not a canonical encoding of 64 random bytes")
		}
		if _, exists := seen[salt]; exists {
			t.Fatal("generated authentication keys and salts were not independent")
		}
		seen[salt] = struct{}{}
	}
}
