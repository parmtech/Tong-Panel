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
	for _, p := range []string{"/", "/ui/style.css", "/ui/app.js"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, p, nil))
		if r.Code != 200 {
			t.Fatalf("%s got %d: %s", p, r.Code, r.Body.String())
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(r.Body.String(), "Development preview") {
		t.Fatal("UI not a labeled preview")
	}
	for _, p := range []string{"/ui/../../../etc/passwd", "/settings"} {
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", p, nil))
		if r.Code == 200 {
			t.Fatalf("unexpected route %s", p)
		}
	}
}
