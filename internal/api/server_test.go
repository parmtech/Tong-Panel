package api

import (
	"bytes"
	"encoding/json"
	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const secret = "development-token-for-tongpanel-only-123456789"

func setup(t *testing.T) *Server {
	t.Helper()
	keys, err := auth.NewDevStore(map[string]rbac.Principal{secret: {UserID: "u1", TenantID: "tenant-A", Role: rbac.Customer}})
	if err != nil {
		t.Fatal(err)
	}
	q, err := jobs.Open(filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Keys: keys, Jobs: q}
}
func call(h http.Handler, method, path, token, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
func TestPublicAndProtectedEndpoints(t *testing.T) {
	h := setup(t).Handler()
	for _, path := range []string{"/healthz", "/api/v1/version"} {
		r := call(h, "GET", path, "", "", "")
		if r.Code != 200 {
			t.Fatalf("%s returned %d", path, r.Code)
		}
	}
	for _, path := range []string{"/api/v1/me", "/api/v1/jobs"} {
		for _, token := range []string{"", "incorrect-token-value-is-not-the-original-token"} {
			r := call(h, "GET", path, token, "", "")
			if r.Code != 401 {
				t.Fatalf("%s leaked %d", path, r.Code)
			}
		}
	}
	if r := call(h, "GET", "/api/v1/me", secret, "", ""); r.Code != 200 {
		t.Fatalf("auth failed %d", r.Code)
	}
	if r := call(h, "POST", "/api/v1/jobs", secret, "", "{\"kind\":\"validation.noop\"}"); r.Code != 400 {
		t.Fatal("no idempotency")
	}
	if r := call(h, "POST", "/api/v1/jobs", secret, "req-001122", "{\"kind\":\"system.exec\"}"); r.Code != 400 {
		t.Fatal("arbitrary command accepted")
	}
	if r := call(h, "POST", "/api/v1/jobs", secret, "req-001122", "{\"kind\":\"validation.noop\",\"command\":\"rm -rf /\"}"); r.Code != 400 {
		t.Fatal("extra field accepted")
	}
	r := call(h, "POST", "/api/v1/jobs", secret, "req-001122", "{\"kind\":\"validation.noop\"}")
	if r.Code != 202 {
		t.Fatalf("status %d: %s", r.Code, r.Body.String())
	}
	var created jobs.Job
	if err := json.Unmarshal(r.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "tenant-A" {
		t.Fatal("tenant forged")
	}
	if rr := call(h, "GET", "/api/v1/jobs/"+created.ID, secret, "", ""); rr.Code != 200 {
		t.Fatal("job inaccessible")
	}
	if rr := call(h, "GET", "/api/v1/jobs/../../etc/passwd", secret, "", ""); rr.Code == 200 {
		t.Fatal("path traversal")
	}
	rr := call(h, "GET", "/api/v1/jobs", secret, "", "")
	if !bytes.Contains(rr.Body.Bytes(), []byte(created.ID)) {
		t.Fatal("list missing job")
	}
}
func TestDenyWhenNoKeys(t *testing.T) {
	s := &Server{}
	for _, p := range []string{"/api/v1/me", "/api/v1/jobs"} {
		r := call(s.Handler(), "GET", p, secret, "", "")
		if r.Code != 503 {
			t.Fatalf("accepted without auth config: %d", r.Code)
		}
	}
}
