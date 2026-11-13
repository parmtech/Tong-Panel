package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

func TestSiteListRechecksMembershipAtCatalogRead(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "alpha", "alpha.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	principal := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	// The principal represents successful middleware authentication. Disable
	// its identity before the actual catalog read to reproduce the gap. A site
	// creation audit references the membership and protects its deletion.
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='u1'"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.siteList(w, httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil), principal)
	if w.Code != 403 || strings.Contains(w.Body.String(), "alpha") {
		t.Fatalf("stale principal leaked: %d %s", w.Code, w.Body.String())
	}
	// Even the development bearer-only setup may not bypass the catalog DB.
	s.SQLJobs = nil
	r := call(s.Handler(), http.MethodGet, "/api/v1/sites", secret, "", "")
	if r.Code != 403 || strings.Contains(r.Body.String(), "alpha") {
		t.Fatalf("dev token bypass: %d %s", r.Code, r.Body.String())
	}
}

func TestSiteListFailsClosedWhenCatalogUsesDifferentDatabase(t *testing.T) {
	s, _, _ := setupSiteAPI(t)
	other, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "other.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })
	if _, err := other.Exec("INSERT INTO tenants(id,name) VALUES('tenant-a','A')"); err != nil {
		t.Fatal(err)
	}
	s.SiteCatalog, err = sites.NewCatalog(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SiteCatalog.Create(context.Background(), sites.SiteSpec{TenantID: "tenant-a", SiteID: "other", PrimaryDomain: "other.example.test", DocumentRoot: "sites/other/public"}); err != nil {
		t.Fatal(err)
	}
	r := call(s.Handler(), http.MethodGet, "/api/v1/sites", secret, "", "")
	if r.Code != 403 || strings.Contains(r.Body.String(), "other") {
		t.Fatalf("unrelated catalog granted: %d %s", r.Code, r.Body.String())
	}
}

func TestSiteListPagesScopeAndValidation(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	h := s.Handler()
	for _, item := range []struct{ token, id, domain string }{
		{secret, "alpha", "alpha.example.test"}, {secret, "bravo", "bravo.example.test"},
		{sqlOtherToken, "foreign", "foreign.example.test"},
	} {
		if r := createTestSite(t, h, item.token, item.id, item.domain); r.Code != 201 {
			t.Fatalf("create %s: %d %s", item.id, r.Code, r.Body.String())
		}
	}
	var page struct {
		Sites []sites.SiteSummary `json:"sites"`
		Next  string              `json:"next_after"`
	}
	for _, tc := range []struct{ token, query, want, next string }{
		{secret, "?limit=1", "alpha", "alpha"},
		{secret, "?prefix=al", "alpha", ""},
		{sqlAuditorToken, "?limit=1&after=alpha", "bravo", ""},
		{sqlOtherToken, "?after=alpha", "foreign", ""},
	} {
		r := call(h, http.MethodGet, "/api/v1/sites"+tc.query, tc.token, "", "")
		if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &page) != nil || len(page.Sites) != 1 || page.Sites[0].SiteID != tc.want || page.Next != tc.next {
			t.Fatalf("%s: %d %s", tc.query, r.Code, r.Body.String())
		}
	}
	r := call(h, http.MethodGet, "/api/v1/sites?after=foreign", secret, "", "")
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"sites":[]`) {
		t.Fatalf("foreign boundary: %d %s", r.Code, r.Body.String())
	}
	for _, query := range []string{"?limit=0", "?limit=51", "?limit=1&limit=2", "?limit=+1", "?after=", "?after=../x", "?tenant=tenant-b", "?limit=%ZZ", "?after=a&unknown=x", "?prefix=", "?prefix=%25", "?prefix=a&prefix=b", "?prefix=" + strings.Repeat("a", 64)} {
		r := call(h, http.MethodGet, "/api/v1/sites"+query, secret, "", "")
		if r.Code != 400 || !strings.Contains(r.Body.String(), "invalid_page") {
			t.Errorf("%s: %d %s", query, r.Code, r.Body.String())
		}
	}
	if r := call(h, http.MethodPost, "/api/v1/sites?limit=1", secret, "", `{}`); r.Code != 404 {
		t.Fatalf("POST query behavior changed: %d", r.Code)
	}
	if r := call(h, http.MethodGet, "/api/v1/sites", "", "", ""); r.Code != 401 {
		t.Fatalf("missing auth: %d", r.Code)
	}
	if r := call((&Server{Keys: s.Keys}).Handler(), http.MethodGet, "/api/v1/sites", secret, "", ""); r.Code != 503 {
		t.Fatalf("missing catalog: %d", r.Code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Body.Len() != 0 {
		t.Fatalf("canceled request wrote a response: %d %s", w.Code, w.Body.String())
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	s.siteList(w, httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil), rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
	if w.Code != 503 {
		t.Fatalf("closed catalog: %d", w.Code)
	}
}
