package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/internal/monitoring"
	"github.com/agentos/aos/internal/security"
	"go.uber.org/zap"
)

// AgentAPIServer provides the full Agent management HTTP API.
type AgentAPIServer struct {
	logger     *zap.Logger
	metrics    *monitoring.Metrics
	agentRepo  repository.AgentRepository
	sandboxMgr security.SandboxManager
	networkMgr security.NetworkPolicyManager
}

// NewAgentAPIServer creates a new agent API server.
func NewAgentAPIServer(
	logger *zap.Logger,
	metrics *monitoring.Metrics,
	agentRepo repository.AgentRepository,
	sandboxMgr security.SandboxManager,
	networkMgr security.NetworkPolicyManager,
) *AgentAPIServer {
	return &AgentAPIServer{
		logger:     logger,
		metrics:    metrics,
		agentRepo:  agentRepo,
		sandboxMgr: sandboxMgr,
		networkMgr: networkMgr,
	}
}

// RegisterRoutes registers all agent API routes.
func (s *AgentAPIServer) RegisterRoutes(mux *http.ServeMux) {
	// Agent CRUD
	mux.HandleFunc("/api/v1/agents", s.handleAgents)
	mux.HandleFunc("/api/v1/agents/", s.handleAgentByID)

	// Security
	mux.HandleFunc("/api/v1/security/sandboxes", s.handleSandboxList)
	mux.HandleFunc("/api/v1/security/sandboxes/", s.handleSandboxOps)
	mux.HandleFunc("/api/v1/security/network-policies", s.handleNetworkPolicyList)
	mux.HandleFunc("/api/v1/security/network-policies/", s.handleNetworkPolicyOps)
}

func (s *AgentAPIServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAgents(w, r)
	case http.MethodPost:
		s.createAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AgentAPIServer) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Lifecycle actions
	if strings.HasSuffix(path, "/start") && r.Method == http.MethodPost {
		s.startAgent(w, r)
		return
	}
	if strings.HasSuffix(path, "/stop") && r.Method == http.MethodPost {
		s.stopAgent(w, r)
		return
	}
	if strings.HasSuffix(path, "/restart") && r.Method == http.MethodPost {
		s.restartAgent(w, r)
		return
	}
	if strings.HasSuffix(path, "/stats") && r.Method == http.MethodGet {
		s.getAgentStats(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getAgent(w, r)
	case http.MethodPut:
		s.updateAgent(w, r)
	case http.MethodDelete:
		s.deleteAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AgentAPIServer) listAgents(w http.ResponseWriter, r *http.Request) {
	s.metrics.IncAPIRequest("GET", "/api/v1/agents", 200)

	query := r.URL.Query()
	page := parseQueryInt(query.Get("page"), 1)
	pageSize := parseQueryInt(query.Get("page_size"), 20)
	statusFilter := query.Get("status")
	nameFilter := query.Get("name")

	agents, err := s.agentRepo.List(r.Context())
	if err != nil {
		s.metrics.IncAPIError("GET", "/api/v1/agents", "LIST_FAILED")
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("LIST_FAILED", err.Error()))
		return
	}

	filtered := filterAgentsList(agents, statusFilter, nameFilter)

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	meta := models.CalculatePaginationMeta(page, pageSize, total)
	writeAPIJSON(w, http.StatusOK, models.SuccessResponseWithMeta(filtered[start:end], meta))
}

