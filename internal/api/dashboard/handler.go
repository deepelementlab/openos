// Package dashboard exposes JSON APIs for Grafana / internal consoles.
package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/agentos/aos/internal/health"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler serves dashboard-oriented endpoints.
type Handler struct {
	Aggregator *health.Aggregator
}

// RegisterRoutes mounts /healthz/aggregate and /metrics on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/dashboard/health", h.handleAggregateHealth)
	mux.Handle("/metrics", promhttp.Handler())
}

func (h *Handler) handleAggregateHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Aggregator == nil {
		http.Error(w, "aggregator not configured", http.StatusServiceUnavailable)
		return
	}
	rep := h.Aggregator.Collect(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep)
}
