package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/stretchr/testify/assert"

)

func TestRestartAgent_Success(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create an agent
	createReq := map[string]interface{}{
		"name":  "test-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var createResponse map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Set agent to running status
	agent, _ := handler.repo.Get(nil, agentID)
	agent.Status = repository.AgentStatusRunning
	handler.repo.Update(nil, agent)
	
	// Restart the agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/restart", nil)
	rec = httptest.NewRecorder()
	handler.Restart(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "stopping", data["status"])
}

func TestRestartAgent_ErrorState(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create and set to error state
	createReq := map[string]interface{}{"name": "err-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var createResponse map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	agent, _ := handler.repo.Get(nil, agentID)
	agent.Status = repository.AgentStatusError
	handler.repo.Update(nil, agent)
	
	// Restart from error state
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/restart", nil)
	rec = httptest.NewRecorder()
	handler.Restart(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRestartAgent_InvalidState(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create agent (pending state)
	createReq := map[string]interface{}{"name": "pending-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var createResponse map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResponse)
	agentID := createResponse["data"].(map[string]interface{})["id"].(string)
	
	// Restart from pending state should fail
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/restart", nil)
	rec = httptest.NewRecorder()
	handler.Restart(rec, req)
	
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestListAgents_FilterByStatus(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create agents with different statuses
	for i := 0; i < 3; i++ {
		createReq := map[string]interface{}{
			"name":  "agent-" + string(rune('A'+i)),
			"image": "nginx",
		}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.Create(rec, req)
	}
	
	// Set first agent to running
	var resp map[string]interface{}
	// Get first agent from list
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	listRec := httptest.NewRecorder()
	handler.List(listRec, listReq)
	json.Unmarshal(listRec.Body.Bytes(), &resp)
	agents := resp["data"].([]interface{})
	firstID := agents[0].(map[string]interface{})["id"].(string)
	
	agent, _ := handler.repo.Get(nil, firstID)
	agent.Status = repository.AgentStatusRunning
	handler.repo.Update(nil, agent)
	
	// Filter by running status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?status=running", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestListAgents_FilterByName(t *testing.T) {
	handler := setupTestHandler(t)
	
	createReq := map[string]interface{}{"name": "my-special-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	// Create another agent
	createReq = map[string]interface{}{"name": "other-agent", "image": "nginx"}
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.Create(rec, req)
	
	// Filter by name
	req = httptest.NewRequest(http.MethodGet, "/api/v1/agents?name=special", nil)
	rec = httptest.NewRecorder()
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestCreateAgent_NameTooLong(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create agent with name > 63 chars
	longName := ""
	for i := 0; i < 64; i++ {
		longName += "x"
	}
	
	reqBody := map[string]interface{}{
		"name":  longName,
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	handler.Create(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAgent_DefaultRuntime(t *testing.T) {
	handler := setupTestHandler(t)
	
	reqBody := map[string]interface{}{
		"name":  "default-runtime-agent",
		"image": "nginx:latest",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	handler.Create(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
	
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "containerd", data["runtime"])
}

func TestCreateAgent_WithLabels(t *testing.T) {
	handler := setupTestHandler(t)
	
	reqBody := map[string]interface{}{
		"name":  "labeled-agent",
		"image": "nginx:latest",
		"labels": map[string]string{
			"env": "production",
			"team": "backend",
		},
		"environment": map[string]string{
			"PORT": "8080",
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()
	
	handler.Create(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestListAgents_Page2(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create 5 agents
	for i := 0; i < 5; i++ {
		createReq := map[string]interface{}{
			"name":  "agent-" + string(rune('A'+i)),
			"image": "nginx",
		}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.Create(rec, req)
	}
	
	// Get page 2 with size 2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?page=2&page_size=2", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(3), meta["total_pages"])
}

func TestListAgents_InvalidPage(t *testing.T) {
	handler := setupTestHandler(t)
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?page=-1&page_size=abc", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateAgent_WithEnvironment(t *testing.T) {
	handler := setupTestHandler(t)
	
	// Create agent
	createReq := map[string]interface{}{"name": "env-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	agentID := resp["data"].(map[string]interface{})["id"].(string)
	
	// Update with environment
	updateReq := map[string]interface{}{
		"environment": map[string]string{"KEY": "value"},
	}
	updateBody, _ := json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+agentID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.Update(rec, req)
	
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStartAgent_AlreadyRunning(t *testing.T) {
	handler := setupTestHandler(t)
	
	createReq := map[string]interface{}{"name": "running-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	agentID := resp["data"].(map[string]interface{})["id"].(string)
	
	agent, _ := handler.repo.Get(nil, agentID)
	agent.Status = repository.AgentStatusRunning
	handler.repo.Update(nil, agent)
	
	// Try to start already running agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/start", nil)
	rec = httptest.NewRecorder()
	handler.Start(rec, req)
	
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestStopAgent_NotRunning(t *testing.T) {
	handler := setupTestHandler(t)
	
	createReq := map[string]interface{}{"name": "stopped-agent", "image": "nginx"}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	agentID := resp["data"].(map[string]interface{})["id"].(string)
	
	// Try to stop a pending agent
	req = httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/stop", nil)
	rec = httptest.NewRecorder()
	handler.Stop(rec, req)
	
	assert.Equal(t, http.StatusConflict, rec.Code)
}
