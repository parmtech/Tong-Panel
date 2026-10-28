package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

type siteTXTStub struct {
	mu      sync.RWMutex
	records map[string][]string
	calls   []string
}

func (s *siteTXTStub) LookupTXT(_ context.Context, name string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, name)
	return append([]string(nil), s.records[name]...), nil
}

func (s *siteTXTStub) set(name string, values ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = make(map[string][]string)
	}
	s.records[name] = append([]string(nil), values...)
}

func (s *siteTXTStub) callCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.calls)
}

func setupSiteAPI(t *testing.T) (*Server, *siteTXTStub, *sql.DB) {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B');
		INSERT INTO identities(id,username) VALUES('u1','alice'),('u2','bob'),('u3','charlie'),('u4','dave');
		INSERT INTO memberships(tenant_id,identity_id,role)
		VALUES('tenant-a','u1','customer'),('tenant-a','u2','customer'),('tenant-b','u3','customer'),('tenant-a','u4','auditor');`); err != nil {
		t.Fatal(err)
	}
	keys, err := auth.NewDevStore(map[string]rbac.Principal{
		secret:          {UserID: "u1", TenantID: "tenant-a", Role: rbac.Customer},
		sqlUserBToken:   {UserID: "u2", TenantID: "tenant-a", Role: rbac.Customer},
		sqlOtherToken:   {UserID: "u3", TenantID: "tenant-b", Role: rbac.Customer},
		sqlAuditorToken: {UserID: "u4", TenantID: "tenant-a", Role: rbac.Auditor},
	})
	if err != nil {
		t.Fatal(err)
	}
	q, err := jobs.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := sites.NewCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	lookup := &siteTXTStub{records: make(map[string][]string)}
	server := &Server{Keys: keys, SQLJobs: q, SiteCatalog: catalog, DomainTXTLookup: lookup}
	return server, lookup, db
}

func createTestSite(t *testing.T, h http.Handler, token, siteID, domain string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"site_id":%q,"primary_domain":%q,"aliases":["www.%s"]}`, siteID, domain, domain)
	return call(h, http.MethodPost, "/api/v1/sites", token, "", body)
}

func TestSiteAPIRequiresAuthAndEnforcesTenantScope(t *testing.T) {
	server, _, _ := setupSiteAPI(t)
	h := server.Handler()
	if r := call(h, http.MethodPost, "/api/v1/sites", "", "", `{"site_id":"site-a","primary_domain":"a.example.test"}`); r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated site creation returned %d", r.Code)
	}
	created := createTestSite(t, h, secret, "site-a", "a.example.test")
	if created.Code != http.StatusCreated {
		t.Fatalf("site creation returned %d: %s", created.Code, created.Body.String())
	}
	var record sites.SiteRecord
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.TenantID != "tenant-a" || record.SiteID != "site-a" ||
		record.DocumentRoot != "sites/site-a/public" || len(record.Domains) != 2 ||
		record.Domains[0].OwnershipState != "pending" {
		t.Fatalf("site response trusted client scope or malformed: %+v", record)
	}
	if r := call(h, http.MethodGet, "/api/v1/sites/site-a", sqlOtherToken, "", ""); r.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant site read returned %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodPost, "/api/v1/sites", sqlAuditorToken, "", `{"site_id":"site-b","primary_domain":"b.example.test"}`); r.Code != http.StatusForbidden {
		t.Fatalf("auditor created a site: %d", r.Code)
	}
	if r := call(h, http.MethodPost, "/api/v1/sites", secret, "", `{"site_id":"site-b","primary_domain":"b.example.test","tenant_id":"tenant-b"}`); r.Code != http.StatusBadRequest {
		t.Fatalf("client-supplied tenant scope was not rejected: %d %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodPost, "/api/v1/sites", secret, "", `{"site_id":"site-a","primary_domain":"a.example.test"}`); r.Code != http.StatusConflict {
		t.Fatalf("duplicate site returned %d", r.Code)
	}
}

