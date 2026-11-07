//go:build linux

package sites

import (
	"errors"
	"os"
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
