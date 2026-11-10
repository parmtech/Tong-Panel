package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func guardedDatabaseRequest(ctx context.Context) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sites/example/databases", strings.NewReader(`{"database_id":"db1"}`)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDatabaseCreationRejectsSeparateSessionPool(t *testing.T) {
	s, _, _ := setupSiteAPI(t)
	if w := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	other, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "separate.db"))
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
		t.Fatal("proof")
	}
	ctx := context.WithValue(context.Background(), sessionProofKey{}, proof)
	w := httptest.NewRecorder()
	s.siteDatabaseCollection(w, guardedDatabaseRequest(ctx), rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}, "example")
	if w.Code != 503 {
		t.Fatalf("separate pool %d %s", w.Code, w.Body.String())
	}
}

func TestDatabaseCreationRechecksAuthority(t *testing.T) {
	for _, update := range []string{"UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'", "UPDATE identities SET disabled=1 WHERE id='u1'", "UPDATE tenants SET status='suspended' WHERE id='tenant-a'", "DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='u2'"} {
		t.Run(update, func(t *testing.T) {
			s, _, db := setupSiteAPI(t)
			if w := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); w.Code != 201 {
				t.Fatal(w.Body.String())
			}
			user := "u1"
			if strings.Contains(update, "u2") {
				user = "u2"
			}
			if _, err := db.Exec(update); err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			s.siteDatabaseCollection(w, guardedDatabaseRequest(context.Background()), rbac.Principal{TenantID: "tenant-a", UserID: user, Role: rbac.Customer}, "example")
			if w.Code != 403 {
				t.Fatalf("revoked %d %s", w.Code, w.Body.String())
			}
			var n int
			if err := db.QueryRow("SELECT count(*) FROM site_databases").Scan(&n); err != nil || n != 0 {
				t.Fatal("denied metadata retained")
			}
		})
	}
}

func TestDatabaseCreationAuditRollbackAndDuplicateRace(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if w := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_database_audit BEFORE INSERT ON audit_events WHEN NEW.action='site.database_metadata_created' BEGIN SELECT RAISE(ABORT,'fail'); END`); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/sites/example/databases"
	if w := call(s.Handler(), http.MethodPost, path, secret, "", `{"database_id":"db1"}`); w.Code != 503 {
		t.Fatalf("audit failure %d %s", w.Code, w.Body.String())
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM site_databases").Scan(&n); err != nil || n != 0 {
		t.Fatal("audit failure retained metadata")
	}
	if _, err := db.Exec("DROP TRIGGER fail_database_audit"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- call(s.Handler(), http.MethodPost, path, secret, "", `{"database_id":"db1"}`).Code
		}()
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[201] != 1 || counts[409] != 1 {
		t.Fatalf("duplicate race: %v", counts)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.database_metadata_created' AND actor_identity_id='u1' AND subject_id='example/db1'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit %d %v", n, err)
	}
}

func TestDatabaseCreationCookieCSRFAndExpiredSession(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if w := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); w.Code != 201 {
		t.Fatal(w.Body.String())
	}
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
	path := "/api/v1/sites/example/databases"
	if w := cookieCall(s.Handler(), http.MethodPost, path, token, "", "https://panel.test", `{"database_id":"db1"}`); w.Code != 403 {
		t.Fatalf("csrf %d", w.Code)
	}
	if w := cookieCall(s.Handler(), http.MethodPost, path, token, csrf, "https://panel.test", `{"database_id":"db1"}`); w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	token, _, _, err = store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := store.ReadProof(token)
	if !ok {
		t.Fatal("proof")
	}
	if err := store.Revoke(token); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), sessionProofKey{}, proof)
	w := httptest.NewRecorder()
	s.siteDatabaseCollection(w, guardedDatabaseRequest(ctx), p, "example")
	if w.Code != 401 {
		t.Fatalf("revoked %d %s", w.Code, w.Body.String())
	}
	token, _, _, err = store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, _ = store.ReadProof(token)
	if _, err := db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE revoked_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = context.WithValue(deadline, sessionProofKey{}, proof)
	w = httptest.NewRecorder()
	s.siteDatabaseCollection(w, guardedDatabaseRequest(ctx), p, "example")
	if w.Code != 401 {
		t.Fatalf("expired %d %s", w.Code, w.Body.String())
	}
}