func TestDomainVerificationAPIIsSingleUseTenantBoundAndStoresOnlyDigest(t *testing.T) {
	server, lookup, db := setupSiteAPI(t)
	h := server.Handler()
	if r := createTestSite(t, h, secret, "site-a", "a.example.test"); r.Code != http.StatusCreated {
		t.Fatalf("create site: %d %s", r.Code, r.Body.String())
	}
	const beginPath = "/api/v1/sites/site-a/domains/a.example.test/verification"
	if r := call(h, http.MethodPost, beginPath, sqlOtherToken, "", ""); r.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant verification begin returned %d", r.Code)
	}
	if r := call(h, http.MethodPost, beginPath, sqlAuditorToken, "", ""); r.Code != http.StatusForbidden {
		t.Fatalf("auditor started verification: %d", r.Code)
	}
	begin := call(h, http.MethodPost, beginPath, secret, "", "")
	if begin.Code != http.StatusCreated {
		t.Fatalf("verification begin returned %d: %s", begin.Code, begin.Body.String())
	}
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(begin.Body.Bytes(), &instruction); err != nil {
		t.Fatal(err)
	}
	if instruction.Name != "_tong-panel-challenge.a.example.test" || len(instruction.Value) != 43 || instruction.ExpiresAt <= 0 {
		t.Fatalf("invalid verification instruction: %+v", instruction)
	}
	var stored string
	if err := db.QueryRow("SELECT verification_token_sha256 FROM site_domains WHERE domain='a.example.test'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 64 || stored == instruction.Value {
		t.Fatalf("one-time proof was not stored as a digest: %q", stored)
	}

	const completePath = "/api/v1/sites/site-a/domains/a.example.test/verification/complete"
	if r := call(h, http.MethodPost, completePath, secret, "", `{"value":"wrong-token"}`); r.Code != http.StatusForbidden {
		t.Fatalf("bad proof returned %d: %s", r.Code, r.Body.String())
	}
	if lookup.callCount() != 0 {
		t.Fatal("DNS lookup occurred before the submitted proof was authenticated")
	}
	lookup.set(instruction.Name, "unrelated", instruction.Value)
	if r := call(h, http.MethodPost, completePath, secret, "", fmt.Sprintf(`{"value":%q}`, instruction.Value)); r.Code != http.StatusOK {
		t.Fatalf("valid TXT proof returned %d: %s", r.Code, r.Body.String())
	}
	verified := call(h, http.MethodGet, "/api/v1/sites/site-a", secret, "", "")
	if verified.Code != http.StatusOK || !strings.Contains(verified.Body.String(), `"OwnershipState":"verified"`) {
		t.Fatalf("verified state not visible to same tenant: %d %s", verified.Code, verified.Body.String())
	}
	if r := call(h, http.MethodPost, completePath, secret, "", fmt.Sprintf(`{"value":%q}`, instruction.Value)); r.Code != http.StatusConflict {
		t.Fatalf("replayed proof returned %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodPost, completePath, secret, "", `{"value":"ok","tenant_id":"tenant-b"}`); r.Code != http.StatusBadRequest {
		t.Fatalf("unexpected completion field was accepted: %d", r.Code)
	}
}

func TestConcurrentDomainVerificationCompletionHasOneWinner(t *testing.T) {
	server, lookup, _ := setupSiteAPI(t)
	h := server.Handler()
	if r := createTestSite(t, h, secret, "site-race", "race.example.test"); r.Code != http.StatusCreated {
		t.Fatalf("create site: %d %s", r.Code, r.Body.String())
	}
	begin := call(h, http.MethodPost, "/api/v1/sites/site-race/domains/race.example.test/verification", secret, "", "")
	if begin.Code != http.StatusCreated {
		t.Fatalf("verification begin returned %d: %s", begin.Code, begin.Body.String())
	}
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(begin.Body.Bytes(), &instruction); err != nil {
		t.Fatal(err)
	}
	lookup.set(instruction.Name, instruction.Value)
	path := "/api/v1/sites/site-race/domains/race.example.test/verification/complete"
	body := fmt.Sprintf(`{"value":%q}`, instruction.Value)
	start := make(chan struct{})
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			codes <- call(h, http.MethodPost, path, secret, "", body).Code
		}()
	}
	close(start)
	a, b := <-codes, <-codes
	winners := 0
	for _, code := range []int{a, b} {
		if code == http.StatusOK {
			winners++
		} else if code != http.StatusConflict && code != http.StatusGone {
			t.Fatalf("unexpected concurrent completion status %d, %d", a, b)
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent proof completions had %d winners (%d, %d), want one", winners, a, b)
	}
}


func TestSiteDomainAPIOnlyRemovesTenantAliasAndReleasesClaim(t *testing.T) {
	server, _, _ := setupSiteAPI(t)
	h := server.Handler()
	if r := createTestSite(t, h, secret, "site-a", "a.example.test"); r.Code != http.StatusCreated {
		t.Fatalf("create site: %d %s", r.Code, r.Body.String())
	}
	const aliasPath = "/api/v1/sites/site-a/domains/www.a.example.test"
	const primaryPath = "/api/v1/sites/site-a/domains/a.example.test"
	if r := call(h, http.MethodDelete, aliasPath, "", "", ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous alias removal returned %d", r.Code)
	}
	if r := call(h, http.MethodDelete, aliasPath, sqlOtherToken, "", ""); r.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant alias removal returned %d", r.Code)
	}
	if r := call(h, http.MethodDelete, aliasPath, sqlAuditorToken, "", ""); r.Code != http.StatusForbidden {
		t.Fatalf("auditor alias removal returned %d", r.Code)
	}
	if r := call(h, http.MethodDelete, primaryPath, secret, "", ""); r.Code != http.StatusConflict {
		t.Fatalf("primary domain removal returned %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodDelete, aliasPath, secret, "", ""); r.Code != http.StatusNoContent {
		t.Fatalf("tenant alias removal returned %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, http.MethodDelete, aliasPath, secret, "", ""); r.Code != http.StatusNotFound {
		t.Fatalf("repeated alias removal returned %d", r.Code)
	}
	var record sites.SiteRecord
	got := call(h, http.MethodGet, "/api/v1/sites/site-a", secret, "", "")
	if got.Code != http.StatusOK {
		t.Fatalf("site read after alias removal returned %d", got.Code)
	}
	if err := json.Unmarshal(got.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Domains) != 1 || record.Domains[0].Name != "a.example.test" || record.Domains[0].Kind != "primary" {
		t.Fatalf("alias remained in site record: %+v", record.Domains)
	}
	// A released name is globally claimable again, but only by a new tenant/site request.
	reclaimed := call(h, http.MethodPost, "/api/v1/sites", sqlOtherToken, "", `{"site_id":"site-b","primary_domain":"b.example.test","aliases":["www.a.example.test"]}`)
	if reclaimed.Code != http.StatusCreated {
		t.Fatalf("released alias could not be claimed: %d %s", reclaimed.Code, reclaimed.Body.String())
	}
}
