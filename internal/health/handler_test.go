package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q", got)
	}
	var body map[string]string
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["status"] != "ok" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestHealthRejectsMutationMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPut, http.MethodHead, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handler().ServeHTTP(rec, httptest.NewRequest(method, "/healthz", nil))
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s returned %d, want 405", method, rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != "GET" {
				t.Fatalf("Allow header = %q", got)
			}
		})
	}
}

func TestUnknownRoutesNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
}
