package api

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func setupCookieAPI(t *testing.T) (*Server, *auth.SQLiteSessionStore, *sql.DB, string, string) {
	t.Helper()
	s, _, path := setupSQLAPI(t)
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	token, csrf, _, err := store.Issue(
		rbac.Principal{UserID: "u1", TenantID: "tenant-a", Role: rbac.Customer},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = store
	return s, store, db, token, csrf
}

func cookieCall(h http.Handler, method, path, token, csrf, origin, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://panel.test"+path, strings.NewReader(body))
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Idempotency-Key", "session-cookie-job-0001")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSQLiteCookieSessionReadAndMutatingRequestGuards(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	h := s.Handler()
	if rec := cookieCall(h, "GET", "/api/v1/me", token, "", "", ""); rec.Code != 200 {
		t.Fatalf("HTTPS session identity refused %d: %s", rec.Code, rec.Body.String())
	}
	const body = "{\"kind\":\"validation.noop\"}"
	for _, tc := range []struct {
		name, csrf, origin string
	}{
		{"no csrf", "", "https://panel.test"},
		{"wrong csrf", strings.Repeat("a", 64), "https://panel.test"},
		{"no origin", csrf, ""},
		{"evil origin", csrf, "https://evil.test"},
		{"http origin", csrf, "http://panel.test"},
		{"path origin", csrf, "https://panel.test/path"},
		{"userinfo origin", csrf, "https://panel.test@evil.test"},
		{"port mismatch", csrf, "https://panel.test:9443"},
		{"multi-origin", csrf, "https://panel.test https://evil.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := cookieCall(h, "POST", "/api/v1/jobs", token, tc.csrf, tc.origin, body)
			if rec.Code != 403 {
				t.Fatalf("CSRF bypass, status %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
	rec := cookieCall(h, "POST", "/api/v1/jobs", token, csrf, "https://panel.test", body)
	if rec.Code != 202 || !strings.Contains(rec.Body.String(), "queued") {
		t.Fatalf("same-origin CSRF-protected enqueue failed %d: %s", rec.Code, rec.Body.String())
	}
	var created struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.ID
	if len(id) != 32 {
		t.Fatalf("invalid job ID %q", id)
	}
	for _, tc := range []struct{ csrf, origin string }{{"", "https://panel.test"}, {csrf, "https://evil.test"}} {
		rec = cookieCall(h, "DELETE", "/api/v1/jobs/"+id, token, tc.csrf, tc.origin, "")
		if rec.Code != 403 {
			t.Fatalf("unguarded cancellation %d: %s", rec.Code, rec.Body.String())
		}
	}
	rec = cookieCall(h, "DELETE", "/api/v1/jobs/"+id, token, csrf, "https://panel.test", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "cancelled") {
		t.Fatalf("guarded cancellation failed %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCookieRequiresActualTLSRejectsAmbiguousCredentials(t *testing.T) {
	s, _, _, token, _ := setupCookieAPI(t)
	h := s.Handler()
	for _, tc := range []struct {
		name       string
		manipulate func(*http.Request)
	}{
		{"plaintext with fake proxy TLS", func(req *http.Request) {
			req.TLS = nil
			req.Header.Set("X-Forwarded-Proto", "https")
		}},
		{"duplicate cookie", func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
		}},
		{"cookie and developer bearer", func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+secret)
		}},
		{"invalid cookie with developer bearer", func(req *http.Request) {
			req.Header.Set("Cookie", auth.BrowserCookieName+"=invalid")
			req.Header.Set("Authorization", "Bearer "+secret)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://panel.test/api/v1/me", nil)
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
			tc.manipulate(req)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 401 {
				t.Fatalf("unsafe cookie used, response %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
	if rec := call(h, "GET", "/api/v1/me", secret, "", ""); rec.Code != 200 {
		t.Fatalf("development bearer unexpectedly disabled %d", rec.Code)
	}
}

func TestCookieMutationRejectsDuplicateOriginHeaders(t *testing.T) {
	s, _, _, token, csrf := setupCookieAPI(t)
	req := httptest.NewRequest("POST", "https://panel.test/api/v1/jobs", strings.NewReader(`{"kind":"validation.noop"}`))
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Add("Origin", "https://panel.test")
	req.Header.Add("Origin", "https://evil.test")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "duplicate-origin-negative-test")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("duplicate Origin headers accepted: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPSCookieLogoutRevokesAuditsAndExpiresCookie(t *testing.T) {
	s, sessions, db, token, csrf := setupCookieAPI(t)
	h := s.Handler()
	rec := cookieCall(h, http.MethodPost, "/api/v1/session/logout", token, csrf, "https://panel.test", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("valid logout returned %d: %s", rec.Code, rec.Body.String())
	}
	if _, ok := sessions.Check(token); ok {
		t.Fatal("logged-out browser session remains active")
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("logout returned %d cookies, want one expiry cookie", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != auth.BrowserCookieName || cookie.Value != "" || cookie.Path != "/" ||
		cookie.Domain != "" || !cookie.Secure || !cookie.HttpOnly ||
		cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != -1 {
		t.Fatalf("logout weakened the host-only secure cookie: %#v", cookie)
	}
	var auditCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events WHERE action='auth.logout' AND actor_identity_id='u1'").Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("logout audit count=%d, want one", auditCount)
	}
	if got := cookieCall(h, http.MethodGet, "/api/v1/me", token, "", "", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("revoked cookie still authenticated: %d %s", got.Code, got.Body.String())
	}
}

func TestHTTPSCookieLogoutRejectsBadOriginCSRFAndBearerFallback(t *testing.T) {
	s, sessions, db, token, csrf := setupCookieAPI(t)
	h := s.Handler()
	for _, tc := range []struct {
		name, token, csrf, origin string
		mutate                    func(*http.Request)
	}{
		{name: "wrong csrf", token: token, csrf: strings.Repeat("a", 64), origin: "https://panel.test"},
		{name: "wrong origin", token: token, csrf: csrf, origin: "https://evil.test"},
		{name: "duplicate cookie", token: token, csrf: csrf, origin: "https://panel.test", mutate: func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/session/logout", nil)
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: tc.token})
			req.Header.Set("X-CSRF-Token", tc.csrf)
			req.Header.Set("Origin", tc.origin)
			if tc.mutate != nil {
				tc.mutate(req)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
				t.Fatalf("unsafe logout returned %d: %s", rec.Code, rec.Body.String())
			}
			if _, ok := sessions.Check(token); !ok {
				t.Fatal("rejected logout revoked the session")
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("rejected logout expired the browser cookie")
			}
		})
	}
	devReq := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/session/logout", nil)
	devReq.TLS = &tls.ConnectionState{}
	devReq.Header.Set("Authorization", "Bearer "+secret)
	devRec := httptest.NewRecorder()
	h.ServeHTTP(devRec, devReq)
	if devRec.Code != http.StatusUnauthorized {
		t.Fatalf("developer bearer was accepted for session logout: %d", devRec.Code)
	}
	var auditCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events WHERE action='auth.logout'").Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 0 {
		t.Fatalf("rejected logout created %d audit records", auditCount)
	}
}

func TestHTTPSCookieLogoutHasSingleWinnerWhenRaced(t *testing.T) {
	s, _, db, token, csrf := setupCookieAPI(t)
	h := s.Handler()
	start := make(chan struct{})
	codes := make(chan int, 2)
	for range 2 {
		go func() {
			<-start
			codes <- cookieCall(h, http.MethodPost, "/api/v1/session/logout", token, csrf, "https://panel.test", "").Code
		}()
	}
	close(start)
	a, b := <-codes, <-codes
	denied := func(code int) bool {
		return code == http.StatusUnauthorized || code == http.StatusForbidden
	}
	if !((a == http.StatusNoContent && denied(b)) || (b == http.StatusNoContent && denied(a))) {
		t.Fatalf("raced logout statuses=(%d,%d), want one success and one denied request", a, b)
	}
	var auditCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events WHERE action='auth.logout'").Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("raced logout produced %d audit rows, want one", auditCount)
	}
}

func TestConcurrentCookieSessionReadsRemainAuthorized(t *testing.T) {
	s, _, _, token, _ := setupCookieAPI(t)
	h := s.Handler()
	const requests = 24
	start := make(chan struct{})
	results := make(chan int, requests)
	for i := 0; i < requests; i++ {
		go func() {
			<-start
			req := httptest.NewRequest("GET", "https://panel.test/api/v1/me", nil)
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}
	close(start)
	for i := 0; i < requests; i++ {
		if code := <-results; code != http.StatusOK {
			t.Fatalf("concurrent authenticated read returned %d", code)
		}
	}
}

func TestSQLiteCookieSessionRevocationAndCurrentRole(t *testing.T) {
	s, sessions, db, token, csrf := setupCookieAPI(t)
	h := s.Handler()
	rec := cookieCall(h, "POST", "/api/v1/jobs", token, csrf, "https://panel.test", "{\"kind\":\"validation.noop\"}")
	if rec.Code != 202 {
		t.Fatalf("session create failed %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	rec = cookieCall(h, "GET", "/api/v1/me", token, "", "", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "auditor") {
		t.Fatalf("role downgrade missing: %d %s", rec.Code, rec.Body.String())
	}
	rec = cookieCall(h, "POST", "/api/v1/jobs", token, csrf, "https://panel.test", "{\"kind\":\"validation.noop\"}")
	if rec.Code != 403 {
		t.Fatalf("auditor retained mutation rights: %d %s", rec.Code, rec.Body.String())
	}
	if err := sessions.Revoke(token); err != nil {
		t.Fatal(err)
	}
	rec = cookieCall(h, "GET", "/api/v1/me", token, "", "", "")
	if rec.Code != 401 {
		t.Fatalf("revoked session was accepted: %d", rec.Code)
	}
}
