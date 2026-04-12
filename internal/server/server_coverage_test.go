package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
	"go.uber.org/zap"
)

// ===== Lifecycle Tests =====

func TestServerCoverage_StartAndShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
	}
	s, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	err = <-done
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}

func TestServerCoverage_Shutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
	}
	s, _ := NewServer(cfg, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	err := s.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestServerCoverage_GetHandler(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()
	if handler == nil {
		t.Fatal("GetHandler() returned nil")
	}
}

func TestServerCoverage_GetConfig(t *testing.T) {
	s := newCovServer(t)
	cfg := s.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig() returned nil")
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.Server.Host)
	}
}

// ===== Health Endpoint Tests =====

func TestServerCoverage_HealthEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestServerCoverage_ReadyEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestServerCoverage_LiveEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestServerCoverage_RootEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// ===== Agent CRUD via Server handlers =====

func TestServerCoverage_CreateAgent(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	body := `{"name":"cov-test","image":"nginx:latest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create agent: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatalf("response missing id: %v", resp)
	}
}

func TestServerCoverage_ListAgents(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	// Create an agent first
	_ = createCovAgent(t, handler, "list-test", "nginx:latest")

	// List agents
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("list agents: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerCoverage_GetAgent(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	agentID := createCovAgent(t, handler, "get-test", "nginx:latest")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("get agent: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerCoverage_DeleteAgent(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	agentID := createCovAgent(t, handler, "delete-test", "nginx:latest")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Server returns 204 No Content on successful delete
	if rec.Code != http.StatusNoContent {
		t.Errorf("delete agent: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerCoverage_UpdateAgent(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	agentID := createCovAgent(t, handler, "update-test", "nginx:latest")

	body := `{"name":"updated-name","image":"nginx:latest"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+agentID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("update agent: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerCoverage_CreateAgent_MissingFields(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", rec.Code)
	}
}

func TestServerCoverage_CreateAgent_InvalidJSON(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestServerCoverage_GetAgent_NotFound(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing agent, got %d", rec.Code)
	}
}

// ===== Metrics Endpoint =====

func TestServerCoverage_MetricsEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	// Metrics endpoint may return 200 or 404 depending on config
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Errorf("expected 200 or 404 for metrics, got %d", rec.Code)
	}
}

// ===== Status Endpoint =====

func TestServerCoverage_StatusEndpoint(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestServerCoverage_StatusEndpoint_MethodNotAllowed(t *testing.T) {
	s := newCovServer(t)
	req := httptest.NewRequest(http.MethodPost, "/status", nil)
	rec := httptest.NewRecorder()
	s.GetHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// ===== Agent Lifecycle (POST to /agents/{id}/start|stop|restart) =====
// Note: The base Server's handleAgent only routes GET/PUT/DELETE.
// POST lifecycle actions are handled by lifecycle.go's handleAgentRoutes,
// which is NOT wired into the base Server's setupRoutes.
// These tests verify the routing behavior (expecting 405 for POST).

func TestServerCoverage_AgentStartLifecycle(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()
	agentID := createCovAgent(t, handler, "start-test", "nginx:latest")

	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	startRec := httptest.NewRecorder()
	handler.ServeHTTP(startRec, startReq)
	// The base Server does not register lifecycle POST handlers, so expect 405
	if startRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("start agent: expected 405 (lifecycle not wired), got %d: %s", startRec.Code, startRec.Body.String())
	}
}

func TestServerCoverage_AgentStopLifecycle(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()
	agentID := createCovAgent(t, handler, "stop-test", "nginx:latest")

	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/stop", nil)
	stopRec := httptest.NewRecorder()
	handler.ServeHTTP(stopRec, stopReq)
	if stopRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("stop agent: expected 405 (lifecycle not wired), got %d: %s", stopRec.Code, stopRec.Body.String())
	}
}

func TestServerCoverage_AgentRestart(t *testing.T) {
	s := newCovServer(t)
	handler := s.GetHandler()
	agentID := createCovAgent(t, handler, "restart-test", "nginx:latest")

	restartReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/restart", nil)
	restartRec := httptest.NewRecorder()
	handler.ServeHTTP(restartRec, restartReq)
	if restartRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("restart agent: expected 405 (lifecycle not wired), got %d: %s", restartRec.Code, restartRec.Body.String())
	}
}

// ===== Helpers =====

func newCovServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
	}
	s, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return s
}

func createCovAgent(t *testing.T, handler http.Handler, name, image string) string {
	t.Helper()
	body := `{"name":"` + name + `","image":"` + image + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create agent: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("create agent: failed to parse response: %v", err)
	}
	id, ok := resp["id"].(string)
	if !ok || id == "" {
		t.Fatalf("create agent: response missing id: %v", resp)
	}
	return id
}
