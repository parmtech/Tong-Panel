package api

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

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
