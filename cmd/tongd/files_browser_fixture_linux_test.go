//go:build linux

package main

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/parmtech/tong-panel/internal/api"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/files"
)

// setupFilesBrowserFixture is used only by the explicitly selected local TLS
// test fixture. Its roots and their contents are private temporary test data.
func setupFilesBrowserFixture(t *testing.T, sessions *auth.SQLiteSessionStore) string {
	t.Helper()
	dirs := []string{t.TempDir(), t.TempDir()}
	for i := 0; i < 201; i++ {
		if err := os.WriteFile(filepath.Join(dirs[0], fmt.Sprintf("a-%03d.txt", i)), []byte("tenant A preview"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dirs[1], "b-only.txt"), []byte("tenant B preview"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := files.OpenRoot(dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	b, err := files.OpenRoot(dirs[1])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	roots, err := files.NewTenantRoots(map[string]*files.Root{"tenant-a": a, "tenant-b": b})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(publicHandler(&api.Server{Sessions: sessions, Files: api.NewTenantFileService(roots)}))
	t.Cleanup(server.Close)
	return server.URL
}
