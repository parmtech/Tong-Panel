package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestSiteDatabasePaginationAPICompatibility(t *testing.T) {
	s, _, _ := setupSiteAPI(t)
	h := s.Handler()
	if w := createTestSite(t, h, secret, "example", "primary.example.test"); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	for _, id := range []string{"db-a", "db-b"} {
		w := call(h, http.MethodPost, "/api/v1/sites/example/databases", secret, "", `{"database_id":"`+id+`"}`)
		if w.Code != 201 {
			t.Fatalf("create %d %s", w.Code, w.Body.String())
		}
	}
	legacy := call(h, http.MethodGet, "/api/v1/sites/example/databases", secret, "", "")
	if legacy.Code != 200 || strings.Contains(legacy.Body.String(), "next_after") {
		t.Fatalf("legacy %d %s", legacy.Code, legacy.Body.String())
	}
	first := call(h, http.MethodGet, "/api/v1/sites/example/databases?page=1&limit=1", secret, "", "")
	if first.Code != 200 || !strings.Contains(first.Body.String(), `"next_after":"db-a"`) || strings.Contains(first.Body.String(), `"DatabaseID":"db-b"`) {
		t.Fatalf("first %d %s", first.Code, first.Body.String())
	}
	last := call(h, http.MethodGet, "/api/v1/sites/example/databases?page=1&limit=1&after=db-a", secret, "", "")
	if last.Code != 200 || !strings.Contains(last.Body.String(), `"DatabaseID":"db-b"`) || !strings.Contains(last.Body.String(), `"next_after":""`) {
		t.Fatalf("last %d %s", last.Code, last.Body.String())
	}
	for _, q := range []string{"?limit=1", "?page=0", "?page=1&limit=51", "?page=1&after=..", "?page=1&after=x&after=y", "?page=1&tenant=other"} {
		if w := call(h, http.MethodGet, "/api/v1/sites/example/databases"+q, secret, "", ""); w.Code != 400 {
			t.Errorf("%s: %d %s", q, w.Code, w.Body.String())
		}
	}
	if w := call(h, http.MethodGet, "/api/v1/sites/missing/databases?page=1", secret, "", ""); w.Code != 404 {
		t.Fatalf("missing site %d %s", w.Code, w.Body.String())
	}
}
