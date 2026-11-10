package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/jobs"
)

type jobsPageFixture struct {
	Jobs []jobs.Job `json:"jobs"`
	Next string     `json:"next_after"`
}

func decodeJobsPage(t *testing.T, rec *httptest.ResponseRecorder) jobsPageFixture {
	t.Helper()
	if rec.Code != 200 {
		t.Fatalf("paged GET returned %d", rec.Code)
	}
	var page jobsPageFixture
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil || len(fields) != 2 || fields["next_after"] == nil || page.Jobs == nil {
		t.Fatal("page did not return jobs array and next_after")
	}
	return page
}

func assertPageAPIError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if rec.Code != status || json.Unmarshal(rec.Body.Bytes(), &response) != nil || response.Error.Code != code {
		t.Fatalf("expected status %d and code %s, got status %d", status, code, rec.Code)
	}
}

func TestJobsPageSQLScopesAndCursorTraversal(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if _, err := db.Exec("UPDATE memberships SET role='owner' WHERE identity_id='u1'; UPDATE memberships SET role='customer' WHERE identity_id='u4'"); err != nil {
		t.Fatal(err)
	}
	var expected []string
	for index := range 6 {
		job, err := s.SQLJobs.CreateFor("tenant-a", "u1", jobs.KindNoop, fmt.Sprintf("page-owner-%08d", index))
		if err != nil {
			t.Fatal(err)
		}
		expected = append(expected, job.ID)
	}
	other, err := s.SQLJobs.CreateFor("tenant-a", "u2", jobs.KindNoop, "page-other-user-0001")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.SQLJobs.CreateFor("tenant-b", "u3", jobs.KindNoop, "page-other-tenant-0001")
	if err != nil {
		t.Fatal(err)
	}
	auditor, err := s.SQLJobs.CreateFor("tenant-a", "u4", jobs.KindNoop, "page-auditor-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE memberships SET role='auditor' WHERE identity_id='u4'"); err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	sort.Strings(expected)
	var seen []string
	after := ""
	for turn := range 4 {
		path := "/api/v1/jobs?limit=2"
		if after != "" {
			path += "&after=" + after
		}
		page := decodeJobsPage(t, call(h, "GET", path, secret, "", ""))
		if len(page.Jobs) > 2 {
			t.Fatal("page exceeded requested bound")
		}
		for _, job := range page.Jobs {
			if job.TenantID != "tenant-a" || job.RequestedBy != "u1" {
				t.Fatal("owner page exposed another actor")
			}
			seen = append(seen, job.ID)
		}
		if turn == 0 && page.Next == "" {
			t.Fatal("first page did not advertise continuation")
		}
		if page.Next == "" {
			break
		}
		if len(page.Jobs) == 0 || page.Next != page.Jobs[len(page.Jobs)-1].ID || page.Next <= after {
			t.Fatal("invalid page continuation")
		}
		after = page.Next
	}
	if strings.Join(seen, ",") != strings.Join(expected, ",") {
		t.Fatal("cursor traversal skipped or repeated owned jobs")
	}
	for _, actor := range []struct{ token, id string }{{sqlUserBToken, other.ID}, {sqlOtherToken, foreign.ID}, {sqlAuditorToken, auditor.ID}} {
		page := decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?limit=1", actor.token, "", ""))
		if len(page.Jobs) != 1 || page.Jobs[0].ID != actor.id || page.Next != "" {
			t.Fatal("requester/auditor page scope mismatch")
		}
	}
	// A cursor is only a position, never authority to read the actor that owns it.
	page := decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?limit=100&after="+other.ID, secret, "", ""))
	for _, job := range page.Jobs {
		if job.RequestedBy != "u1" || job.ID <= other.ID {
			t.Fatal("cross-user cursor changed authorization")
		}
	}
	page = decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?after="+strings.Repeat("f", 32), secret, "", ""))
	if len(page.Jobs) != 0 || page.Next != "" {
		t.Fatal("end cursor did not return empty terminal page")
	}
}

