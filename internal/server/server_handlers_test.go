package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/data/repository"
	"go.uber.org/zap"
)

func newHandlersTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    8080,
			Mode:                    "test",
			GracefulShutdownTimeout: 1,
		},
	}
	srv, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

// --- Health endpoints ---

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleHealth_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := body["version"]; !ok {
		t.Error("expected version key in health response")
	}
	if _, ok := body["timestamp"]; !ok {
		t.Error("expected timestamp key in health response")
	}
}

func TestHandleReady_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	srv.handleReady(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleReady_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/ready", nil)
	rec := httptest.NewRecorder()
	srv.handleReady(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleLive_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()
	srv.handleLive(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleLive_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPut, "/live", nil)
	rec := httptest.NewRecorder()
	srv.handleLive(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- Root endpoint ---

func TestHandleRoot_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.handleRoot(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["name"] != "Agent OS" {
		t.Errorf("expected name=Agent OS, got %v", body["name"])
	}
	endpoints, ok := body["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("expected endpoints map")
	}
	if _, ok := endpoints["health"]; !ok {
		t.Error("expected health endpoint in response")
	}
	if _, ok := endpoints["agents"]; !ok {
		t.Error("expected agents endpoint in response")
	}
}

func TestHandleRoot_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	srv.handleRoot(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- Metrics endpoint ---

func TestHandleMetrics_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.handleMetrics(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/plain" {
		t.Errorf("expected text/plain content-type, got %s", ct)
	}
}

func TestHandleMetrics_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.handleMetrics(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- Agent CRUD ---

func TestHandleAgents_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	srv.handleAgents(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAgent_MethodNotAllowed(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/agent-1", nil)
	rec := httptest.NewRecorder()
	srv.handleAgent(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestGetAgent_Found(t *testing.T) {
	srv := newHandlersTestServer(t)
	srv.agentRepo.Create(context.Background(), &repository.Agent{
		ID:     "agent-42",
		Name:   "find-me",
		Image:  "nginx",
		Status: repository.AgentStatusRunning,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent-42", nil)
	rec := httptest.NewRecorder()
	srv.getAgent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var agent repository.Agent
	if err := json.Unmarshal(rec.Body.Bytes(), &agent); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if agent.ID != "agent-42" {
		t.Errorf("expected agent-42, got %s", agent.ID)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.getAgent(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetAgent_EmptyID(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/", nil)
	rec := httptest.NewRecorder()
	srv.getAgent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteAgent_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	srv.agentRepo.Create(context.Background(), &repository.Agent{
		ID: "agent-del", Name: "del-me", Image: "nginx", Status: repository.AgentStatusRunning,
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/agent-del", nil)
	rec := httptest.NewRecorder()
	srv.deleteAgent(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/nope", nil)
	rec := httptest.NewRecorder()
	srv.deleteAgent(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteAgent_EmptyID(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/", nil)
	rec := httptest.NewRecorder()
	srv.deleteAgent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateAgent_Success(t *testing.T) {
	srv := newHandlersTestServer(t)
	srv.agentRepo.Create(context.Background(), &repository.Agent{
		ID: "agent-upd", Name: "original", Image: "nginx", Status: repository.AgentStatusPending,
	})

	body := `{"name":"updated","status":"running"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/agent-upd", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.updateAgent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var agent repository.Agent
	json.Unmarshal(rec.Body.Bytes(), &agent)
	if agent.Name != "updated" {
		t.Errorf("expected name=updated, got %s", agent.Name)
	}
	if agent.Status != repository.AgentStatusRunning {
		t.Errorf("expected running, got %s", agent.Status)
	}
}

func TestUpdateAgent_NotFound(t *testing.T) {
	srv := newHandlersTestServer(t)
	body := `{"name":"x"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/nope", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	srv.updateAgent(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateAgent_EmptyID(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	srv.updateAgent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateAgent_InvalidBody(t *testing.T) {
	srv := newHandlersTestServer(t)
	srv.agentRepo.Create(context.Background(), &repository.Agent{
		ID: "agent-bad", Name: "x", Image: "y", Status: repository.AgentStatusPending,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/agent-bad", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()
	srv.updateAgent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateAgent_ResourcesAndRuntime(t *testing.T) {
	srv := newHandlersTestServer(t)
	srv.agentRepo.Create(context.Background(), &repository.Agent{
		ID: "agent-res", Name: "x", Image: "y", Status: repository.AgentStatusPending,
	})

	body := `{"runtime":"gvisor","resources":{"cpu":"2","memory":"4Gi"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/agent-res", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.updateAgent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var agent repository.Agent
	json.Unmarshal(rec.Body.Bytes(), &agent)
	if agent.Runtime != "gvisor" {
		t.Errorf("expected gvisor, got %s", agent.Runtime)
	}
	if agent.Resources["cpu"] != "2" {
		t.Errorf("expected cpu=2, got %s", agent.Resources["cpu"])
	}
}

func TestCreateAgent_InvalidJSON(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.createAgent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListAgents_Empty(t *testing.T) {
	srv := newHandlersTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	srv.listAgents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)
	pagination, ok := body["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pagination in response")
	}
	if pagination["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", pagination["total"])
	}
}

// --- writeJSON helper ---

func TestWriteJSON_Error(t *testing.T) {
	err := writeJSON(httptest.NewRecorder(), make(chan int))
	if err == nil {
		t.Error("expected error for unmarshallable type")
	}
}
