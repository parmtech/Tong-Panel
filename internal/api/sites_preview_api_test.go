package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/sites"
)

func verifySiteDomainViaAPI(t *testing.T, h http.Handler, lookup *siteTXTStub, token, siteID, domain string) {
	t.Helper()
	path := "/api/v1/sites/" + siteID + "/domains/" + domain + "/verification"
	begin := call(h, http.MethodPost, path, token, "", "")
	if begin.Code != http.StatusCreated {
		t.Fatalf("begin ownership proof returned %d: %s", begin.Code, begin.Body.String())
	}
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(begin.Body.Bytes(), &instruction); err != nil {
		t.Fatal(err)
	}
	lookup.set(instruction.Name, instruction.Value)
	complete := call(h, http.MethodPost, path+"/complete", token, "", `{"value":"`+instruction.Value+`"}`)
	if complete.Code != http.StatusOK {
		t.Fatalf("complete ownership proof returned %d: %s", complete.Code, complete.Body.String())
	}
}

func TestSiteNginxPreviewUsesTenantVerifiedDomainsAndKeepsPendingAliasesOut(t *testing.T) {
	server, lookup, _ := setupSiteAPI(t)
	server.SiteHostingPaths = &sites.HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	h := server.Handler()
	const previewPath = "/api/v1/sites/site-a/preview/nginx"
	if r := call(h, http.MethodGet, previewPath, "", "", ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous preview returned %d", r.Code)
	}
	if r := createTestSite(t, h, secret, "site-a", "a.example.test"); r.Code != http.StatusCreated {
		t.Fatalf("create site: %d %s", r.Code, r.Body.String())
	}
	verifySiteDomainViaAPI(t, h, lookup, secret, "site-a", "a.example.test")

	if r := call(h, http.MethodGet, previewPath, sqlOtherToken, "", ""); r.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant preview returned %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodPost, previewPath, secret, "", ""); r.Code != http.StatusMethodNotAllowed {
		t.Fatalf("non-GET preview returned %d", r.Code)
	}
	if r := call(h, http.MethodGet, previewPath+"?tenant_id=tenant-b", secret, "", ""); r.Code != http.StatusBadRequest {
		t.Fatalf("preview query override returned %d", r.Code)
	}

	preview := call(h, http.MethodGet, previewPath, secret, "", "")
	if preview.Code != http.StatusOK {
		t.Fatalf("verified tenant preview returned %d: %s", preview.Code, preview.Body.String())
	}
	if got := preview.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("preview Content-Type=%q", got)
	}
	if preview.Header().Get("Cache-Control") != "no-store" || preview.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("preview missing no-store/nosniff headers: %#v", preview.Header())
	}
	if !strings.Contains(preview.Body.String(), "server_name a.example.test;") ||
		strings.Contains(preview.Body.String(), "www.a.example.test") {
		t.Fatalf("preview did not use verified primary or included pending alias:\n%s", preview.Body.String())
	}

	auditor := call(h, http.MethodGet, previewPath, sqlAuditorToken, "", "")
	if auditor.Code != http.StatusOK || auditor.Body.String() != preview.Body.String() {
		t.Fatalf("auditor read-only preview returned %d: %s", auditor.Code, auditor.Body.String())
	}
}

func TestSiteNginxPreviewFailsClosedWhenConfigOrOwnershipIsMissing(t *testing.T) {
	server, lookup, _ := setupSiteAPI(t)
	h := server.Handler()
	if r := createTestSite(t, h, secret, "site-pending", "pending.example.test"); r.Code != http.StatusCreated {
		t.Fatalf("create pending site: %d %s", r.Code, r.Body.String())
	}
	const previewPath = "/api/v1/sites/site-pending/preview/nginx"
	if r := call(h, http.MethodGet, previewPath, secret, "", ""); r.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured preview returned %d", r.Code)
	}

	server.SiteHostingPaths = &sites.HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	r := call(server.Handler(), http.MethodGet, previewPath, secret, "", "")
	if r.Code != http.StatusConflict || !strings.Contains(r.Body.String(), `"site_not_ready"`) {
		t.Fatalf("pending-primary preview returned %d: %s", r.Code, r.Body.String())
	}

	verifySiteDomainViaAPI(t, server.Handler(), lookup, secret, "site-pending", "pending.example.test")
	server.SiteHostingPaths = &sites.HostingPaths{TenantRoot: "/srv/../etc", PHPSocketRoot: "/run/tong/php"}
	r = call(server.Handler(), http.MethodGet, previewPath, secret, "", "")
	if r.Code != http.StatusServiceUnavailable {
		t.Fatalf("unsafe server-owned path preview returned %d", r.Code)
	}
}