func TestJobsPageStrictOptInAndLegacyCompatibility(t *testing.T) {
	s := setup(t)
	h := s.Handler()
	for _, query := range []string{"limit=", "limit=0", "limit=101", "limit=-1", "limit=%2B1", "limit=1.5", "limit=99999999999999999999999", "limit=%", "limit=2;other=1", "limit=1&limit=2", "after=", "after=short", "after=" + strings.Repeat("A", 32), "after=" + strings.Repeat("a", 32) + "&after=" + strings.Repeat("b", 32), "limit=2&tenant=other", "%6cimit=2&other=1"} {
		t.Run(query, func(t *testing.T) {
			assertPageAPIError(t, call(h, "GET", "/api/v1/jobs?"+query, secret, "", ""), 400, "invalid_page")
		})
	}
	assertPageAPIError(t, call(h, "GET", "/api/v1/jobs?limit=bad", "", "", ""), 401, "unauthorized")
	for _, path := range []string{"/api/v1/jobs", "/api/v1/jobs?unrecognized=1", "/api/v1/jobs?unrecognized=%"} {
		rec := call(h, "GET", path, secret, "", "")
		var fields map[string]json.RawMessage
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &fields) != nil || len(fields) != 1 || string(fields["jobs"]) != "null" {
			t.Fatal("legacy empty journal response changed")
		}
	}
	page := decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?limit=50", secret, "", ""))
	if len(page.Jobs) != 0 || page.Next != "" {
		t.Fatal("empty journal page malformed")
	}
	for index := range 3 {
		if _, err := s.Jobs.Create("tenant-A", jobs.KindNoop, fmt.Sprintf("journal-page-%08d", index)); err != nil {
			t.Fatal(err)
		}
	}
	page = decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?limit=2", secret, "", ""))
	if len(page.Jobs) != 2 || page.Next == "" {
		t.Fatal("journal paging missing")
	}
	last := decodeJobsPage(t, call(h, "GET", "/api/v1/jobs?limit=2&after="+page.Next, secret, "", ""))
	if len(last.Jobs) != 1 || last.Next != "" || last.Jobs[0].ID <= page.Next {
		t.Fatal("journal continuation invalid")
	}
	created := call(h, "POST", "/api/v1/jobs?limit=invalid", secret, "post-query-unchanged-0001", `{"kind":"validation.noop"}`)
	if created.Code != 202 {
		t.Fatal("POST acquired GET pagination semantics")
	}
}

func TestJobsPageCookieCancellationAndBackendFailures(t *testing.T) {
	s, _, _, token, _ := setupCookieAPI(t)
	page := decodeJobsPage(t, cookieCall(s.Handler(), "GET", "/api/v1/jobs?limit=10", token, "", "", ""))
	if len(page.Jobs) != 0 || page.Next != "" {
		t.Fatal("cookie page scope mismatch")
	}
	for _, sqlite := range []bool{false, true} {
		var server *Server
		if sqlite {
			server, _, _ = setupSQLAPI(t)
		} else {
			server = setup(t)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest("GET", "/api/v1/jobs?limit=1", nil).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+secret)
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		assertPageAPIError(t, rec, 503, "jobs_unavailable")
	}
	broken, _, db := setupSiteAPI(t)
	if _, err := db.Exec("DROP TABLE jobs"); err != nil {
		t.Fatal(err)
	}
	assertPageAPIError(t, call(broken.Handler(), "GET", "/api/v1/jobs?limit=1", secret, "", ""), http.StatusServiceUnavailable, "jobs_unavailable")
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='u1'"); err != nil {
		t.Fatal(err)
	}
	assertPageAPIError(t, call(broken.Handler(), "GET", "/api/v1/jobs?limit=1", secret, "", ""), http.StatusUnauthorized, "unauthorized")
}
