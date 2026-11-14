//go:build linux

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestFilePaginationCookieAndPinnedTenantRoots(t *testing.T) {
	s, sessions, _, tokenA, _ := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)
	for root, names := range map[string][]string{
		rootA: {"a-one", "a-two", "a-three"},
		rootB: {"b-only"},
	} {
		dir := filepath.Join(root, "pages")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	issue := func(p rbac.Principal) string {
		token, _, _, err := sessions.Issue(p, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	tokenB := issue(rbac.Principal{UserID: "u3", TenantID: "tenant-b", Role: rbac.Customer})
	auditor := issue(rbac.Principal{UserID: "u4", TenantID: "tenant-a", Role: rbac.Auditor})
	h := s.Handler()
	type listing struct {
		Entries []TenantFileEntry `json:"entries"`
		Next    string            `json:"next_after"`
	}
	page := func(token, query string, wantNames []string, wantNext string) {
		t.Helper()
		response := sessionGET(h, "/api/v1/files?path=pages&page=1&limit=2"+query, token)
		if response.Code != http.StatusOK {
			t.Fatalf("listing: %d %s", response.Code, response.Body.String())
		}
		var body listing
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(body.Entries))
		for _, e := range body.Entries {
			names = append(names, e.Name)
		}
		if !reflect.DeepEqual(names, wantNames) || body.Next != wantNext {
			t.Fatalf("got names %v and cursor %q, want %v and %q", names, body.Next, wantNames, wantNext)
		}
	}
	page(tokenA, "", []string{"a-one", "a-three"}, "a-three")
	page(tokenA, "&after=a-three", []string{"a-two"}, "")
	page(tokenB, "", []string{"b-only"}, "")
	for _, tc := range []struct {
		token, query string
		want         int
	}{
		{"", "", http.StatusUnauthorized},
		{auditor, "", http.StatusForbidden},
		{tokenA, "&tenant=tenant-b", http.StatusBadRequest},
	} {
		response := sessionGET(h, "/api/v1/files?path=pages&page=1&limit=2"+tc.query, tc.token)
		if response.Code != tc.want || strings.Contains(response.Body.String(), "b-only") {
			t.Fatalf("denied listing: %d %s, want %d", response.Code, response.Body.String(), tc.want)
		}
	}
	if err := sessions.Revoke(tokenA); err != nil {
		t.Fatal(err)
	}
	response := sessionGET(h, "/api/v1/files?path=pages&page=1&limit=2&after=a-three", tokenA)
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "a-two") {
		t.Fatalf("revoked second page: %d %s", response.Code, response.Body.String())
	}
}
