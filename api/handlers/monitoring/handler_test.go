package monitoring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/health"
	"github.com/agentos/aos/internal/monitoring"
	"go.uber.org/zap"
)

func setupMonitoringHandler(t *testing.T) *MonitoringHandler {
	t.Helper()
	logger := zap.NewNop()
	metrics, err := monitoring.NewMetrics(&config.MonitoringConfig{Enabled: true, PrometheusEnabled: true})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	checker := health.NewChecker()
	return NewMonitoringHandler(metrics, checker, logger)
}

func TestMonitoringHandler_Health(t *testing.T) {
	handler := setupMonitoringHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := response["version"]; !ok {
		t.Fatal("expected version in response")
	}
	if _, ok := response["timestamp"]; !ok {
		t.Fatal("expected timestamp in response")
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected application/json, got %s", contentType)
	}
}

func TestMonitoringHandler_Metrics(t *testing.T) {
	handler := setupMonitoringHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.Metrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Fatalf("expected text/plain, got %s", contentType)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty metrics output")
	}
}

func TestMonitoringHandler_Ready(t *testing.T) {
	handler := setupMonitoringHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Fatalf("expected OK, got %s", rec.Body.String())
	}
}

func TestMonitoringHandler_Live(t *testing.T) {
	handler := setupMonitoringHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()

	handler.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Fatalf("expected OK, got %s", rec.Body.String())
	}
}
