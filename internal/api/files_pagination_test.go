package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/rbac"
)

type pagedTestFiles struct {
	tenant, path, after, prefix string
	limit                       int
}

func (f *pagedTestFiles) ListForTenant(_, _ string, _ int) ([]TenantFileEntry, error) {
	return []TenantFileEntry{{Name: "legacy", Kind: "file"}}, nil
}
func (f *pagedTestFiles) ReadForTenant(_, _ string, _ int64) ([]byte, error) { return nil, nil }
func (f *pagedTestFiles) WriteNewForTenant(_, _ string, _ io.Reader, _ int64) (int64, error) {
	return 0, nil
}
func (f *pagedTestFiles) RemoveForTenant(_, _ string) (int64, error) { return 0, nil }
func (f *pagedTestFiles) ListPageForTenant(tenant, path, after string, limit int) ([]TenantFileEntry, string, error) {
	f.tenant, f.path, f.after, f.limit = tenant, path, after, limit
	return []TenantFileEntry{{Name: "a", Kind: "file"}}, "a", nil
}

func (f *pagedTestFiles) ListPagePrefixForTenant(tenant, path, after, prefix string, limit int) ([]TenantFileEntry, string, error) {
	f.prefix = prefix
	return f.ListPageForTenant(tenant, path, after, limit)
}

func TestFileListPaginationOptInAndAuthorization(t *testing.T) {
	files := &pagedTestFiles{}
	s := &Server{Files: files}
	owner := rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner}
	run := func(query string, p rbac.Principal) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/v1/files"+query, nil)
		w := httptest.NewRecorder()
		s.fileList(w, r, p)
		return w
	}
	w := run("?path=public&page=1&after=old&limit=2", owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"next_after":"a"`) || files.tenant != "tenant-a" || files.path != "public" || files.after != "old" || files.limit != 2 {
		t.Fatalf("response %d %s service=%+v", w.Code, w.Body.String(), files)
	}
	w = run("?page=1&prefix=app&path=public&after=app-a", owner)
	if w.Code != 200 || files.prefix != "app" || files.tenant != "tenant-a" || files.after != "app-a" {
		t.Fatalf("filtered response %d service=%+v", w.Code, files)
	}
	w = run("", owner)
	if w.Code != 200 || strings.Contains(w.Body.String(), "next_after") {
		t.Fatalf("legacy response %d %s", w.Code, w.Body.String())
	}
	for _, query := range []string{"?prefix=a", "?page=1&prefix=a&prefix=b", "?page=1&prefix=a%2Fb", "?page=1&prefix=" + strings.Repeat("x", 129), "?after=a", "?page=2", "?page=1&page=1", "?page=1&after=a&after=b", "?page=1&tenant=other", "?page=1&limit=1001"} {
		if w = run(query, owner); w.Code != 400 {
			t.Errorf("%s returned %d", query, w.Code)
		}
	}
	w = run("?page=1", rbac.Principal{UserID: "auditor", TenantID: "tenant-a", Role: rbac.Auditor})
	if w.Code != 403 {
		t.Fatalf("auditor got %d", w.Code)
	}
}
