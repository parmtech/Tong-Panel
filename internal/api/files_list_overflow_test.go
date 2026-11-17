//go:build linux

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesListRejectsOverLimitWithoutReturningPartialEntries(t *testing.T) {
	s, _, _, tokenA, _ := setupCookieAPI(t)
	rootA, rootB := attachTestTenantFiles(t, s)

	if err := os.Mkdir(filepath.Join(rootA, "overflow"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tenant-a-first.txt", "tenant-a-second.txt"} {
		if err := os.WriteFile(filepath.Join(rootA, "overflow", name), []byte("visible"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(rootB, "overflow"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "overflow", "tenant-b-only.txt"), []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	rec := sessionGET(s.Handler(), "/api/v1/files?path=overflow&limit=1", tokenA)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("over-limit listing status=%d body=%q; want %d", rec.Code, rec.Body.String(), http.StatusRequestEntityTooLarge)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Entries json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("over-limit listing response is not JSON: %v; body=%q", err, rec.Body.String())
	}
	if body.Error.Code != "file_limit_exceeded" {
		t.Fatalf("over-limit listing error=%q; want file_limit_exceeded", body.Error.Code)
	}
	if body.Entries != nil {
		t.Fatalf("over-limit listing returned entries field %s: %s", body.Entries, rec.Body.String())
	}
	for _, name := range []string{"tenant-a-first.txt", "tenant-a-second.txt", "tenant-b-only.txt"} {
		if strings.Contains(rec.Body.String(), name) {
			t.Fatalf("over-limit listing leaked %q: %s", name, rec.Body.String())
		}
	}
}
