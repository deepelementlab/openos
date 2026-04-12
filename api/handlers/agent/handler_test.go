package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestHandler(t *testing.T) *AgentHandler {
	logger := zap.NewNop()
	repo := repository.NewInMemoryAgentRepository()
	return NewAgentHandler(repo, logger)
}

func TestListAgents_Empty(t *testing.T) {
	handler := setupTestHandler(t)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	data := response["data"].([]interface{})
	assert.Empty(t, data)
}

func TestCreateAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	reqBody := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
		"resources": map[string]string{
			"cpu":    "500m",
			"memory": "512Mi",
		},
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	handler.Create(rec, req)
	
	assert.Equal(t, http.StatusCreated, rec.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "test-agent", data["name"])
	assert.Equal(t, "nginx:latest", data["image"])
	assert.Equal(t, "pending", data["status"])
	assert.NotEmpty(t, rec.Header().Get("Location"))
}

func TestCreateAgent_MissingName(t *testing.T) {
	handler := setupTestHandler(t)
	
	reqBody := map[string]interface{}{
		"image": "nginx:latest",
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	handler.Create(rec, req)
	
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "VALIDATION_FAILED", response["error"].(map[string]interface{})["code"])
}

func TestGetAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent first
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Get the agent
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	rec := httptest.NewRecorder()
	
	handler.Get(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, agentID, data["id"])
	assert.Equal(t, "test-agent", data["name"])
}

func TestGetAgent_NotFound(t *testing.T) {
	handler := setupTestHandler(t)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent", nil)
	rec := httptest.NewRecorder()
	
	handler.Get(rec, req)
	
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent first
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Update the agent
	updateReq := map[string]interface{}{
		"name": "updated-agent",
		"resources": map[string]string{
			"cpu": "1000m",
		},
	}
	updateBody, _ := json.Marshal(updateReq)
	
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+agentID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	handler.Update(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "updated-agent", data["name"])
}

func TestDeleteAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent first
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Delete the agent
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	rec := httptest.NewRecorder()
	
	handler.Delete(rec, req)
	
	assert.Equal(t, http.StatusNoContent, rec.Code)
	
	// Verify agent is deleted
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID, nil)
	getRec := httptest.NewRecorder()
	handler.Get(getRec, getReq)
	assert.Equal(t, http.StatusNotFound, getRec.Code)
}

func TestDeleteAgent_RunningAgent(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent first
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Manually set status to running
	agent, _ := handler.repo.Get(nil, agentID)
	agent.Status = repository.AgentStatusRunning
	handler.repo.Update(nil, agent)
	
	// Try to delete running agent
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID, nil)
	rec := httptest.NewRecorder()
	
	handler.Delete(rec, req)
	
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestListAgents_WithPagination(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create 5 agents
	for i := 0; i < 5; i++ {
		createReq := map[string]interface{}{
			"name":  "test-agent-" + string(rune('A'+i)),
			"image": "nginx:latest",
		}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.Create(rec, req)
	}
	
	// List with pagination
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?page=1&page_size=3", nil)
	rec := httptest.NewRecorder()
	
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	
	data := response["data"].([]interface{})
	assert.Len(t, data, 3)
	
	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(1), meta["page"])
	assert.Equal(t, float64(3), meta["page_size"])
	assert.Equal(t, float64(5), meta["total"])
	assert.Equal(t, float64(2), meta["total_pages"])
}

func TestStartAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Start the agent
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	rec := httptest.NewRecorder()
	
	handler.Start(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "creating", data["status"])
}

func TestStopAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create and start an agent
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	
	createReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	createReqHTTP.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReqHTTP)
	
	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Manually set status to running
	agent, _ := handler.repo.Get(nil, agentID)
	agent.Status = repository.AgentStatusRunning
	handler.repo.Update(nil, agent)
	
	// Stop the agent
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/stop", nil)
	rec := httptest.NewRecorder()
	
	handler.Stop(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "stopping", data["status"])
}
