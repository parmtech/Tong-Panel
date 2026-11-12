package api

import (
	"crypto/tls"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

type siteDetailReadRoute struct {
	name string
	path string
}

var siteDetailReadRoutes = []siteDetailReadRoute{
	{name: "site", path: "/api/v1/sites/site-read"},
	{name: "nginx preview", path: "/api/v1/sites/site-read/preview/nginx"},
	{name: "database list", path: "/api/v1/sites/site-read/databases"},
	{name: "database detail", path: "/api/v1/sites/site-read/databases/wp"},
}

func setupSiteDetailRead(t *testing.T) (*Server, *sql.DB, rbac.Principal) {
	t.Helper()
	s, _, db := setupSiteAPI(t)
	s.SiteHostingPaths = &sites.HostingPaths{TenantRoot: "/srv/tong", PHPSocketRoot: "/run/tong/php"}
	if got := createTestSite(t, s.Handler(), secret, "site-read", "read.example.test").Code; got != http.StatusCreated {
		t.Fatalf("create site: status %d", got)
	}
	if _, err := db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE tenant_id='tenant-a' AND site_id='site-read'"); err != nil {
		t.Fatal(err)
	}
	if got := call(s.Handler(), http.MethodPost, "/api/v1/sites/site-read/databases", secret, "", `{"database_id":"wp"}`).Code; got != http.StatusCreated {
		t.Fatalf("create database metadata: status %d", got)
	}
	return s, db, rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
}

func callSiteDetailResource(s *Server, path, token string, cookie bool, beforeRead func()) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie {
		req = httptest.NewRequest(http.MethodGet, "https://panel.test"+path, nil)
		req.TLS = &tls.ConnectionState{}
		req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.protected(func(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
		if beforeRead != nil {
			beforeRead()
		}
		s.siteResource(w, r, p)
	})(w, req)
	return w
}

func TestSiteDetailReadsRecheckCurrentMembership(t *testing.T) {
	changes := []struct {
		name string
		sql  string
	}{
		{name: "membership removed", sql: "DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='u2'"},
		{name: "identity disabled", sql: "UPDATE identities SET disabled=1 WHERE id='u2'"},
		{name: "tenant suspended", sql: "UPDATE tenants SET status='suspended' WHERE id='tenant-a'"},
	}
	for _, route := range siteDetailReadRoutes {
		for _, change := range changes {
			t.Run(route.name+"/"+change.name, func(t *testing.T) {
				s, db, _ := setupSiteDetailRead(t)
				if got := callSiteDetailResource(s, route.path, sqlUserBToken, false, nil).Code; got != http.StatusOK {
					t.Fatalf("authorized control returned %d", got)
				}
				response := callSiteDetailResource(s, route.path, sqlUserBToken, false, func() {
					if _, err := db.Exec(change.sql); err != nil {
						t.Fatal(err)
					}
				})
				if response.Code != http.StatusForbidden {
					t.Fatalf("stale principal returned %d: %s", response.Code, response.Body.String())
				}
				if response.Body.Len() == 0 {
					t.Fatal("denial response missing")
				}
				for _, sensitive := range []string{"read.example.test", "/srv/tong", "site-read", "wp"} {
					if strings.Contains(response.Body.String(), sensitive) {
						t.Fatalf("denial exposed %q: %s", sensitive, response.Body.String())
					}
				}
			})
		}
	}
}

