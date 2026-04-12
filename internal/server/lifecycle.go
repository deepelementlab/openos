package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agentos/aos/internal/data/repository"
	"go.uber.org/zap"
)

// handleAgentRoutes dispatches agent lifecycle operations.
// Routes:
//   GET    /api/v1/agents/{id}       → getAgent
//   PUT    /api/v1/agents/{id}       → updateAgent
//   DELETE /api/v1/agents/{id}       → deleteAgent
//   POST   /api/v1/agents/{id}/start → startAgent
//   POST   /api/v1/agents/{id}/stop  → stopAgent
//   POST   /api/v1/agents/{id}/restart → restartAgent
func (s *Server) handleAgentRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/api/v1/agents/")
	if id == "" {
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}

	// Check for lifecycle action suffixes
	switch {
	case strings.HasSuffix(id, "/start"):
		id = strings.TrimSuffix(id, "/start")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.startAgent(w, r, id)
		return
	case strings.HasSuffix(id, "/stop"):
		id = strings.TrimSuffix(id, "/stop")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.stopAgent(w, r, id)
		return
	case strings.HasSuffix(id, "/restart"):
		id = strings.TrimSuffix(id, "/restart")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.restartAgent(w, r, id)
		return
	}

	// Regular CRUD
	switch r.Method {
	case http.MethodGet:
		s.getAgentByID(w, r, id)
	case http.MethodPut:
		s.updateAgentByID(w, r, id)
	case http.MethodDelete:
		s.deleteAgentByID(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// startAgent transitions an agent from pending/stopped to creating state.
func (s *Server) startAgent(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/start", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Validate state transition
	if agent.Status != repository.AgentStatusPending &&
		agent.Status != repository.AgentStatusStopped &&
		agent.Status != repository.AgentStatusError {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/start", "invalid_state")
		writeJSONError(w, http.StatusConflict,
			fmt.Sprintf("cannot start agent in %s state", agent.Status))
		return
	}

	agent.Status = repository.AgentStatusCreating
	agent.UpdatedAt = time.Now().UTC()

	if err := s.agentRepo.Update(r.Context(), agent); err != nil {
		s.logger.Error("failed to start agent", zap.Error(err), zap.String("id", id))
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/start", "update_failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to start agent")
		return
	}

	s.logger.Info("agent start initiated", zap.String("id", id))
	s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/agents/{id}/start", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    agent,
	})
}

// stopAgent transitions an agent from running to stopping state.
func (s *Server) stopAgent(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/stop", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	if agent.Status != repository.AgentStatusRunning &&
		agent.Status != repository.AgentStatusCreating {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/stop", "invalid_state")
		writeJSONError(w, http.StatusConflict,
			fmt.Sprintf("cannot stop agent in %s state", agent.Status))
		return
	}

	agent.Status = repository.AgentStatusStopping
	agent.UpdatedAt = time.Now().UTC()

	if err := s.agentRepo.Update(r.Context(), agent); err != nil {
		s.logger.Error("failed to stop agent", zap.Error(err), zap.String("id", id))
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/stop", "update_failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to stop agent")
		return
	}

	s.logger.Info("agent stop initiated", zap.String("id", id))
	s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/agents/{id}/stop", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    agent,
	})
}

// restartAgent transitions an agent from running/error to stopping state (for restart).
func (s *Server) restartAgent(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/restart", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	if agent.Status != repository.AgentStatusRunning &&
		agent.Status != repository.AgentStatusError {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/restart", "invalid_state")
		writeJSONError(w, http.StatusConflict,
			fmt.Sprintf("cannot restart agent in %s state", agent.Status))
		return
	}

	agent.Status = repository.AgentStatusStopping
	agent.UpdatedAt = time.Now().UTC()

	if err := s.agentRepo.Update(r.Context(), agent); err != nil {
		s.logger.Error("failed to restart agent", zap.Error(err), zap.String("id", id))
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents/{id}/restart", "update_failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to restart agent")
		return
	}

	s.logger.Info("agent restart initiated", zap.String("id", id))
	s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/agents/{id}/restart", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    agent,
	})
}

// getAgentByID retrieves an agent by ID.
func (s *Server) getAgentByID(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodGet, "/api/v1/agents/{id}", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}
	s.metrics.IncAPIRequest(http.MethodGet, "/api/v1/agents/{id}", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    agent,
	})
}

// deleteAgentByID deletes an agent (must not be running).
func (s *Server) deleteAgentByID(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodDelete, "/api/v1/agents/{id}", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Cannot delete running agent
	if agent.Status == repository.AgentStatusRunning {
		s.metrics.IncAPIError(http.MethodDelete, "/api/v1/agents/{id}", "invalid_state")
		writeJSONError(w, http.StatusConflict, "cannot delete running agent, stop it first")
		return
	}

	if err := s.agentRepo.Delete(r.Context(), id); err != nil {
		s.metrics.IncAPIError(http.MethodDelete, "/api/v1/agents/{id}", "delete_failed")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	s.metrics.IncAgentDeleted()
	s.metrics.IncAPIRequest(http.MethodDelete, "/api/v1/agents/{id}", http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// updateAgentByID updates an agent by ID.
func (s *Server) updateAgentByID(w http.ResponseWriter, r *http.Request, id string) {
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/{id}", "not_found")
		writeJSONError(w, http.StatusNotFound, "agent not found")
		return
	}

	var req struct {
		Name      string                 `json:"name,omitempty"`
		Image     string                 `json:"image,omitempty"`
		Runtime   string                 `json:"runtime,omitempty"`
		Status    repository.AgentStatus `json:"status,omitempty"`
		Resources map[string]string      `json:"resources,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/{id}", "bad_request")
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Image != "" {
		agent.Image = req.Image
	}
	if req.Runtime != "" {
		agent.Runtime = req.Runtime
	}
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.Resources != nil {
		agent.Resources = req.Resources
	}
	agent.UpdatedAt = time.Now().UTC()

	if err := s.agentRepo.Update(r.Context(), agent); err != nil {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/{id}", "update_failed")
		writeJSONError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}

	s.metrics.IncAPIRequest(http.MethodPut, "/api/v1/agents/{id}", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    agent,
	})
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    http.StatusText(code),
			"message": message,
		},
	})
}
