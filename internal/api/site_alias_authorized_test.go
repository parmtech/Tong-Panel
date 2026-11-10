package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestAliasMutationRechecksAuthorityAndAudits(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	add := func(ctx context.Context) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/sites/example/domains", strings.NewReader(`{"domain":"ALIAS.example.test"}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		s.siteDomainAdd(w, req, p, "example")
		return w
	}
	remove := func(ctx context.Context) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.siteDomainRemove(w, httptest.NewRequest("DELETE", "/api/v1/sites/example/domains/alias.example.test", nil).WithContext(ctx), p, "example", "alias.example.test")
		return w
	}
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	if w := add(context.Background()); w.Code != 403 {
		t.Fatalf("stale role add: %d %s", w.Code, w.Body.String())
	}
	if _, err := db.Exec("UPDATE memberships SET role='customer' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	if w := add(context.Background()); w.Code != 201 {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.alias_added' AND subject_id='example/alias.example.test' AND actor_identity_id='u1'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit add %d %v", n, err)
	}
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	if w := remove(context.Background()); w.Code != 403 {
		t.Fatalf("stale role remove: %d", w.Code)
	}
	if _, err := db.Exec("UPDATE memberships SET role='customer' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	if w := remove(context.Background()); w.Code != 204 {
		t.Fatalf("remove: %d %s", w.Code, w.Body.String())
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.alias_removed' AND subject_id='example/alias.example.test' AND actor_identity_id='u1'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit remove %d %v", n, err)
	}
}

func TestAliasMutationAuditFailureAndIgnoredDeleteRollback(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_alias BEFORE INSERT ON audit_events WHEN NEW.action='site.alias_added' BEGIN SELECT RAISE(ABORT,'fail'); END`); err != nil {
		t.Fatal(err)
	}
	w := call(s.Handler(), http.MethodPost, "/api/v1/sites/example/domains", secret, "", `{"domain":"alias.example.test"}`)
	if w.Code != 503 {
		t.Fatalf("audit failure %d", w.Code)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM site_domains WHERE domain='alias.example.test'").Scan(&n); err != nil || n != 0 {
		t.Fatal("failed audit retained alias")
	}
	if _, err := db.Exec("DROP TRIGGER fail_alias"); err != nil {
		t.Fatal(err)
	}
	if w := call(s.Handler(), http.MethodPost, "/api/v1/sites/example/domains", secret, "", `{"domain":"alias.example.test"}`); w.Code != 201 {
		t.Fatalf("add %d %s", w.Code, w.Body.String())
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_remove BEFORE INSERT ON audit_events WHEN NEW.action='site.alias_removed' BEGIN SELECT RAISE(ABORT,'fail'); END`); err != nil {
		t.Fatal(err)
	}
	if w := call(s.Handler(), http.MethodDelete, "/api/v1/sites/example/domains/alias.example.test", secret, "", ""); w.Code != 503 {
		t.Fatalf("remove audit failure %d", w.Code)
	}
	if err := db.QueryRow("SELECT count(*) FROM site_domains WHERE domain='alias.example.test'").Scan(&n); err != nil || n != 1 {
		t.Fatal("failed audit removed alias")
	}
	if _, err := db.Exec("DROP TRIGGER fail_remove"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER ignore_alias BEFORE DELETE ON site_domains WHEN OLD.kind='alias' BEGIN SELECT RAISE(IGNORE); END`); err != nil {
		t.Fatal(err)
	}
	if w := call(s.Handler(), http.MethodDelete, "/api/v1/sites/example/domains/alias.example.test", secret, "", ""); w.Code != 503 {
		t.Fatalf("ignored delete %d", w.Code)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.alias_removed'").Scan(&n); err != nil || n != 0 {
		t.Fatal("false removal audit")
	}
}

func TestAliasRealCookieAndCSRF(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = store
	token, csrf, _, err := store.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/sites/example/domains"
	if w := cookieCall(s.Handler(), http.MethodPost, path, token, "", "https://panel.test", `{"domain":"alias.example.test"}`); w.Code != 403 {
		t.Fatalf("missing csrf %d", w.Code)
	}
	if w := cookieCall(s.Handler(), http.MethodPost, path, token, csrf, "https://panel.test", `{"domain":"alias.example.test"}`); w.Code != 201 {
		t.Fatalf("cookie add %d %s", w.Code, w.Body.String())
	}
	if w := cookieCall(s.Handler(), http.MethodDelete, path+"/alias.example.test", token, csrf, "https://panel.test", ""); w.Code != 204 {
		t.Fatalf("cookie remove %d %s", w.Code, w.Body.String())
	}
}

func TestAliasDeniedForDisabledIdentityAndSuspendedTenant(t *testing.T) {
	for _, update := range []string{"UPDATE identities SET disabled=1 WHERE id='u1'", "UPDATE tenants SET status='suspended' WHERE id='tenant-a'"} {
		t.Run(update, func(t *testing.T) {
			s, _, db := setupSiteAPI(t)
			if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
				t.Fatal(r.Body.String())
			}
			if _, err := db.Exec(update); err != nil {
				t.Fatal(err)
			}
			p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
			req := httptest.NewRequest("POST", "/api/v1/sites/example/domains", strings.NewReader(`{"domain":"alias.example.test"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.siteDomainAdd(w, req, p, "example")
			if w.Code != 403 {
				t.Fatalf("revoked authority %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAliasDeniedAfterMembershipRemoval(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	if _, err := db.Exec("INSERT INTO identities(id,username) VALUES('alias-user','alias-user'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','alias-user','customer')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='alias-user'"); err != nil {
		t.Fatal(err)
	}
	p := rbac.Principal{TenantID: "tenant-a", UserID: "alias-user", Role: rbac.Customer}
	req := httptest.NewRequest("POST", "/api/v1/sites/example/domains", strings.NewReader(`{"domain":"alias.example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.siteDomainAdd(w, req, p, "example")
	if w.Code != 403 {
		t.Fatalf("removed membership %d %s", w.Code, w.Body.String())
	}
}

func TestAliasMutationRevokedAndExpiredSession(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	token, _, _, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := store.ReadProof(token)
	if !ok {
		t.Fatal("missing proof")
	}
	if err := store.Revoke(token); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/sites/example/domains", strings.NewReader(`{"domain":"alias.example.test"}`)).WithContext(context.WithValue(context.Background(), sessionProofKey{}, proof))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.siteDomainAdd(w, req, p, "example")
	if w.Code != 401 {
		t.Fatalf("revoked %d %s", w.Code, w.Body.String())
	}
	token, _, _, err = store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, _ = store.ReadProof(token)
	if _, err = db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE revoked_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req = httptest.NewRequest("POST", "/api/v1/sites/example/domains", strings.NewReader(`{"domain":"alias.example.test"}`)).WithContext(context.WithValue(ctx, sessionProofKey{}, proof))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.siteDomainAdd(w, req, p, "example")
	if w.Code != 401 {
		t.Fatalf("expired %d %s", w.Code, w.Body.String())
	}
}
