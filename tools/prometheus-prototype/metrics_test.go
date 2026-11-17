package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrototypeExposesOnlyMarker(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "tong_panel_prototype_info 1") {
		t.Fatalf("missing marker: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "go_gc_") {
		t.Fatal("unexpected default Go metrics")
	}
}
