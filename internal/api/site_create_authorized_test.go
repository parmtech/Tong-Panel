package api

import (
	"context"
	"crypto/tls"
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

const guardedSiteBody = `{"site_id":"guarded","primary_domain":"guarded.example.test","aliases":["z.example.test","a.example.test"]}`

func siteCreateRequest(ctx context.Context) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(guardedSiteBody)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestSiteCreateRechecksCurrentMembershipAndAuditsAtomically(t *testing.T) {
	for _, tc := range []struct{ name, revoke string }{
		{"role downgraded", "UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"},
		{"identity disabled", "UPDATE identities SET disabled=1 WHERE id='u1'"},
		{"tenant suspended", "UPDATE tenants SET status='suspended' WHERE id='tenant-a'"},
		{"membership removed", "DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='u1'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, db := setupSiteAPI(t)
			p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
			if _, err := db.Exec(tc.revoke); err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			s.siteCollection(w, siteCreateRequest(context.Background()), p)
			if w.Code != 403 {
				t.Fatalf("stale principal status %d: %s", w.Code, w.Body.String())
			}
			for _, table := range []string{"sites", "site_domains", "audit_events"} {
				var count int
				if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
					t.Fatalf("%s mutated: %d %v", table, count, err)
				}
			}
		})
	}
	s, _, db := setupSiteAPI(t)
	w := httptest.NewRecorder()
	s.siteCollection(w, siteCreateRequest(context.Background()), rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
	if w.Code != 201 || strings.Index(w.Body.String(), "a.example.test") > strings.Index(w.Body.String(), "z.example.test") {
		t.Fatalf("created response: %d %s", w.Code, w.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM audit_events WHERE tenant_id='tenant-a' AND actor_identity_id='u1' AND actor_kind='user' AND action='site.created' AND subject_id='guarded'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit: %d %v", count, err)
	}
}

func TestSiteCreateAuditFailureRollsBackSiteAndDomains(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if _, err := db.Exec(`CREATE TRIGGER reject_site_audit BEFORE INSERT ON audit_events WHEN NEW.action='site.created' BEGIN SELECT RAISE(ABORT,'audit denied'); END`); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.siteCollection(w, siteCreateRequest(context.Background()), rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
	if w.Code != 503 {
		t.Fatalf("audit failure: %d %s", w.Code, w.Body.String())
	}
	for _, table := range []string{"sites", "site_domains", "audit_events"} {
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s retained %d %v", table, n, err)
		}
	}
}

func TestSiteCreateSessionRecheckAndExpiredConnectionRelease(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = store
	p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	token, csrf, _, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://panel.test/api/v1/sites", strings.NewReader(guardedSiteBody))
	req.TLS = &tls.ConnectionState{}
	requestCookie := &http.Cookie{Name: auth.BrowserCookieName, Value: token}
	req.AddCookie(requestCookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://panel.test")
	req.Header.Set("X-CSRF-Token", csrf)
	principal, code, _ := s.cookiePrincipal(req, requestCookie)
	if code != 0 {
		t.Fatal(code)
	}
	proof, ok := store.ReadProof(token)
	if !ok {
		t.Fatal("proof missing")
	}
	req = req.WithContext(context.WithValue(req.Context(), sessionProofKey{}, proof))
	if err := store.Revoke(token); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.siteCollection(w, req, principal)
	if w.Code != 401 {
		t.Fatalf("revoked session: %d %s", w.Code, w.Body.String())
	}
	token, _, _, err = store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, _ = store.ReadProof(token)
	if _, err := db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE revoked_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req = siteCreateRequest(context.WithValue(ctx, sessionProofKey{}, proof))
	w = httptest.NewRecorder()
	s.siteCollection(w, req, principal)
	if w.Code != 401 {
		t.Fatalf("expired session (possible connection deadlock): %d %s", w.Code, w.Body.String())
	}
	var revoked int
	if err := db.QueryRow("SELECT count(*) FROM sessions WHERE revoked_at IS NOT NULL").Scan(&revoked); err != nil || revoked != 2 {
		t.Fatalf("expired session revocation: %d %v", revoked, err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sites").Scan(&n); err != nil || n != 0 {
		t.Fatalf("expired created site: %d %v", n, err)
	}
}

func TestSiteCreateRejectsSessionProofFromDifferentPool(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	other, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "other.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	store, err := auth.NewSQLiteSessionStore(other)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := store.ReadProof(strings.Repeat("ab", 32))
	if !ok {
		t.Fatal("missing proof")
	}
	w := httptest.NewRecorder()
	req := siteCreateRequest(context.WithValue(context.Background(), sessionProofKey{}, proof))
	s.siteCollection(w, req, rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
	if w.Code != 503 {
		t.Fatalf("mismatched pool: %d %s", w.Code, w.Body.String())
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sites").Scan(&n); err != nil || n != 0 {
		t.Fatalf("created: %d %v", n, err)
	}
}

func TestSiteCreateRealCookieAndCSRF(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = store
	p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	token, csrf, _, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	denied := cookieCall(s.Handler(), http.MethodPost, "/api/v1/sites", token, "", "https://panel.test", guardedSiteBody)
	if denied.Code != 403 {
		t.Fatalf("missing CSRF: %d %s", denied.Code, denied.Body.String())
	}
	created := cookieCall(s.Handler(), http.MethodPost, "/api/v1/sites", token, csrf, "https://panel.test", guardedSiteBody)
	if created.Code != 201 {
		t.Fatalf("valid cookie: %d %s", created.Code, created.Body.String())
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.created' AND actor_identity_id='u1'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit: %d %v", count, err)
	}
}

func TestSiteCreateWaitsForCommittedRoleChange(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	var sequence int
	var name, path string
	if err := db.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &path); err != nil {
		t.Fatal(err)
	}
	other, err := database.Open(context.Background(), filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	conn, err := other.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	if _, err = conn.ExecContext(context.Background(), "UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		w := httptest.NewRecorder()
		s.siteCollection(w, siteCreateRequest(ctx), rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer})
		result <- w
	}()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for db.Stats().InUse == 0 {
		select {
		case w := <-result:
			t.Fatalf("create exited before acquiring connection: %d %s", w.Code, w.Body.String())
		case <-ctx.Done():
			t.Fatal("site request did not acquire its connection")
		case <-ticker.C:
		}
	}
	select {
	case w := <-result:
		t.Fatalf("create bypassed held writer: %d %s", w.Code, w.Body.String())
	default:
	}
	if _, err = conn.ExecContext(context.Background(), "COMMIT"); err != nil {
		t.Fatal(err)
	}
	select {
	case w := <-result:
		if w.Code != 403 {
			t.Fatalf("committed downgrade: %d %s", w.Code, w.Body.String())
		}
	case <-ctx.Done():
		t.Fatal("site request did not finish")
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sites").Scan(&n); err != nil || n != 0 {
		t.Fatalf("unexpected site: %d %v", n, err)
	}
}
