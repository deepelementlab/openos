package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Use a package-level singleton to avoid promauto duplicate registration
var testCollector *PrometheusCollector

func init() {
	testCollector = NewPrometheusCollector()
}

func TestPrometheusCollector_HTTPHandler(t *testing.T) {
	handler := testCollector.HTTPHandler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPrometheusCollector_RecordAgentMetrics(t *testing.T) {
	pc := testCollector
	pc.RecordAgentCreated("containerd", "nginx:latest", 500*time.Millisecond)
	pc.RecordAgentStarted("containerd")
	pc.RecordAgentStopped("containerd", "graceful")
	pc.RecordAgentDeleted("containerd", "user_request")
	pc.UpdateAgentCount("running", "containerd", 5)
}

func TestPrometheusCollector_RecordAPIMetrics(t *testing.T) {
	pc := testCollector
	pc.RecordAPIRequest("GET", "/api/v1/agents", http.StatusOK, 50*time.Millisecond)
	pc.RecordAPIRequest("POST", "/api/v1/agents", http.StatusCreated, 150*time.Millisecond)
	pc.RecordAPIError("GET", "/api/v1/agents", "not_found")
}

func TestPrometheusCollector_RecordSchedulerMetrics(t *testing.T) {
	pc := testCollector
	pc.RecordSchedulerTask("best_fit", "scheduled")
	pc.UpdateSchedulerQueue("pending", 10)
	pc.RecordSchedulerError("allocation", "insufficient_resources")
}

func TestPrometheusCollector_RecordAuthMetrics(t *testing.T) {
	pc := testCollector
	pc.RecordAuthSuccess("jwt", "user-1", 5*time.Millisecond)
	pc.RecordAuthFailure("jwt", "invalid_token")
	pc.RecordAuthorizationAttempt("agents", "read", true)
	pc.RecordAuthorizationAttempt("agents", "delete", false)
}

func TestPrometheusCollector_ResourceMetrics(t *testing.T) {
	pc := testCollector
	pc.UpdateResourceMetrics("node-1", "cpu", "cores", 8, 4, 0.5)
	pc.UpdateResourceMetrics("node-1", "memory", "bytes", 8*1024*1024*1024, 4*1024*1024*1024, 0.5)
}

func TestPrometheusCollector_SystemMetrics(t *testing.T) {
	pc := testCollector
	pc.UpdateSystemMetrics("node-1", 45.5, 8*1024*1024*1024, 100*1024*1024*1024)
	pc.UpdateNetworkMetrics("node-1", "eth0", 1024000, 512000)
}

func TestPrometheusCollector_ConcurrentOperations(t *testing.T) {
	pc := testCollector
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- struct{}{} }()
			pc.RecordAgentCreated("gvisor", "img", time.Duration(idx)*time.Millisecond)
			pc.RecordAPIRequest("GET", "/test", http.StatusOK, time.Millisecond)
			pc.UpdateAgentCount("running", "gvisor", float64(idx))
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPrometheusCollector_IntegratedWorkflow(t *testing.T) {
	pc := testCollector
	pc.RecordAgentCreated("kata", "nginx:latest", 300*time.Millisecond)
	pc.UpdateAgentCount("pending", "kata", 1)
	pc.RecordAgentStarted("kata")
	pc.RecordAPIRequest("POST", "/api/v1/agents", http.StatusCreated, 350*time.Millisecond)
	pc.RecordAuthSuccess("jwt", "admin", 5*time.Millisecond)
	pc.RecordAuthorizationAttempt("agents", "create", true)
	pc.UpdateResourceMetrics("node-2", "cpu", "cores", 1, 7, 0.125)
	pc.UpdateSystemMetrics("node-2", 12.5, 2*1024*1024*1024, 50*1024*1024*1024)
}
