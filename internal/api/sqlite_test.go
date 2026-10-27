package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
	"github.com/parmtech/tong-panel/internal/rbac"
)

const sqlUserBToken = "development-token-user-b-11111111111111111111111111"
const sqlOtherToken = "development-token-other-22222222222222222222222222"
const sqlAuditorToken = "development-token-auditor-3333333333333333333333"
const sqlStaleOwnerToken = "development-token-stale-owner-44444444444444444444"

// The SQL mode is experimental and explicitly seeded by test code; the daemon
// must never silently make a production owner or an HTTP password bootstrap.
func setupSQLAPI(t *testing.T) (*Server, *jobs.SQLiteStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "api.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B');
		INSERT INTO identities(id,username) VALUES('u1','alice'),('u2','bob'),('u3','charlie'),('u4','dave');
		INSERT INTO memberships(tenant_id,identity_id,role)
		VALUES('tenant-a','u1','customer'),('tenant-a','u2','customer'),('tenant-b','u3','customer'),('tenant-a','u4','auditor');`)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := auth.NewDevStore(map[string]rbac.Principal{
		secret:          {UserID: "u1", TenantID: "tenant-a", Role: rbac.Customer},
		sqlUserBToken:   {UserID: "u2", TenantID: "tenant-a", Role: rbac.Customer},
		sqlOtherToken:   {UserID: "u3", TenantID: "tenant-b", Role: rbac.Customer},
		sqlAuditorToken: {UserID: "u4", TenantID: "tenant-a", Role: rbac.Auditor},
		sqlStaleOwnerToken: {UserID: "u1", TenantID: "tenant-a", Role: rbac.Owner},
	})
	if err != nil {
		t.Fatal(err)
	}
	q, err := jobs.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Keys: keys, SQLJobs: q}, q, path
}


// assertCreated makes the test failures include the HTTP error schema if needed.
func assertCreated(t *testing.T, status int, body []byte) jobs.Job {
	t.Helper()
	if status != 202 {
		t.Fatalf("SQL enqueue status=%d, body=%s", status, string(body))
	}
	var job jobs.Job
	if err := json.Unmarshal(body, &job); err != nil || job.ID == "" {
		t.Fatalf("invalid SQL enqueue response: %+v, %v", job, err)
	}
	return job
}

func TestDeveloperSQLiteJobsAPIActorScopeIdempotencyAndCancel(t *testing.T) {
	s, _, _ := setupSQLAPI(t)
	h := s.Handler()
	if r := call(h, "GET", "/api/v1/jobs", "", "", ""); r.Code != 401 {
		t.Fatalf("unauthenticated SQL read=%d", r.Code)
	}
	// A token's stale OWNER role must be replaced with the live database role.
	if rec := call(h, "GET", "/api/v1/me", sqlStaleOwnerToken, "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "\"Role\":\"customer\"") {
		t.Fatalf("stale token role was not downgraded: %d %s", rec.Code, rec.Body.String())
	}
	const body = `{"kind":"validation.noop"}`
	createdResp := call(h, "POST", "/api/v1/jobs", secret, "sqlite-key-0001", body)
	a := assertCreated(t, createdResp.Code, createdResp.Body.Bytes())
	if a.RequestedBy != "u1" || a.TenantID != "tenant-a" {
		t.Fatalf("SQL request actor forged: %+v", a)
	}
	retry := call(h, "POST", "/api/v1/jobs", secret, "sqlite-key-0001", body)
	if got := assertCreated(t, retry.Code, retry.Body.Bytes()); got.ID != a.ID {
		t.Fatalf("SQL idempotency changed ID from %s to %s", a.ID, got.ID)
	}
	for _, token := range []string{sqlUserBToken, sqlOtherToken} {
		if r := call(h, "GET", "/api/v1/jobs/"+a.ID, token, "", ""); r.Code != 404 {
			t.Fatalf("cross-identity SQL read leaked status=%d", r.Code)
		}
		if r := call(h, "DELETE", "/api/v1/jobs/"+a.ID, token, "", ""); r.Code != 404 {
			t.Fatalf("cross-identity SQL delete leaked status=%d", r.Code)
		}
		if r := call(h, "GET", "/api/v1/jobs", token, "", ""); r.Code != 200 || strings.Contains(r.Body.String(), a.ID) {
			t.Fatalf("cross-identity SQL list leaked %d: %s", r.Code, r.Body.String())
		}
	}
	if r := call(h, "POST", "/api/v1/jobs", sqlUserBToken, "sqlite-key-0001", body); r.Code != 409 {
		t.Fatalf("same-tenant reused idempotency key should conflict: %d", r.Code)
	}
	bResp := call(h, "POST", "/api/v1/jobs", sqlUserBToken, "sqlite-key-0002", body)
	b := assertCreated(t, bResp.Code, bResp.Body.Bytes())
	if r := call(h, "GET", "/api/v1/jobs", secret, "", ""); r.Code != 200 || !strings.Contains(r.Body.String(), a.ID) || strings.Contains(r.Body.String(), b.ID) {
		t.Fatalf("SQL own-job list mismatch %d: %s", r.Code, r.Body.String())
	}
	if r := call(h, "DELETE", "/api/v1/jobs/"+a.ID, sqlAuditorToken, "", ""); r.Code != 403 {
		t.Fatalf("auditor was allowed to cancel: %d", r.Code)
	}
	for i := 0; i < 2; i++ {
		r := call(h, "DELETE", "/api/v1/jobs/"+a.ID, secret, "", "")
		if r.Code != 200 || !strings.Contains(r.Body.String(), "cancelled") {
			t.Fatalf("cancel attempt %d: %d %s", i, r.Code, r.Body.String())
		}
	}
	if r := call(h, "GET", "/api/v1/jobs/"+a.ID, secret, "", ""); r.Code != 200 || !strings.Contains(r.Body.String(), "cancelled") {
		t.Fatalf("cancel not persisted: %d %s", r.Code, r.Body.String())
	}
	if r := call(h, "POST", "/api/v1/jobs", secret, "sqlite-key-0003", `{"kind":"system.exec"}`); r.Code != 400 {
		t.Fatalf("unsafe host command accepted: %d", r.Code)
	}
}

func TestDeveloperSQLiteBackendFailsClosedForMissingOrDisabledMembership(t *testing.T) {
	s, _, path := setupSQLAPI(t)
	h := s.Handler()
	// Queues must never bootstrap an absent identity merely because it supplied
	// a syntactically valid developer token.
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	const body = `{"kind":"validation.noop"}`
	if r := call(h, "POST", "/api/v1/jobs", secret, "sqlite-missing-01", body); r.Code != 401 {
		t.Fatalf("missing membership accepted: %d %s", r.Code, r.Body.String())
	}
	if r := call(h, "GET", "/api/v1/jobs", secret, "", ""); r.Code != 401 {
		t.Fatalf("missing membership could list SQL queue: %d", r.Code)
	}
	if _, err := db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','u1','customer')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if r := call(h, "GET", "/api/v1/jobs", secret, "", ""); r.Code != 401 {
		t.Fatalf("suspended tenant read accepted %d", r.Code)
	}
	if _, err := db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='u1'"); err != nil {
		t.Fatal(err)
	}
	if r := call(h, "GET", "/api/v1/jobs", secret, "", ""); r.Code != 401 {
		t.Fatalf("disabled identity read accepted %d", r.Code)
	}
}

func TestDeveloperSQLiteJobsAPIRejectsCancellingLeasedJob(t *testing.T) {
	s, q, _ := setupSQLAPI(t)
	h := s.Handler()
	r := call(h, "POST", "/api/v1/jobs", secret, "sqlite-key-lease-01", `{"kind":"validation.noop"}`)
	job := assertCreated(t, r.Code, r.Body.Bytes())
	claim, err := q.ClaimNext("worker-test", time.Minute)
	if err != nil || claim.Job.ID != job.ID {
		t.Fatalf("lease failed: claim=%+v err=%v", claim, err)
	}
	if rr := call(h, "DELETE", "/api/v1/jobs/"+job.ID, secret, "", ""); rr.Code != 409 {
		t.Fatalf("leased job erroneously reported cancelled: %d %s", rr.Code, rr.Body.String())
	}
}
