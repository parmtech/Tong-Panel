package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/api"
)

const metricsTestToken = "metrics-fixture-canary-separate-secret-12345"
const apiTestToken = "api-fixture-canary-separate-secret-678901234"

func TestDevelopmentMetricsOptInConfiguration(t *testing.T) {
	if handler, err := newDevelopmentMetrics("", apiTestToken); err != nil || handler != nil {
		t.Fatal("not disabled by default")
	}
	for _, value := range []string{"short", " " + metricsTestToken, metricsTestToken + "\n", strings.Repeat("x", 4097), apiTestToken, strings.Repeat("x", 31) + "\tX"} {
		if handler, err := newDevelopmentMetrics(value, apiTestToken); handler != nil || !errors.Is(err, errDevelopmentMetricsConfiguration) || strings.Contains(err.Error(), value) {
			t.Fatal("invalid metrics configuration did not fail without disclosing its value")
		}
	}
}

func TestDevelopmentMetricsAccessAndMinimalPayload(t *testing.T) {
	metrics, err := newDevelopmentMetrics(metricsTestToken, apiTestToken)
	if err != nil {
		t.Fatal(err)
	}
	handler := withDevelopmentMetrics(publicHandler(&api.Server{}), metrics)
	for _, tc := range []struct {
		name, remote, method, path string
		authorizations             []string
		status                     int
	}{
		{"valid", "127.0.0.1:1234", "GET", "/metrics", []string{"Bearer " + metricsTestToken}, 200},
		{"missing", "127.0.0.1:1234", "GET", "/metrics", nil, 401},
		{"api token", "127.0.0.1:1234", "GET", "/metrics", []string{"Bearer " + apiTestToken}, 401},
		{"query token", "127.0.0.1:1234", "GET", "/metrics?token=" + metricsTestToken, nil, 401},
		{"duplicate", "127.0.0.1:1234", "GET", "/metrics", []string{"Bearer " + metricsTestToken, "Bearer " + metricsTestToken}, 401},
		{"remote", "192.0.2.1:1234", "GET", "/metrics", []string{"Bearer " + metricsTestToken}, 404},
		{"malformed remote", "localhost:1234", "GET", "/metrics", []string{"Bearer " + metricsTestToken}, 404},
		{"ipv6", "[::1]:1234", "GET", "/metrics", []string{"Bearer " + metricsTestToken}, 404},
		{"method", "127.0.0.1:1234", "POST", "/metrics", []string{"Bearer " + metricsTestToken}, 405},
		{"query", "127.0.0.1:1234", "GET", "/metrics?extra=1", []string{"Bearer " + metricsTestToken}, 400},
		{"trailing path", "127.0.0.1:1234", "GET", "/metrics/", []string{"Bearer " + metricsTestToken}, 404},
		{"health", "127.0.0.1:1234", "GET", "/healthz", nil, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.RemoteAddr = tc.remote
			r.Header.Set("X-Forwarded-For", "127.0.0.1")
			for _, value := range tc.authorizations {
				r.Header.Add("Authorization", value)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d", w.Code, tc.status)
			}
			if strings.Contains(w.Body.String(), metricsTestToken) || strings.Contains(w.Body.String(), apiTestToken) {
				t.Fatal("credential disclosed")
			}
			if tc.name == "valid" {
				body := w.Body.String()
				if !strings.Contains(body, "tong_panel_development_metrics_info 1\n") || strings.Count(body, "# TYPE ") != 1 || strings.Contains(body, "go_gc_") || strings.Contains(body, "process_") {
					t.Fatalf("unexpected metric families: %s", body)
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("scrape cacheable")
				}
			}
		})
	}
	disabled := withDevelopmentMetrics(publicHandler(&api.Server{}), nil)
	w := httptest.NewRecorder()
	disabled.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 404 {
		t.Fatal("disabled endpoint exposed")
	}
}

func TestDevelopmentMetricsActualLoopbackHTTP(t *testing.T) {
	metrics, err := newDevelopmentMetrics(metricsTestToken, apiTestToken)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(withDevelopmentMetrics(publicHandler(&api.Server{}), metrics))
	defer server.Close()
	r, _ := http.NewRequest("GET", server.URL+"/metrics", nil)
	r.Header.Set("Authorization", "Bearer "+metricsTestToken)
	response, err := server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), "tong_panel_development_metrics_info 1\n") {
		t.Fatal("actual loopback scrape failed")
	}
}

func TestDevelopmentMetricsInvalidConfigurationBeforeState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TONG_DEV_LISTEN_ADDR", "127.0.0.1:8090")
	t.Setenv("TONG_DEV_TOKEN", apiTestToken)
	t.Setenv("TONG_DEV_METRICS_TOKEN", apiTestToken)
	t.Setenv("TONG_DATA_DIR", filepath.Join(root, "must-not-create"))
	t.Setenv("TONG_DEV_SQLITE_DB", "")
	t.Setenv("TONG_DEV_RUN_NOOP_WORKER", "")
	if err := runDevelopmentDaemon(context.Background(), log.New(io.Discard, "", 0)); !errors.Is(err, errDevelopmentMetricsConfiguration) {
		t.Fatal("metrics configuration was not rejected first")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("invalid metrics configuration created state")
	}
}
