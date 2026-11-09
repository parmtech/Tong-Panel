//go:build linux

package sites

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/files"
)

// TestPrepareWordPressRuntimeForCleanVM is enabled only by the disposable VM
// lab. Credentials arrive through a mode-0600 file and are never logged.
func TestPrepareWordPressRuntimeForCleanVM(t *testing.T) {
	rootPath := os.Getenv("TONG_WORDPRESS_RUNTIME_ROOT")
	credentialsPath := os.Getenv("TONG_WORDPRESS_RUNTIME_CREDENTIALS")
	if rootPath == "" || credentialsPath == "" {
		t.Skip("set only by the disposable WordPress runtime VM lab")
	}
	var config WordPressDatabaseConfig
	info, err := os.Lstat(credentialsPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 4096 {
		t.Fatal("private runtime credentials file must be a small regular file with mode 0600")
	}
	file, err := os.Open(credentialsPath)
	if err != nil {
		t.Fatal("could not read private runtime credentials")
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		t.Fatal("private runtime credentials changed while opening")
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(data) > 4096 {
		t.Fatal("could not read bounded private runtime credentials")
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal("private runtime credentials are invalid JSON")
	}
	root, err := files.OpenRoot(rootPath)
	if err != nil {
		t.Fatalf("open disposable site root: %v", err)
	}
	defer root.Close()
	if os.Getenv("TONG_WORDPRESS_RUNTIME_INSTALL") == "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err := InstallPinnedWordPressCore(ctx, root, "wordpress"); err != nil {
			t.Fatalf("install pinned WordPress core: %v", err)
		}
	} else if _, err := os.Stat(filepath.Join(rootPath, "wordpress", "wp-settings.php")); err != nil {
		t.Fatalf("pinned WordPress core is unavailable: %v", err)
	}
	if err := WriteWordPressDatabaseConfigNew(root, "wordpress/wp-config.php", config); err != nil {
		t.Fatalf("write private WordPress database configuration: %v", err)
	}
}