func (s *AgentAPIServer) createAgent(w http.ResponseWriter, r *http.Request) {
	s.metrics.IncAPIRequest("POST", "/api/v1/agents", 201)
	startTime := time.Now()

	var req models.CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("BAD_REQUEST", "invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("VALIDATION_FAILED", err.Error()))
		return
	}

	now := time.Now().UTC()
	runtime := req.Runtime
	if runtime == "" {
		runtime = "containerd"
	}

	agent := &repository.Agent{
		ID:        fmt.Sprintf("agent-%s", generateShortID()),
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   runtime,
		Resources: req.Resources,
		Status:    repository.AgentStatusPending,
		Metadata: map[string]interface{}{
			"environment": req.Environment,
			"labels":      req.Labels,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.agentRepo.Create(r.Context(), agent); err != nil {
		s.metrics.IncAPIError("POST", "/api/v1/agents", "CREATE_FAILED")
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("CREATE_FAILED", err.Error()))
		return
	}

	// Create sandbox for the agent
	sandboxConfig := &security.SandboxConfig{
		AgentID:        agent.ID,
		Type:           security.SandboxType(runtime),
		ReadOnlyRootFS: true,
		RunAsUser:      1000,
	}
	if req.Resources != nil {
		sandboxConfig.ResourceLimits = &security.ResourceLimits{
			CPULimit:    parseResourceInt(req.Resources["cpu"], 1000),
			MemoryLimit: parseResourceInt(req.Resources["memory"], 512*1024*1024),
			PidsLimit:   100,
		}
	}

	sandbox, err := s.sandboxMgr.CreateSandbox(r.Context(), sandboxConfig)
	if err != nil {
		s.logger.Warn("failed to create sandbox for agent", zap.Error(err))
	} else {
		s.logger.Info("sandbox created for agent", zap.String("agentId", agent.ID), zap.String("sandboxId", sandbox.ID))
	}

	// Create default network policy
	netPolicy := &security.NetworkPolicy{
		AgentID:        agent.ID,
		Name:           "default-" + agent.ID,
		DefaultIngress: security.NetActionDeny,
		DefaultEgress:  security.NetActionAllow,
		Rules: []security.NetworkRule{
			{ID: "dns-egress", Direction: "egress", Protocol: security.ProtocolUDP, FromPort: 53, ToPort: 53, Action: security.NetActionAllow},
			{ID: "http-egress", Direction: "egress", Protocol: security.ProtocolTCP, FromPort: 80, ToPort: 80, Action: security.NetActionAllow},
			{ID: "https-egress", Direction: "egress", Protocol: security.ProtocolTCP, FromPort: 443, ToPort: 443, Action: security.NetActionAllow},
		},
	}
	if err := s.networkMgr.CreatePolicy(r.Context(), netPolicy); err != nil {
		s.logger.Warn("failed to create network policy for agent", zap.Error(err))
	}

	s.metrics.IncAgentCreated()
	s.metrics.ObserveAgentCreationDuration(time.Since(startTime).Seconds())

	w.Header().Set("Location", "/api/v1/agents/"+agent.ID)
	writeAPIJSON(w, http.StatusCreated, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) getAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(r.URL.Path)
	if id == "" {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("MISSING_ID", "agent id is required"))
		return
	}

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) updateAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(r.URL.Path)
	if id == "" {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("MISSING_ID", "agent id is required"))
		return
	}

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	var req models.UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("BAD_REQUEST", "invalid request body"))
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
	if req.Resources != nil {
		agent.Resources = req.Resources
	}
	agent.UpdatedAt = time.Now().UTC()

	if err := s.agentRepo.Update(r.Context(), agent); err != nil {
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("UPDATE_FAILED", err.Error()))
		return
	}

	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(r.URL.Path)

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	if agent.Status == repository.AgentStatusRunning {
		writeAPIJSON(w, http.StatusConflict, models.ErrorResponse("INVALID_STATE", "cannot delete running agent, stop it first"))
		return
	}

	// Cleanup sandbox
	sandboxes, _ := s.sandboxMgr.ListSandboxes(r.Context())
	for _, sb := range sandboxes {
		if sb.AgentID == id && sb.Status == "running" {
			s.sandboxMgr.DestroySandbox(r.Context(), sb.ID)
		}
	}

	// Cleanup network policy
	s.networkMgr.DeletePolicy(r.Context(), id)

	if err := s.agentRepo.Delete(r.Context(), id); err != nil {
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("DELETE_FAILED", err.Error()))
		return
	}

	s.metrics.IncAgentDeleted()
	w.WriteHeader(http.StatusNoContent)
}

