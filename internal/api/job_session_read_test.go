package api

import (
	"context"
	"crypto/tls"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func setupJobSessionRead(t *testing.T) (*Server, *auth.SQLiteSessionStore, *sql.DB, string, jobs.Job) {
	t.Helper()
	s, _, db := setupSiteAPI(t)
	sessions, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = sessions
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.SQLJobs.CreateFor("tenant-a", "u1", jobs.KindNoop, "private-job-read-request-0001")
	if err != nil {
		t.Fatal(err)
	}
	return s, sessions, db, token, job
}

// Run the real authentication middleware, then commit a change before the
// handler reads. This makes the authentication/read boundary deterministic.
func callJobReadAfterAuthentication(s *Server, path, token string, beforeRead func()) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "https://panel.test"+path, nil)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
	rec := httptest.NewRecorder()
	s.protected(func(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
		if beforeRead != nil {
			beforeRead()
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") {
			s.jobDetail(w, r, p)
		} else {
			s.jobListCreate(w, r, p)
		}
	})(rec, req)
	return rec
}

func TestJobReadsRecheckCookieSessionAfterAuthentication(t *testing.T) {
	for _, route := range []string{"list", "page", "detail"} {
		for _, state := range []string{"revoked", "rotated", "absolute expiry", "idle expiry", "malformed expiry", "identity disabled", "tenant suspended"} {
			t.Run(route+"/"+state, func(t *testing.T) {
				s, sessions, db, token, job := setupJobSessionRead(t)
				path := "/api/v1/jobs"
				if route == "page" {
					path += "?limit=1"
				} else if route == "detail" {
					path += "/" + job.ID
				}
				control := cookieCall(s.Handler(), http.MethodGet, path, token, "", "", "")
				if control.Code != http.StatusOK || !strings.Contains(control.Body.String(), job.ID) {
					t.Fatalf("authorized control returned %d: %s", control.Code, control.Body.String())
				}
				response := callJobReadAfterAuthentication(s, path, token, func() {
					var err error
					switch state {
					case "revoked":
						err = sessions.Revoke(token)
					case "rotated":
						_, _, _, err = sessions.Rotate(context.Background(), token)
					case "absolute expiry":
						_, err = db.Exec("UPDATE sessions SET expires_at=? WHERE identity_id='u1' AND revoked_at IS NULL", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano))
					case "idle expiry":
						_, err = db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE identity_id='u1' AND revoked_at IS NULL")
					case "malformed expiry":
						_, err = db.Exec("UPDATE sessions SET expires_at='malformed' WHERE identity_id='u1' AND revoked_at IS NULL")
					case "identity disabled":
						_, err = db.Exec("UPDATE identities SET disabled=1 WHERE id='u1'")
					case "tenant suspended":
						_, err = db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'")
					}
					if err != nil {
						t.Fatal(err)
					}
				})
				assertPageAPIError(t, response, http.StatusUnauthorized, "unauthorized")
				for _, private := range []string{job.ID, job.IdempotencyKey, job.RequestedBy} {
					if strings.Contains(response.Body.String(), private) {
						t.Fatalf("denied read exposed job field %q", private)
					}
				}
				if state == "absolute expiry" || state == "idle expiry" {
					proof, _ := sessions.ReadProof(token)
					digest := proof.Digest()
					var revoked bool
					if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || !revoked {
						t.Fatalf("observed expiry was not durably revoked: %t %v", revoked, err)
					}
				}
			})
		}
	}
}

func TestJobReadsRejectSessionProofFromAnotherPool(t *testing.T) {
	s, _, db, token, job := setupJobSessionRead(t)
	var sequence int
	var name, path string
	if err := db.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &path); err != nil || name != "main" {
		t.Fatalf("database path: %q %v", name, err)
	}
	other, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	// Even the same persisted session in the same file is insufficient when
	// authentication and job reads use different authoritative pools.
	s.Sessions, err = auth.NewSQLiteSessionStore(other)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/api/v1/jobs", "/api/v1/jobs?limit=1", "/api/v1/jobs/" + job.ID} {
		response := cookieCall(s.Handler(), http.MethodGet, route, token, "", "", "")
		assertPageAPIError(t, response, http.StatusServiceUnavailable, "jobs_unavailable")
		if strings.Contains(response.Body.String(), job.ID) || strings.Contains(response.Body.String(), job.IdempotencyKey) {
			t.Fatal("foreign-pool proof returned job data")
		}
	}
}

func TestJobCookieReadsStopAfterAuthenticatedRequestCancellation(t *testing.T) {
	s, _, _, token, job := setupJobSessionRead(t)
	for _, route := range []string{"/api/v1/jobs", "/api/v1/jobs?limit=1", "/api/v1/jobs/" + job.ID} {
		t.Run(route, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest(http.MethodGet, "https://panel.test"+route, nil).WithContext(ctx)
			req.TLS = &tls.ConnectionState{}
			req.AddCookie(&http.Cookie{Name: auth.BrowserCookieName, Value: token})
			response := httptest.NewRecorder()
			authenticated := false
			s.protected(func(w http.ResponseWriter, r *http.Request, p rbac.Principal) {
				authenticated = true
				cancel()
				if strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") {
					s.jobDetail(w, r, p)
				} else {
					s.jobListCreate(w, r, p)
				}
			})(response, req)
			if !authenticated || response.Body.Len() != 0 {
				t.Fatalf("canceled authenticated read wrote a response: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
