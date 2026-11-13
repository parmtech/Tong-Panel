//go:build linux

package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/sites"
)

type openAPIContractCase struct {
	Name            string              `json:"name"`
	OperationID     string              `json:"operation_id"`
	Method          string              `json:"method"`
	Path            string              `json:"path"`
	RequestHeaders  map[string][]string `json:"request_headers"`
	RequestBody     []byte              `json:"request_body"`
	Status          int                 `json:"status"`
	ResponseHeaders map[string][]string `json:"response_headers"`
	ResponseBody    []byte              `json:"response_body"`
}

// This exercises configured library backends. Synthetic TXT verification and
// catalog database records do not provision DNS, hosting, or a database server.
func TestCurrentOpenAPIContractScenarios(t *testing.T) {
	s, lookup, _ := setupSiteAPI(t)
	h := s.Handler()
	var cases []openAPIContractCase
	covered := make(map[string]bool)
	requestID := regexp.MustCompile(`^[0-9a-f]{16}$`)
	headers := func(h http.Header, names ...string) map[string][]string {
		out := make(map[string][]string)
		for _, name := range names {
			if values := h.Values(name); len(values) != 0 {
				out[name] = append([]string(nil), values...)
			}
		}
		return out
	}
	run := func(handler http.Handler, name, operation, method, path, token, contentType, body string, status int, code string, customize func(*http.Request)) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "https://panel.test"+path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if customize != nil {
			customize(req)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s: status %d, want %d", name, rec.Code, status)
		}
		if !requestID.MatchString(rec.Header().Get("X-Request-ID")) {
			t.Fatalf("%s: invalid request ID", name)
		}
		if code != "" {
			var envelope struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Error.Code != code {
				t.Fatalf("%s: expected error code %s", name, code)
			}
		}
		if status >= 200 && status < 300 {
			covered[operation] = true
		}
		cases = append(cases, openAPIContractCase{name, operation, method, req.URL.RequestURI(), headers(req.Header, "Content-Type", "Accept", "Origin", "Idempotency-Key"), []byte(body), rec.Code, headers(rec.Header(), "Content-Type", "Cache-Control", "X-Content-Type-Options", "X-Request-ID"), append([]byte(nil), rec.Body.Bytes()...)})
		return rec
	}
	plain := func(name, operation, method, path, body string, status int) *httptest.ResponseRecorder {
		ct := ""
		if body != "" {
			ct = "application/json"
		}
		return run(h, name, operation, method, path, secret, ct, body, status, "", nil)
	}
	idem := func(req *http.Request) { req.Header.Set("Idempotency-Key", "contract-job-0001") }
	run(h, "health", "getHealth", "GET", "/healthz", "", "", "", 200, "", nil)
	run(h, "version", "getVersion", "GET", "/api/v1/version", "", "", "", 200, "", nil)
	me := plain("bearer principal", "getCurrentPrincipal", "GET", "/api/v1/me", "", 200)
	if !strings.Contains(me.Body.String(), `"TenantID":"tenant-a"`) {
		t.Fatal("principal tenant mismatch")
	}
	run(h, "missing auth", "getCurrentPrincipal", "GET", "/api/v1/me", "", "", "", 401, "unauthorized", nil)
	run((&Server{}).Handler(), "unconfigured auth", "getCurrentPrincipal", "GET", "/api/v1/me", "", "", "", 503, "auth_unconfigured", nil)
	run((&Server{Keys: s.Keys}).Handler(), "unconfigured jobs", "listJobs", "GET", "/api/v1/jobs", secret, "", "", 503, "jobs_unconfigured", nil)
	plain("empty SQLite queue", "listJobs", "GET", "/api/v1/jobs", "", 200)
	plain("empty SQLite bounded page", "listJobs", "GET", "/api/v1/jobs?limit=2", "", 200)
	run(h, "invalid page limit", "listJobs", "GET", "/api/v1/jobs?limit=101", secret, "", "", 400, "invalid_page", nil)
	journal := setup(t)
	run(journal.Handler(), "empty journal queue", "listJobs", "GET", "/api/v1/jobs", secret, "", "", 200, "", nil)
	created := run(h, "enqueue", "createJob", "POST", "/api/v1/jobs", secret, "application/json", `{"kind":"validation.noop"}`, 202, "", idem)
	var job jobs.Job
	if err := json.Unmarshal(created.Body.Bytes(), &job); err != nil || job.ID == "" {
		t.Fatal("missing created job ID")
	}
	run(h, "enqueue additional page fixture", "createJob", "POST", "/api/v1/jobs", secret, "application/json", `{"kind":"validation.noop"}`, 202, "", func(req *http.Request) { req.Header.Set("Idempotency-Key", "contract-page-extra-0001") })
	pageResponse := plain("bounded page with continuation", "listJobs", "GET", "/api/v1/jobs?limit=1", "", 200)
	var page struct {
		Jobs []jobs.Job `json:"jobs"`
		Next string     `json:"next_after"`
	}
	if err := json.Unmarshal(pageResponse.Body.Bytes(), &page); err != nil || len(page.Jobs) != 1 || page.Next != page.Jobs[0].ID {
		t.Fatal("bounded contract page missing continuation")
	}
	plain("bounded page after job", "listJobs", "GET", "/api/v1/jobs?after="+job.ID, "", 200)
	replay := run(h, "idempotent enqueue", "createJob", "POST", "/api/v1/jobs", secret, "application/json", `{"kind":"validation.noop"}`, 202, "", idem)
	var replayJob jobs.Job
	if err := json.Unmarshal(replay.Body.Bytes(), &replayJob); err != nil || replayJob.ID != job.ID {
		t.Fatal("idempotency replay changed job")
	}
	run(h, "missing idempotency", "createJob", "POST", "/api/v1/jobs", secret, "application/json", `{"kind":"validation.noop"}`, 400, "idempotency_required", nil)
	run(h, "oversized job JSON", "createJob", "POST", "/api/v1/jobs", secret, "application/json", `{"kind":"`+strings.Repeat("x", 2050)+`"}`, 400, "invalid_request", idem)
	run(h, "auditor cannot enqueue", "createJob", "POST", "/api/v1/jobs", sqlAuditorToken, "application/json", `{"kind":"validation.noop"}`, 403, "forbidden", idem)
	jobPath := "/api/v1/jobs/" + job.ID
	plain("read own job", "getJob", "GET", jobPath, "", 200)
	run(h, "same tenant other owner", "getJob", "GET", jobPath, sqlUserBToken, "", "", 404, "not_found", nil)
	run(h, "other tenant job", "getJob", "GET", jobPath, sqlOtherToken, "", "", 404, "not_found", nil)
	cancelled := plain("cancel own queued job", "cancelJob", "DELETE", jobPath, "", 200)
	if !strings.Contains(cancelled.Body.String(), `"state":"cancelled"`) {
		t.Fatal("job was not cancelled")
	}
	run(h, "files backend absent", "listFiles", "GET", "/api/v1/files", secret, "", "", 503, "files_unavailable", nil)
	attachTestTenantFiles(t, s)
	s.FileQuota = newTestFileQuota(1 << 20)
	h = s.Handler()
	plain("list rooted public directory", "listFiles", "GET", "/api/v1/files?path=public", "", 200)
	filePath := "/api/v1/files/content?path=public%2Fcontract.bin"
	run(h, "create rooted file", "createFile", "POST", filePath, secret, "application/octet-stream", "contract-content", 201, "", nil)
	read := plain("read rooted file", "readFile", "GET", filePath, "", 200)
	if read.Body.String() != "contract-content" {
		t.Fatal("file round trip mismatch")
	}
	plain("delete rooted file", "deleteFile", "DELETE", filePath, "", 204)
	plain("create catalog site", "createSite", "POST", "/api/v1/sites", `{"site_id":"contract-site","primary_domain":"contract.example.test","aliases":[]}`, 201)
	plain("list catalog sites", "listSites", "GET", "/api/v1/sites?limit=1", "", 200)
	run(h, "invalid site page", "listSites", "GET", "/api/v1/sites?limit=51", secret, "", "", 400, "invalid_page", nil)
	sitePath := "/api/v1/sites/contract-site"
	run(h, "pending primary cannot prepare ACME", "preflightSiteACMEHTTP01", "GET", sitePath+"/preflight/acme-http01", secret, "", "", 409, "site_not_ready", nil)
	plain("get catalog site", "getSite", "GET", sitePath, "", 200)
	run(h, "other tenant site", "getSite", "GET", sitePath, sqlOtherToken, "", "", 404, "not_found", nil)
	plain("add pending alias", "addSiteAlias", "POST", sitePath+"/domains", `{"domain":"alias.example.test"}`, 201)
	plain("delete pending alias", "deleteSiteAlias", "DELETE", sitePath+"/domains/alias.example.test", "", 204)
	proofPath := sitePath + "/domains/contract.example.test/verification"
	proof := plain("begin synthetic TXT proof", "beginDomainVerification", "POST", proofPath, "", 201)
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(proof.Body.Bytes(), &instruction); err != nil || instruction.Name == "" || instruction.Value == "" {
		t.Fatal("invalid proof instruction")
	}
	lookup.set(instruction.Name+".", instruction.Value)
	proofBody, err := json.Marshal(map[string]string{"value": instruction.Value})
	if err != nil {
		t.Fatal(err)
	}
	plain("complete synthetic TXT proof", "completeDomainVerification", "POST", proofPath+"/complete", string(proofBody), 200)
	plain("read verified ACME snapshot", "preflightSiteACMEHTTP01", "GET", sitePath+"/preflight/acme-http01", "", 200)
	run(h, "other tenant ACME snapshot", "preflightSiteACMEHTTP01", "GET", sitePath+"/preflight/acme-http01", sqlOtherToken, "", "", 404, "not_found", nil)
	s.SiteHostingPaths = &sites.HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	h = s.Handler()
	preview := plain("render verified catalog preview", "previewSiteNginx", "GET", sitePath+"/preview/nginx", "", 200)
	if !strings.Contains(preview.Body.String(), "contract.example.test") {
		t.Fatal("preview omitted verified domain")
	}
	plain("list empty database catalog", "listSiteDatabases", "GET", sitePath+"/databases", "", 200)
	plain("list empty database catalog page", "listSiteDatabases", "GET", sitePath+"/databases?page=1&limit=1", "", 200)
	plain("create database metadata", "createSiteDatabase", "POST", sitePath+"/databases", `{"database_id":"contractdb"}`, 201)
	plain("get database metadata", "getSiteDatabase", "GET", sitePath+"/databases/contractdb", "", 200)
	cookieServer, _, _, token, csrf := setupCookieAPI(t)
	cookie := func(withCSRF bool, origin string) func(*http.Request) {
		return func(req *http.Request) {
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
			if withCSRF {
				req.Header.Set("X-CSRF-Token", csrf)
			}
			if origin != "" {
				req.Header.Set("Origin", origin)
			}
		}
	}
	ch := cookieServer.Handler()
	run(ch, "plaintext cookie denied", "getCurrentPrincipal", "GET", "/api/v1/me", "", "", "", 401, "unauthorized", func(req *http.Request) {
		cookie(false, "")(req)
		req.URL.Scheme = "http"
		req.TLS = nil
	})
	run(ch, "cookie mutation requires CSRF", "logoutSession", "POST", "/api/v1/session/logout", "", "", "", 403, "csrf_failed", cookie(false, "https://panel.test"))
	run(ch, "cookie mutation requires same origin", "logoutSession", "POST", "/api/v1/session/logout", "", "", "", 403, "csrf_failed", cookie(true, "https://other.test"))
	logout := run(ch, "revoke cookie session", "logoutSession", "POST", "/api/v1/session/logout", "", "", "", 204, "", cookie(true, "https://panel.test"))
	if cookies := logout.Result().Cookies(); len(cookies) != 1 || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatal("logout did not expire cookie")
	}
	run(ch, "revoked cookie denied", "getCurrentPrincipal", "GET", "/api/v1/me", "", "", "", 401, "unauthorized", cookie(false, ""))
	for _, operation := range strings.Fields("getHealth getVersion getCurrentPrincipal logoutSession listJobs createJob getJob cancelJob listFiles readFile createFile deleteFile listSites createSite getSite previewSiteNginx preflightSiteACMEHTTP01 addSiteAlias deleteSiteAlias beginDomainVerification completeDomainVerification listSiteDatabases createSiteDatabase getSiteDatabase") {
		if !covered[operation] {
			t.Fatalf("missing success scenario for %s", operation)
		}
	}
	if path := os.Getenv("TONG_OPENAPI_REPORT"); path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal("create private contract report:", err)
		}
		if err := json.NewEncoder(f).Encode(struct {
			Cases []openAPIContractCase `json:"cases"`
		}{cases}); err != nil {
			_ = f.Close()
			t.Fatal("write contract report:", err)
		}
		if err := f.Close(); err != nil {
			t.Fatal("close contract report:", err)
		}
	}
}
