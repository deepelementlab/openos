package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/agentos/aos/internal/health"
	"github.com/agentos/aos/internal/monitoring"
	"github.com/agentos/aos/internal/version"
	"go.uber.org/zap"
)

// MonitoringHandler handles metrics and health HTTP endpoints.
type MonitoringHandler struct {
	metrics       *monitoring.Metrics
	healthChecker *health.Checker
	logger        *zap.Logger
}

func NewMonitoringHandler(m *monitoring.Metrics, hc *health.Checker, logger *zap.Logger) *MonitoringHandler {
	return &MonitoringHandler{
		metrics:       m,
		healthChecker: hc,
		logger:        logger,
	}
}

// Health responds with aggregated health status.
func (h *MonitoringHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.healthChecker.Check(r.Context())
	status := h.healthChecker.Status()
	status["version"] = version.GetVersion()
	status["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

// Metrics responds with Prometheus-style metrics.
func (h *MonitoringHandler) Metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.metrics.GetMetrics()))
}

// Ready responds to readiness probes.
func (h *MonitoringHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// Live responds to liveness probes.
func (h *MonitoringHandler) Live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
