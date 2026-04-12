package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/server"
	"go.uber.org/zap"
)

func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
		Mode: "test",
	}
	srv, err := server.NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv
}

func TestAgentLifecycle(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.GetHandler()

	body := bytes.NewBufferString(`{"name":"test-agent","image":"alpine:latest"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create agent: expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	agentID, ok := createResp["id"].(string)
	if !ok || agentID == "" {
		t.Fatalf("create response missing id: %v", createResp)
	}
	if createResp["status"] != "creating" {
		t.Errorf("expected status creating, got %v", createResp["status"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get agent: expected 200, got %d", w.Code)
	}
	var getResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if getResp["status"] != "creating" {
		t.Errorf("expected status creating, got %v", getResp["status"])
	}

	body = bytes.NewBufferString(`{"status":"running"}`)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+agentID, body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update agent: expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var updateResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&updateResp); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}
	if updateResp["status"] != "running" {
		t.Errorf("expected status running after update, got %v", updateResp["status"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list agents: expected 200, got %d", w.Code)
	}
	var listResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	agents, ok := listResp["agents"].([]interface{})
	if !ok {
		t.Fatalf("agents field is not an array: %v", listResp)
	}
	found := false
	for _, a := range agents {
		if m, ok := a.(map[string]interface{}); ok && m["id"] == agentID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created agent not found in list")
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete agent: expected 204, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("get deleted agent: expected 404, got %d", w.Code)
	}
}
