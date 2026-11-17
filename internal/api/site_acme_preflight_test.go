package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSiteACMEHTTP01PreflightIsReadOnlyAndTenantScoped(t *testing.T) {
	s, lookup, db := setupSiteAPI(t)
	h := s.Handler()
	const route = "/api/v1/sites/site-a/preflight/acme-http01"
	if got := createTestSite(t, h, secret, "site-a", "a.example.test"); got.Code != 201 {
		t.Fatal(got.Body.String())
	}
	for _, tc := range []struct {
		method, path, token string
		status              int
	}{
		{"GET", route, "", 401}, {"GET", route, secret, 409}, {"GET", route, sqlOtherToken, 404},
		{"POST", route, secret, 405}, {"GET", route + "?tenant_id=tenant-b", secret, 400},
	} {
		if got := call(h, tc.method, tc.path, tc.token, "", ""); got.Code != tc.status {
			t.Fatalf("%s %s: %d want %d", tc.method, tc.path, got.Code, tc.status)
		}
	}
	verifySiteDomainViaAPI(t, h, lookup, secret, "site-a", "a.example.test")
	beforeDNS := lookup.callCount()
	for _, token := range []string{secret, sqlAuditorToken} {
		got := call(h, "GET", route, token, "", "")
		if got.Code != 200 {
			t.Fatalf("preflight: %d %s", got.Code, got.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(got.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 5 || body["site_id"] != "site-a" || body["challenge"] != "http-01" || body["snapshot_only"] != true || body["execution_enabled"] != false {
			t.Fatalf("unexpected preflight shape: %v", body)
		}
		domains, ok := body["domains"].([]any)
		if !ok || len(domains) != 1 || domains[0] != "a.example.test" {
			t.Fatal("unverified alias included")
		}
		if got.Header().Get("Cache-Control") != "no-store" || got.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("missing response guards")
		}
		if strings.Contains(got.Body.String(), secret) {
			t.Fatal("secret exposed")
		}
	}
	if lookup.callCount() != beforeDNS {
		t.Fatal("preflight queried DNS")
	}
	var jobs int
	if err := db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobs); err != nil || jobs != 0 {
		t.Fatal("preflight queued work")
	}
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='u1'"); err != nil {
		t.Fatal(err)
	}
	if got := call(h, "GET", route, secret, "", ""); got.Code != 401 {
		t.Fatalf("disabled identity: %d", got.Code)
	}
	s.SiteCatalog = nil
	if got := call(s.Handler(), "GET", route, sqlAuditorToken, "", ""); got.Code != 503 {
		t.Fatalf("missing backend: %d", got.Code)
	}
}
