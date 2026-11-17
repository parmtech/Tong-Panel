// Package metrics is an isolated proof of concept; it is not wired to tongd.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler exposes a constant prototype marker, not a service health check.
// The caller must restrict access before exposing this handler to a network.
func NewHandler() http.Handler {
	registry := prometheus.NewRegistry()
	marker := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tong_panel_prototype_info",
		Help: "Indicates that the isolated metrics prototype is responding.",
	})
	marker.Set(1)
	registry.MustRegister(marker)
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
