package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/parmtech/tong-panel/internal/auth"
)

func TestCookieSafeMethodsDoNotMutateJobsEndpoint(t *testing.T) {
	server, _, db, token, _ := setupCookieAPI(t)
	h := server.Handler()
	jobsAndAudits := func(t *testing.T) [2]int {
		t.Helper()
		var jobCount, auditCount int
		if err := db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobCount); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT count(*) FROM audit_events").Scan(&auditCount); err != nil {
			t.Fatal(err)
		}
		return [2]int{jobCount, auditCount}
	}
	before := jobsAndAudits(t)

	for _, method := range []string{http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "https://panel.test/api/v1/jobs", nil)
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
			if method == http.MethodOptions {
				req.Header.Set("Origin", "https://attacker.test")
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
				req.Header.Set("Access-Control-Request-Headers", "content-type,idempotency-key")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s returned %d, want 405", method, rec.Code)
			}
			if rec.Header().Get("Allow") != "GET, POST" {
				t.Fatalf("%s Allow=%q, want GET, POST", method, rec.Header().Get("Allow"))
			}
			for _, name := range []string{
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Credentials",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Expose-Headers",
				"Set-Cookie",
			} {
				if values := rec.Header().Values(name); len(values) != 0 {
					t.Fatalf("%s unexpectedly returned %s headers: %v", method, name, values)
				}
			}
			if method == http.MethodOptions {
				var body failure
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != "method_not_allowed" {
					t.Fatalf("OPTIONS error envelope code=%q err=%v", body.Error.Code, err)
				}
			}
			if after := jobsAndAudits(t); after != before {
				t.Fatalf("%s changed job/audit counts from %v to %v", method, before, after)
			}
		})
	}
}
