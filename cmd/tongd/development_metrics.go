package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var errDevelopmentMetricsConfiguration = errors.New("invalid development metrics configuration; use a separate random token of 32 to 4096 characters")

// newDevelopmentMetrics is disabled by default. It does not report health,
// tenant data, request URLs, process details or host state.
func newDevelopmentMetrics(token, apiToken string) (http.Handler, error) {
	if token == "" {
		return nil, nil
	}
	if len(token) < 32 || len(token) > 4096 || strings.TrimSpace(token) != token || token == apiToken {
		return nil, errDevelopmentMetricsConfiguration
	}
	for _, char := range token {
		if char < 33 || char > 126 {
			return nil, errDevelopmentMetricsConfiguration
		}
	}
	expected := sha256.Sum256([]byte("Bearer " + token))
	registry := prometheus.NewRegistry()
	marker := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tong_panel_development_metrics_info",
		Help: "Indicates that the optional development metrics endpoint is responding; not a health check.",
	})
	marker.Set(1)
	registry.MustRegister(marker)
	scrape := promhttp.HandlerFor(registry, promhttp.HandlerOpts{MaxRequestsInFlight: 1, Timeout: 5 * time.Second})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		remote, err := netip.ParseAddrPort(r.RemoteAddr)
		if err != nil || !remote.Addr().Is4() || !remote.Addr().IsLoopback() {
			http.NotFound(w, r)
			return
		}
		values := r.Header.Values("Authorization")
		if len(values) != 1 || len(values[0]) > 4103 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="development-metrics"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		provided := sha256.Sum256([]byte(values[0]))
		if subtle.ConstantTimeCompare(expected[:], provided[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="development-metrics"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.RawQuery != "" {
			http.Error(w, "query parameters are unsupported", http.StatusBadRequest)
			return
		}
		scrape.ServeHTTP(w, r)
	}), nil
}

func withDevelopmentMetrics(next, metrics http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" && metrics != nil {
			metrics.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