func (s *AgentAPIServer) startAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(strings.TrimSuffix(r.URL.Path, "/start"))

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	if agent.Status != repository.AgentStatusPending && agent.Status != repository.AgentStatusStopped {
		writeAPIJSON(w, http.StatusConflict, models.ErrorResponse("INVALID_STATE",
			fmt.Sprintf("cannot start agent in %s state", agent.Status)))
		return
	}

	agent.Status = repository.AgentStatusRunning
	agent.UpdatedAt = time.Now().UTC()
	s.agentRepo.Update(r.Context(), agent)

	s.metrics.IncAgentStarted()
	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) stopAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(strings.TrimSuffix(r.URL.Path, "/stop"))

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	if agent.Status != repository.AgentStatusRunning {
		writeAPIJSON(w, http.StatusConflict, models.ErrorResponse("INVALID_STATE",
			fmt.Sprintf("cannot stop agent in %s state", agent.Status)))
		return
	}

	agent.Status = repository.AgentStatusStopped
	agent.UpdatedAt = time.Now().UTC()
	s.agentRepo.Update(r.Context(), agent)

	s.metrics.IncAgentStopped()
	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) restartAgent(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(strings.TrimSuffix(r.URL.Path, "/restart"))

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	agent.Status = repository.AgentStatusRunning
	agent.UpdatedAt = time.Now().UTC()
	s.agentRepo.Update(r.Context(), agent)

	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(agent))
}

func (s *AgentAPIServer) getAgentStats(w http.ResponseWriter, r *http.Request) {
	id := extractAgentID(strings.TrimSuffix(r.URL.Path, "/stats"))

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "agent not found"))
		return
	}

	stats := map[string]interface{}{
		"agentId":   agent.ID,
		"status":    agent.Status,
		"uptime":    time.Since(agent.UpdatedAt).Seconds(),
		"resources": agent.Resources,
	}

	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(stats))
}

// Security endpoints

func (s *AgentAPIServer) handleSandboxList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sandboxes, err := s.sandboxMgr.ListSandboxes(r.Context())
	if err != nil {
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("LIST_FAILED", err.Error()))
		return
	}
	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(sandboxes))
}

func (s *AgentAPIServer) handleSandboxOps(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/api/v1/security/sandboxes/")

	if id == "" {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("MISSING_ID", "sandbox id is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		state, err := s.sandboxMgr.GetSandboxState(r.Context(), id)
		if err != nil {
			writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
			return
		}
		writeAPIJSON(w, http.StatusOK, models.SuccessResponse(state))

	case http.MethodDelete:
		if err := s.sandboxMgr.DestroySandbox(r.Context(), id); err != nil {
			writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AgentAPIServer) handleNetworkPolicyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	policies, err := s.networkMgr.ListPolicies(r.Context())
	if err != nil {
		writeAPIJSON(w, http.StatusInternalServerError, models.ErrorResponse("LIST_FAILED", err.Error()))
		return
	}
	writeAPIJSON(w, http.StatusOK, models.SuccessResponse(policies))
}

func (s *AgentAPIServer) handleNetworkPolicyOps(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	agentID := strings.TrimPrefix(path, "/api/v1/security/network-policies/")

	if agentID == "" {
		writeAPIJSON(w, http.StatusBadRequest, models.ErrorResponse("MISSING_ID", "agent id is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		policy, err := s.networkMgr.GetPolicy(r.Context(), agentID)
		if err != nil {
			writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
			return
		}
		writeAPIJSON(w, http.StatusOK, models.SuccessResponse(policy))

	case http.MethodDelete:
		if err := s.networkMgr.DeletePolicy(r.Context(), agentID); err != nil {
			writeAPIJSON(w, http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper functions

func extractAgentID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/agents/"), "/")
	return parts[0]
}

func parseQueryInt(param string, defaultValue int) int {
	val, err := strconv.Atoi(param)
	if err != nil || val < 1 {
		return defaultValue
	}
	return val
}

func parseResourceInt(s string, defaultValue int64) int64 {
	if s == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

func filterAgentsList(agents []*repository.Agent, status, name string) []*repository.Agent {
	if status == "" && name == "" {
		return agents
	}

	filtered := make([]*repository.Agent, 0)
	for _, a := range agents {
		if status != "" && string(a.Status) != status {
			continue
		}
		if name != "" && !strings.Contains(strings.ToLower(a.Name), strings.ToLower(name)) {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}

func generateShortID() string {
	return strings.ReplaceAll(generateUUID(), "-", "")[:12]
}

func generateUUID() string {
	// Simple UUID v4 generation without external dependency
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func writeAPIJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
