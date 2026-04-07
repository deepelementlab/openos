package agent

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/data/repository"
	"go.uber.org/zap"
)

// AgentHandler handles agent-related HTTP requests.
// It wraps the AgentRepository and provides clean HTTP handler methods
// that can be registered with a router.
type AgentHandler struct {
	repo   repository.AgentRepository
	logger *zap.Logger
}

func NewAgentHandler(repo repository.AgentRepository, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{repo: repo, logger: logger}
}

// List returns all agents.
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.repo.List(r.Context())
	if err != nil {
		h.logger.Error("failed to list agents", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list agents")
		return
	}
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agents))
}

// Create creates a new agent.
func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string            `json:"name"`
		Image     string            `json:"image"`
		Runtime   string            `json:"runtime,omitempty"`
		Resources map[string]string `json:"resources,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Image) == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name and image are required")
		return
	}

	now := time.Now().UTC()
	agent := &repository.Agent{
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   req.Runtime,
		Resources: req.Resources,
		Status:    repository.AgentStatusPending,
		Metadata:  map[string]interface{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(r.Context(), agent); err != nil {
		h.logger.Error("failed to create agent", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "CREATE_FAILED", "failed to create agent")
		return
	}
	h.writeJSON(w, http.StatusCreated, models.SuccessResponse(agent))
}

// Get returns a single agent by ID.
func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Update updates an existing agent.
func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}

	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
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
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
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

	if err := h.repo.Update(r.Context(), agent); err != nil {
		h.logger.Error("failed to update agent", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update agent")
		return
	}
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Delete removes an agent.
func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *AgentHandler) writeError(w http.ResponseWriter, code int, errCode, msg string) {
	h.writeJSON(w, code, models.ErrorResponse(errCode, msg))
}