func TestSiteDetailReadsHonorCurrentReadOnlyRole(t *testing.T) {
	for _, route := range siteDetailReadRoutes {
		t.Run(route.name, func(t *testing.T) {
			s, db, principal := setupSiteDetailRead(t)
			principal.UserID = "u2"
			if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u2'"); err != nil {
				t.Fatal(err)
			}
			response := callSiteDetailResource(s, route.path, sqlUserBToken, false, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("current read-only role returned %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestSiteDetailReadsRecheckCookieSession(t *testing.T) {
	for _, route := range siteDetailReadRoutes {
		for _, state := range []string{"revoked", "expired", "idle expired", "membership removed", "identity disabled", "tenant suspended"} {
			t.Run(route.name+"/"+state, func(t *testing.T) {
				s, db, _ := setupSiteDetailRead(t)
				principal := rbac.Principal{TenantID: "tenant-a", UserID: "u2", Role: rbac.Customer}
				store, err := auth.NewSQLiteSessionStore(db)
				if err != nil {
					t.Fatal(err)
				}
				s.Sessions = store
				token, _, _, err := store.Issue(principal, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				if got := callSiteDetailResource(s, route.path, token, true, nil).Code; got != http.StatusOK {
					t.Fatalf("authorized cookie control returned %d", got)
				}
				response := callSiteDetailResource(s, route.path, token, true, func() {
					switch state {
					case "revoked":
						err = store.Revoke(token)
					case "expired":
						_, err = db.Exec("UPDATE sessions SET expires_at=? WHERE identity_id=? AND revoked_at IS NULL", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano), principal.UserID)
					case "idle expired":
						_, err = db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE identity_id=? AND revoked_at IS NULL", principal.UserID)
					case "membership removed":
						_, err = db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='u2'")
					case "identity disabled":
						_, err = db.Exec("UPDATE identities SET disabled=1 WHERE id='u2'")
					case "tenant suspended":
						_, err = db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'")
					}
					if err != nil {
						t.Fatal(err)
					}
				})
				if response.Code != http.StatusUnauthorized {
					t.Fatalf("stale cookie proof returned %d: %s", response.Code, response.Body.String())
				}
				if state == "expired" || state == "idle expired" {
					proof, ok := store.ReadProof(token)
					if !ok {
						t.Fatal("session proof unavailable")
					}
					digest := proof.Digest()
					var revoked bool
					if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || !revoked {
						t.Fatalf("expired session was not durably revoked: revoked=%t err=%v", revoked, err)
					}
				}
			})
		}
	}
}

func TestSiteDetailReadRejectsSessionProofFromDifferentPool(t *testing.T) {
	s, _, _ := setupSiteDetailRead(t)
	principal := rbac.Principal{TenantID: "tenant-a", UserID: "u2", Role: rbac.Customer}
	_, _, otherDB := setupSiteAPI(t)
	otherStore, err := auth.NewSQLiteSessionStore(otherDB)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := otherStore.Issue(principal, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = otherStore
	response := callSiteDetailResource(s, siteDetailReadRoutes[0].path, token, true, nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("separate-pool proof returned %d: %s", response.Code, response.Body.String())
	}
}

func TestSiteDetailReadsDoNotCrossTenant(t *testing.T) {
	s, db, _ := setupSiteDetailRead(t)
	if got := createTestSite(t, s.Handler(), sqlOtherToken, "tenant-b-read", "tenant-b-read.example.test").Code; got != http.StatusCreated {
		t.Fatalf("create tenant-b site: status %d", got)
	}
	if _, err := db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE tenant_id='tenant-b' AND site_id='tenant-b-read'"); err != nil {
		t.Fatal(err)
	}
	if got := call(s.Handler(), http.MethodPost, "/api/v1/sites/tenant-b-read/databases", sqlOtherToken, "", `{"database_id":"tenant-b-db"}`).Code; got != http.StatusCreated {
		t.Fatalf("create tenant-b database metadata: status %d", got)
	}

	for _, route := range siteDetailReadRoutes {
		t.Run(route.name, func(t *testing.T) {
			path := strings.Replace(route.path, "site-read", "tenant-b-read", 1)
			response := callSiteDetailResource(s, path, sqlUserBToken, false, nil)
			if response.Code != http.StatusNotFound {
				t.Fatalf("tenant-a requester read tenant-b resource: status %d: %s", response.Code, response.Body.String())
			}
			for _, sensitive := range []string{"tenant-b-read.example.test", "tenant-b-read", "tenant-b-db", "/srv/tong"} {
				if strings.Contains(response.Body.String(), sensitive) {
					t.Fatalf("cross-tenant denial exposed %q: %s", sensitive, response.Body.String())
				}
			}
		})
	}
}
