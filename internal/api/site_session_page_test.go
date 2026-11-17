package api

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

func TestSiteSessionListPrefixIsLiteralAndTenantScopedAtAPI(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	sessions, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = sessions
	h := s.Handler()
	for _, item := range []struct{ token, id, domain string }{
		{secret, "app_01", "app-underscore.example.test"},
		{secret, "app_02", "app-underscore-second.example.test"},
		{secret, "appX01", "app-x.example.test"},
		{sqlOtherToken, "app_03", "tenant-b-match.example.test"},
	} {
		if r := createTestSite(t, h, item.token, item.id, item.domain); r.Code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", item.id, r.Code, r.Body.String())
		}
	}

	issue := func(tenantID, userID string) string {
		t.Helper()
		token, _, _, err := sessions.Issue(rbac.Principal{TenantID: tenantID, UserID: userID, Role: rbac.Customer}, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	type sitePage struct {
		Sites []sites.SiteSummary `json:"sites"`
		Next  string              `json:"next_after"`
	}
	readPage := func(token, path string) (*httptest.ResponseRecorder, sitePage) {
		t.Helper()
		r := cookieCall(h, http.MethodGet, path, token, "", "", "")
		var page sitePage
		if r.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, r.Code, r.Body.String())
		}
		if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
			t.Fatalf("%s: decode page: %v", path, err)
		}
		return r, page
	}
	assertOnlySite := func(path string, r *httptest.ResponseRecorder, page sitePage, wantSiteID, wantDomain, wantNext string) {
		t.Helper()
		if len(page.Sites) != 1 || page.Sites[0].SiteID != wantSiteID || page.Sites[0].PrimaryDomain != wantDomain || page.Next != wantNext {
			t.Fatalf("%s: returned %#v with cursor %q, want only %s at %s and cursor %q", path, page.Sites, page.Next, wantSiteID, wantDomain, wantNext)
		}
		for _, excluded := range []string{"appX01", "app-x.example.test", "app_01", "app-underscore.example.test", "app_02", "app-underscore-second.example.test", "app_03", "tenant-b-match.example.test"} {
			if excluded != wantSiteID && excluded != wantDomain && strings.Contains(r.Body.String(), excluded) {
				t.Fatalf("%s: response disclosed excluded site %q: %s", path, excluded, r.Body.String())
			}
		}
	}

	tenantAToken := issue("tenant-a", "u1")
	path := "/api/v1/sites?prefix=app_&limit=1"
	r, page := readPage(tenantAToken, path)
	assertOnlySite(path, r, page, "app_01", "app-underscore.example.test", "app_01")

	path += "&after=" + page.Next
	r, page = readPage(tenantAToken, path)
	assertOnlySite(path, r, page, "app_02", "app-underscore-second.example.test", "")

	tenantBPath := "/api/v1/sites?prefix=app_&limit=1"
	r, page = readPage(issue("tenant-b", "u3"), tenantBPath)
	assertOnlySite(tenantBPath, r, page, "app_03", "tenant-b-match.example.test", "")
}

func TestSiteListRechecksAuthenticatedSessionSnapshot(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "alpha", "alpha.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	sessions, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = sessions
	principal := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	issue := func() string {
		token, _, _, e := sessions.Issue(principal, time.Hour)
		if e != nil {
			t.Fatal(e)
		}
		return token
	}
	request := func(token string) (*http.Request, rbac.Principal) {
		req := httptest.NewRequest(http.MethodGet, "https://panel.test/api/v1/sites", nil)
		req.TLS = &tls.ConnectionState{}
		cookie := &http.Cookie{Name: auth.BrowserCookieName, Value: token}
		req.AddCookie(cookie)
		p, status, _ := s.cookiePrincipal(req, cookie)
		if status != 0 {
			t.Fatalf("authentication: %d", status)
		}
		proof, ok := sessions.ReadProof(token)
		if !ok {
			t.Fatal("missing proof")
		}
		return req.WithContext(context.WithValue(req.Context(), sessionProofKey{}, proof)), p
	}
	run := func(req *http.Request, p rbac.Principal, want int) {
		t.Helper()
		w := httptest.NewRecorder()
		s.siteList(w, req, p)
		if w.Code != want || (want != 200 && strings.Contains(w.Body.String(), "alpha")) {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
	}
	token := issue()
	if response := cookieCall(s.Handler(), http.MethodGet, "/api/v1/sites", token, "", "", ""); response.Code != 200 || !strings.Contains(response.Body.String(), "alpha") {
		t.Fatalf("actual cookie path: %d %s", response.Code, response.Body.String())
	}
	req, p := request(token)
	run(req, p, 200)
	if err := sessions.Revoke(token); err != nil {
		t.Fatal(err)
	}
	run(req, p, 401)
	if response := cookieCall(s.Handler(), http.MethodGet, "/api/v1/sites", token, "", "", ""); response.Code != 401 {
		t.Fatalf("revoked cookie: %d", response.Code)
	}
	token = issue()
	req, p = request(token)
	if _, _, _, err := sessions.Rotate(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	run(req, p, 401)
	token = issue()
	req, p = request(token)
	if _, err := db.Exec("UPDATE sessions SET expires_at=? WHERE identity_id='u1' AND revoked_at IS NULL", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	run(req, p, 401)
	raw, _ := hex.DecodeString(token)
	digest := sha256.Sum256(raw)
	var revoked bool
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("observed expired session not revoked: %v %v", revoked, err)
	}
	token = issue()
	req, p = request(token)
	if _, err := db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE identity_id='u1' AND revoked_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	run(req, p, 401)
	raw, _ = hex.DecodeString(token)
	digest = sha256.Sum256(raw)
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("observed idle timeout not revoked: %v %v", revoked, err)
	}
	token = issue()
	req, p = request(token)
	if _, err := db.Exec("UPDATE sessions SET expires_at='malformed' WHERE identity_id='u1' AND revoked_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	run(req, p, 401)
}

func TestSiteSessionReadRequiresSameDatabasePool(t *testing.T) {
	s, _, _ := setupSiteAPI(t)
	other, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "separate.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	sessions, err := auth.NewSQLiteSessionStore(other)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := sessions.ReadProof(strings.Repeat("ab", 32))
	if !ok {
		t.Fatal("proof")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil)
	req = req.WithContext(context.WithValue(req.Context(), sessionProofKey{}, proof))
	w := httptest.NewRecorder()
	s.siteList(w, req, rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
	if w.Code != 503 {
		t.Fatalf("separate pool: %d %s", w.Code, w.Body.String())
	}
}
