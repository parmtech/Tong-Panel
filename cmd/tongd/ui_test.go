package main

import (
	"github.com/parmtech/tong-panel/internal/api"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUIAndStaticRouting(t *testing.T) {
	h := publicHandler(&api.Server{})
	for _, p := range []string{"/", "/ui/style.css", "/ui/app.js", "/ui/panel-status.mjs", "/ui/site-preview.mjs", "/ui/files-explorer.mjs"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, p, nil))
		if r.Code != 200 {
			t.Fatalf("%s got %d: %s", p, r.Code, r.Body.String())
		}
		if (p == "/ui/panel-status.mjs" || p == "/ui/site-preview.mjs" || p == "/ui/files-explorer.mjs") && !strings.HasPrefix(r.Header().Get("Content-Type"), "text/javascript") {
			t.Fatalf("module has unexpected Content-Type %q", r.Header().Get("Content-Type"))
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(r.Body.String(), "Development preview") {
		t.Fatal("UI not a labeled preview")
	}
	for _, label := range []string{`aria-label="Home"`, `aria-label="Roadmap"`, `aria-label="Updates"`, `aria-label="Source code"`, `id="refresh-status"`, `id="site-preview-form"`, `id="site-preview-status"`, `id="files-explorer-title"`, `id="files-refresh"`, `id="files-status"`, `aria-live="polite"`} {
		if !strings.Contains(r.Body.String(), label) {
			t.Fatalf("UI missing accessible status/navigation contract %q", label)
		}
	}
	for _, p := range []string{"/ui/../../../etc/passwd", "/settings"} {
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", p, nil))
		if r.Code == 200 {
			t.Fatalf("unexpected route %s", p)
		}
	}
}
