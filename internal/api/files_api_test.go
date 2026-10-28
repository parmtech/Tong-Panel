//go:build linux

package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/files"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func attachTestTenantFiles(t *testing.T, s *Server) (string, string) {
	t.Helper()
	rootAPath, rootBPath := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(rootAPath, "private.txt"), []byte("tenant-a-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBPath, "private.txt"), []byte("tenant-b-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(rootAPath, "public"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootAPath, "public", "index.html"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(external, []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(rootAPath, "escape")); err != nil {
		t.Fatal(err)
	}

	rootA, err := files.OpenRoot(rootAPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootA.Close() })
	rootB, err := files.OpenRoot(rootBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootB.Close() })
	roots, err := files.NewTenantRoots(map[string]*files.Root{"tenant-a": rootA, "tenant-b": rootB})
	if err != nil {
		t.Fatal(err)
	}
	s.Files = NewTenantFileService(roots)
	return rootAPath, rootBPath
}

func sessionGET(h http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "https://panel.test"+path, nil)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFilesReadAPIUsesAuthenticatedTenantAndRejectsAmbiguousPaths(t *testing.T) {
	s, sessions, _, tokenA, _ := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)
	tokenB, _, _, err := sessions.Issue(rbac.Principal{UserID: "u3", TenantID: "tenant-b", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()

	for _, tc := range []struct{ token, want string }{{tokenA, "tenant-a-secret"}, {tokenB, "tenant-b-secret"}} {
		rec := sessionGET(h, "/api/v1/files/content?path=private.txt", tc.token)
		if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
			t.Fatalf("tenant content status=%d body=%q; want %q", rec.Code, rec.Body.String(), tc.want)
		}
		if rec.Header().Get("Content-Type") != "application/octet-stream" ||
			rec.Header().Get("Cache-Control") != "no-store" ||
			rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("unsafe file response headers: %#v", rec.Header())
		}
	}
	if _, err := os.Stat(filepath.Join(rootA, "private.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "private.txt")); err != nil {
		t.Fatal(err)
	}

	listing := sessionGET(h, "/api/v1/files?path=public", tokenA)
	if listing.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listing.Code, listing.Body.String())
	}
	var body struct {
		Entries []TenantFileEntry `json:"entries"`
	}
	if err := json.Unmarshal(listing.Body.Bytes(), &body); err != nil || len(body.Entries) != 1 || body.Entries[0].Name != "index.html" {
		t.Fatalf("listing=%s entries=%#v err=%v", listing.Body.String(), body.Entries, err)
	}

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/v1/files/content?path=../tenant-b/private.txt", http.StatusBadRequest},
		{"/api/v1/files/content?path=private.txt&path=escape", http.StatusBadRequest},
		{"/api/v1/files/content?path=private.txt&tenant=tenant-b", http.StatusBadRequest},
		{"/api/v1/files/content?path=escape", http.StatusBadRequest},
		{"/api/v1/files/content?path=missing.txt", http.StatusNotFound},
		{"/api/v1/files?path=public&limit=0", http.StatusBadRequest},
	} {
		rec := sessionGET(h, tc.path, tokenA)
		if rec.Code != tc.want || rec.Body.String() == "tenant-b-secret" || rec.Body.String() == "outside-secret" {
			t.Errorf("file request %q returned %d %q; want %d", tc.path, rec.Code, rec.Body.String(), tc.want)
		}
	}
}

func TestFilesReadAPIDefaultClosedAuthRoleAndReadOnlyMethods(t *testing.T) {
	s, sessions, _, token, csrf := setupCookieAPI(t)
	attachTestTenantFiles(t, s)
	h := s.Handler()

	unauthenticated := httptest.NewRequest(http.MethodGet, "https://panel.test/api/v1/files", nil)
	unauthenticated.TLS = &tls.ConnectionState{}
	unauthenticatedResponse := httptest.NewRecorder()
	h.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated file list status=%d, want 401", unauthenticatedResponse.Code)
	}

	if rec := cookieCall(h, http.MethodPost, "/api/v1/files?path=write.txt", token, csrf, "https://panel.test", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("file mutation method status=%d body=%s, want 405", rec.Code, rec.Body.String())
	}
	if rec := sessionGET(h, "/api/v1/files", token); rec.Code != http.StatusOK {
		t.Fatalf("authenticated list status=%d body=%s", rec.Code, rec.Body.String())
	}

	_, _, _, auditorToken, err := sessions.Issue(rbac.Principal{UserID: "u4", TenantID: "tenant-a", Role: rbac.Auditor}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if rec := sessionGET(h, "/api/v1/files", auditorToken); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor file list status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}

	s.Files = nil
	if rec := sessionGET(s.Handler(), "/api/v1/files", token); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured file service status=%d body=%s, want 503", rec.Code, rec.Body.String())
	}
}

