package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/data/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	repo   repository.AgentRepository
	logger *zap.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(repo repository.AgentRepository, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{repo: repo, logger: logger}
}

// List returns all agents with optional filtering and pagination.
// Query parameters:
//   - page: page number (default: 1)
//   - page_size: items per page (default: 20, max: 100)
//   - status: filter by status (pending, running, stopped, error)
//   - name: filter by name (case-insensitive substring match)
//   - label: filter by label (format: key or key=value)
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	
	// Pagination
	page := parseIntParam(query.Get("page"), 1)
	pageSize := parseIntParam(query.Get("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100 // Max page size
	}
	
	// Filters
	statusFilter := query.Get("status")
	nameFilter := query.Get("name")
	labelFilter := query.Get("label")
	
	agents, err := h.repo.List(r.Context())
	if err != nil {
		h.logger.Error("failed to list agents", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list agents")
		return
	}
	
	// Apply filters
	filtered := h.filterAgents(agents, statusFilter, nameFilter, labelFilter)
	
	// Apply pagination
	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	
	pagedAgents := filtered[start:end]
	
	meta := models.CalculatePaginationMeta(page, pageSize, total)
	h.writeJSON(w, http.StatusOK, models.SuccessResponseWithMeta(pagedAgents, meta))
}

// Create creates a new agent.
// Request body:
//   - name: agent name (required, max 63 chars)
//   - image: container image (required)
//   - runtime: container runtime (optional, default: containerd)
//   - resources: resource limits (optional)
//   - environment: environment variables (optional)
//   - labels: labels for organization (optional)
func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string            `json:"name"`
		Image       string            `json:"image"`
		Runtime     string            `json:"runtime,omitempty"`
		Resources   map[string]string `json:"resources,omitempty"`
		Environment map[string]string `json:"environment,omitempty"`
		Labels      map[string]string `json:"labels,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	
	// Validation
	if strings.TrimSpace(req.Name) == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name is required")
		return
	}
	if strings.TrimSpace(req.Image) == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "image is required")
		return
	}
	if len(req.Name) > 63 {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name must be 63 characters or less")
		return
	}
	
	// Set default runtime if not specified
	if req.Runtime == "" {
		req.Runtime = "containerd"
	}

	// Generate unique ID
	id := generateAgentID()
	now := time.Now().UTC()
	
	agent := &repository.Agent{
		ID:        id,
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   req.Runtime,
		Resources: req.Resources,
		Status:    repository.AgentStatusPending,
		Metadata: map[string]interface{}{
			"environment": req.Environment,
			"labels":      req.Labels,
			"created_by":  r.Header.Get("X-User-ID"),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(r.Context(), agent); err != nil {
		h.logger.Error("failed to create agent", zap.Error(err), zap.String("name", req.Name))
		h.writeError(w, http.StatusInternalServerError, "CREATE_FAILED", "failed to create agent")
		return
	}
	
	h.logger.Info("agent created successfully", 
		zap.String("id", agent.ID), 
		zap.String("name", agent.Name),
		zap.String("status", string(agent.Status)))
	
	w.Header().Set("Location", "/api/v1/agents/"+agent.ID)
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
// Supports partial updates - only provided fields will be updated.
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
		Name        string                 `json:"name,omitempty"`
		Image       string                 `json:"image,omitempty"`
		Runtime     string                 `json:"runtime,omitempty"`
		Status      repository.AgentStatus `json:"status,omitempty"`
		Resources   map[string]string      `json:"resources,omitempty"`
		Environment map[string]string      `json:"environment,omitempty"`
		Labels      map[string]string      `json:"labels,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	// Apply updates
	if req.Name != "" {
		if len(req.Name) > 63 {
			h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name must be 63 characters or less")
			return
		}
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
	
	// Update metadata
	if req.Environment != nil || req.Labels != nil {
		metadata := agent.Metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		if req.Environment != nil {
			metadata["environment"] = req.Environment
		}
		if req.Labels != nil {
			metadata["labels"] = req.Labels
		}
		agent.Metadata = metadata
	}
	
	agent.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(r.Context(), agent); err != nil {
		h.logger.Error("failed to update agent", zap.Error(err), zap.String("id", id))
		h.writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update agent")
		return
	}
	
	h.logger.Info("agent updated successfully", zap.String("id", id))
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Delete removes an agent.
// Agent must be in stopped or error state to be deleted.
func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	
	// Check if agent exists and can be deleted
	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	
	// Validate state
	if agent.Status == repository.AgentStatusRunning {
		h.writeError(w, http.StatusConflict, "INVALID_STATE", 
			"cannot delete running agent, stop it first")
		return
	}
	
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	
	h.logger.Info("agent deleted successfully", zap.String("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// Start starts an agent.
// Agent must be in pending or stopped state.
func (h *AgentHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	id = strings.TrimSuffix(id, "/start")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	
	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	
	// Validate state transition
	if agent.Status != repository.AgentStatusPending && agent.Status != repository.AgentStatusStopped {
		h.writeError(w, http.StatusConflict, "INVALID_STATE", 
			fmt.Sprintf("cannot start agent in %s state", agent.Status))
		return
	}
	
	agent.Status = repository.AgentStatusCreating
	agent.UpdatedAt = time.Now().UTC()
	
	if err := h.repo.Update(r.Context(), agent); err != nil {
		h.logger.Error("failed to start agent", zap.Error(err), zap.String("id", id))
		h.writeError(w, http.StatusInternalServerError, "START_FAILED", "failed to start agent")
		return
	}
	
	h.logger.Info("agent start initiated", zap.String("id", id))
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Stop stops an agent.
// Agent must be in running state.
func (h *AgentHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	id = strings.TrimSuffix(id, "/stop")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	
	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	
	// Validate state transition
	if agent.Status != repository.AgentStatusRunning {
		h.writeError(w, http.StatusConflict, "INVALID_STATE", 
			fmt.Sprintf("cannot stop agent in %s state", agent.Status))
		return
	}
	
	agent.Status = repository.AgentStatusStopping
	agent.UpdatedAt = time.Now().UTC()
	
	if err := h.repo.Update(r.Context(), agent); err != nil {
		h.logger.Error("failed to stop agent", zap.Error(err), zap.String("id", id))
		h.writeError(w, http.StatusInternalServerError, "STOP_FAILED", "failed to stop agent")
		return
	}
	
	h.logger.Info("agent stop initiated", zap.String("id", id))
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Restart restarts an agent.
// Agent must be in running or error state.
func (h *AgentHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	id = strings.TrimSuffix(id, "/restart")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "agent id is required")
		return
	}
	
	agent, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	
	// Can restart from running or error states
	if agent.Status != repository.AgentStatusRunning && agent.Status != repository.AgentStatusError {
		h.writeError(w, http.StatusConflict, "INVALID_STATE", 
			fmt.Sprintf("cannot restart agent in %s state", agent.Status))
		return
	}
	
	agent.Status = repository.AgentStatusStopping
	agent.UpdatedAt = time.Now().UTC()
	
	if err := h.repo.Update(r.Context(), agent); err != nil {
		h.logger.Error("failed to restart agent", zap.Error(err), zap.String("id", id))
		h.writeError(w, http.StatusInternalServerError, "RESTART_FAILED", "failed to restart agent")
		return
	}
	
	h.logger.Info("agent restart initiated", zap.String("id", id))
	h.writeJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

// Helper functions

func (h *AgentHandler) filterAgents(agents []*repository.Agent, status, name, label string) []*repository.Agent {
	if status == "" && name == "" && label == "" {
		return agents
	}
	
	filtered := make([]*repository.Agent, 0)
	for _, agent := range agents {
		// Status filter
		if status != "" && string(agent.Status) != status {
			continue
		}
		// Name filter (case-insensitive substring match)
		if name != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(name)) {
			continue
		}
		// Label filter
		if label != "" {
			labels, ok := agent.Metadata["labels"].(map[string]string)
			if !ok {
				continue
			}
			found := false
			for k, v := range labels {
				if k+"="+v == label || k == label {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, agent)
	}
	return filtered
}

func generateAgentID() string {
	return fmt.Sprintf("agent-%s", uuid.New().String()[:8])
}

func parseIntParam(param string, defaultValue int) int {
	if param == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(param)
	if err != nil || val < 1 {
		return defaultValue
	}
	return val
}

func (h *AgentHandler) writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *AgentHandler) writeError(w http.ResponseWriter, code int, errCode, msg string) {
	h.writeJSON(w, code, models.ErrorResponse(errCode, msg))
}
