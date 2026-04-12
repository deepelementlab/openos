package server

import (
	"bytes"
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

func newWFServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                    "127.0.0.1",
			Port:                    0,
			Mode:                    "test",
			GracefulShutdownTimeout: 2 * time.Second,
		},
	}
	srv, err := NewServer(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

func TestHandleWorkflows_Get(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflows(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := body["registered_workflows"]; !ok {
		t.Fatal("expected registered_workflows key in response")
	}
}

func TestHandleWorkflows_MethodNotAllowed(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflows(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWorkflow_Get(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/agent-deploy", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflow(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "available" {
		t.Fatalf("expected status=available, got %v", body["status"])
	}
}

func TestHandleWorkflow_MethodNotAllowed(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/agent-deploy", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflow(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleStartWorkflow_InvalidJSON(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/start", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartWorkflow(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleStartWorkflow_MissingFields(t *testing.T) {
	srv := newWFServer(t)
	body := `{"workflow_id":"agent-deploy"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartWorkflow(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("workflow_id and entity_id are required")) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleStartWorkflow_MethodNotAllowed(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/start", nil)
	rec := httptest.NewRecorder()
	srv.handleStartWorkflow(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWorkflowInstance_GetNotFound(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflowInstance(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkflowInstance_DeleteNotFound(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflowInstance(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkflowInstance_MethodNotAllowed(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/some-id", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkflowInstance(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleStateMachine_GetNotFound(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/state-machines/nonexistent-entity", nil)
	rec := httptest.NewRecorder()
	srv.handleStateMachine(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleStateMachine_PostInvalidJSON(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/state-machines/some-entity", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStateMachine(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleStateMachine_MethodNotAllowed(t *testing.T) {
	srv := newWFServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/state-machines/some-entity", nil)
	rec := httptest.NewRecorder()
	srv.handleStateMachine(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestInitPostgresDataComponents_IncompleteConfig(t *testing.T) {
	cfg := &config.Config{}
	_, _, _, err := initPostgresDataComponents(cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for incomplete config")
	}
	if err.Error() != "database config incomplete" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleAgentBaseline_EmptyID(t *testing.T) {
	srv := newWFServer(t)
	_, err := srv.scheduleAgentBaseline(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent id")
	}
	if err.Error() != "agent id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleAgentBaseline_NilScheduler(t *testing.T) {
	srv := newWFServer(t)
	srv.scheduler = nil
	nodeID, err := srv.scheduleAgentBaseline(context.Background(), "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nodeID != "node-local-1" {
		t.Fatalf("expected node-local-1, got %s", nodeID)
	}
}

func TestHandleWorkflows_Integration(t *testing.T) {
	srv := newWFServer(t)

	startBody := `{"workflow_id":"agent-deploy","entity_id":"agent-test-001","entity_type":"agent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows//start", strings.NewReader(startBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStartWorkflow(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["workflow_id"] != "agent-deploy" {
		t.Fatalf("expected workflow_id=agent-deploy, got %v", resp["workflow_id"])
	}
	if resp["entity_id"] != "agent-test-001" {
		t.Fatalf("expected entity_id=agent-test-001, got %v", resp["entity_id"])
	}
	if resp["instance_id"] == nil || resp["instance_id"] == "" {
		t.Fatal("expected non-empty instance_id")
	}
	if resp["status"] == nil {
		t.Fatal("expected status field")
	}

	instanceID, ok := resp["instance_id"].(string)
	if !ok || instanceID == "" {
		t.Fatal("instance_id is not a valid string")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+instanceID, nil)
	getRec := httptest.NewRecorder()
	srv.handleWorkflowInstance(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for get instance, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestHandleStateMachine_PostSendEvent(t *testing.T) {
	srv := newWFServer(t)
	ctx := context.Background()

	_, err := srv.stateMachineEngine.CreateMachine(ctx, "agent-sm-001", "agent", nil)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}

	eventBody := `{"event":"schedule"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/state-machines/agent-sm-001", strings.NewReader(eventBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleStateMachine(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["success"] != true {
		t.Fatalf("expected success=true, got %v", resp["success"])
	}
	if resp["from_state"] != "created" {
		t.Fatalf("expected from_state=created, got %v", resp["from_state"])
	}
	if resp["to_state"] != "scheduled" {
		t.Fatalf("expected to_state=scheduled, got %v", resp["to_state"])
	}
}
