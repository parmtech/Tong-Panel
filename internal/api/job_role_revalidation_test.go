package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestJobCreateRechecksRoleAfterHTTPPrincipalWasAuthenticated(t *testing.T) {
	server, queue, path := setupSQLAPI(t)
	role, err := queue.RoleFor("tenant-a", "u1")
	if err != nil {
		t.Fatal(err)
	}
	// Model the real post-authentication boundary without a timing-dependent
	// goroutine race or a production-only test hook: retain the principal the
	// middleware just authorised, then commit revocation through another handle.
	authenticated := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Role(role)}
	if !authenticated.Can("jobs:create", "tenant-a") {
		t.Fatal("fixture principal could not originally enqueue")
	}
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'"); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(`{"kind":"validation.noop"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "withdrawn-job-authority-0001")
	response := httptest.NewRecorder()
	server.jobListCreate(response, request, authenticated)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"forbidden"`) {
		t.Fatalf("withdrawn current role returned %d: %s", response.Code, response.Body.String())
	}
	for _, query := range []string{"SELECT count(*) FROM jobs", "SELECT count(*) FROM audit_events WHERE action='job.created'"} {
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("denied post-auth request mutated jobs/audit: count=%d err=%v", count, err)
		}
	}
}
