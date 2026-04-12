package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/internal/monitoring"
	"github.com/agentos/aos/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupAgentAPIServer(t *testing.T) *AgentAPIServer {
	logger := zap.NewNop()
	agentRepo := repository.NewInMemoryAgentRepository()
	sandboxMgr := security.NewInMemorySandboxManager()
	networkMgr := security.NewInMemoryNetworkPolicyManager()

	cfg := &monitoring.Metrics{}
	metrics, err := monitoring.NewMetrics(nil)
	require.NoError(t, err)
	_ = cfg

	return NewAgentAPIServer(logger, metrics, agentRepo, sandboxMgr, networkMgr)
}

func TestAgentAPIServer_CreateAndGetAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
		"resources": map[string]string{
			"cpu":    "500",
			"memory": "536870912",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)

	data := createResp["data"].(map[string]interface{})
	agentID := data["id"].(string)
	assert.NotEmpty(t, agentID)
	assert.Equal(t, "test-agent", data["name"])
	assert.Equal(t, "pending", data["status"])

	// Get agent
	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var getResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &getResp)
	assert.Equal(t, agentID, getResp["data"].(map[string]interface{})["id"])
}

func TestAgentAPIServer_ListAgents(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create 3 agents sequentially (small delay to avoid ID collision)
	for i := 0; i < 3; i++ {
		reqBody := map[string]interface{}{
			"name":  fmt.Sprintf("list-agent-%d", i),
			"image": "nginx:latest",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if !assert.Equal(t, http.StatusCreated, rec.Code) {
			t.Logf("Create agent %d failed: %s", i, rec.Body.String())
			return
		}
	}

	// List agents
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?page=1&page_size=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var listResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	data := listResp["data"].([]interface{})
	assert.Len(t, data, 3)

	meta := listResp["meta"].(map[string]interface{})
	assert.Equal(t, float64(3), meta["total"])
}

func TestAgentAPIServer_UpdateAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "original",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Update agent
	updateBody := map[string]interface{}{
		"name": "updated",
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+agentID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var updateResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &updateResp)
	assert.Equal(t, "updated", updateResp["data"].(map[string]interface{})["name"])
}

func TestAgentAPIServer_DeleteAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "to-delete",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Delete agent
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify deleted
	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAgentAPIServer_StartStopAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "lifecycle-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Start agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var startResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &startResp)
	assert.Equal(t, "running", startResp["data"].(map[string]interface{})["status"])

	// Stop agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/stop", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var stopResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &stopResp)
	assert.Equal(t, "stopped", stopResp["data"].(map[string]interface{})["status"])
}

func TestAgentAPIServer_GetAgentStats(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "stats-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Get stats
	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID+"/stats", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var statsResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &statsResp)
	data := statsResp["data"].(map[string]interface{})
	assert.Equal(t, agentID, data["agentId"])
}

func TestAgentAPIServer_SecurityEndpoints(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent (creates sandbox + network policy automatically)
	reqBody := map[string]interface{}{
		"name":  "security-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// List sandboxes
	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/sandboxes", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var sandboxResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &sandboxResp)
	sandboxes := sandboxResp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(sandboxes), 1)

	// List network policies
	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/network-policies", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var netResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &netResp)
	policies := netResp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(policies), 1)
}

func TestAgentAPIServer_CreateAgent_ValidationFailure(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Missing name
	reqBody := map[string]interface{}{
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.False(t, errResp["success"].(bool))
}

func TestAgentAPIServer_GetAgent_NotFound(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAgentAPIServer_DeleteRunningAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create and start agent
	reqBody := map[string]interface{}{
		"name":  "running-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Start agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Try to delete running agent
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestAgentAPIServer_RestartAgent(t *testing.T) {
	srv := setupAgentAPIServer(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create agent
	reqBody := map[string]interface{}{
		"name":  "restart-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	agentID := createResp["data"].(map[string]interface{})["id"].(string)

	// Start first
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Restart
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/restart", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var restartResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &restartResp)
	assert.Equal(t, "running", restartResp["data"].(map[string]interface{})["status"])
}
