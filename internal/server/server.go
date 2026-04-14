// Package server implements the main HTTP server for Agent OS.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agentos/aos/api/middleware"
	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/database"
	"github.com/agentos/aos/internal/data/outbox"
	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/internal/data/transaction"
	"github.com/agentos/aos/internal/health"
	"github.com/agentos/aos/internal/kernel"
	"github.com/agentos/aos/internal/monitoring"
	"github.com/agentos/aos/internal/orchestration"
	"github.com/agentos/aos/internal/orchestration/workflow"
	"github.com/agentos/aos/internal/scheduler"
	"github.com/agentos/aos/internal/version"
	runtimefacade "github.com/agentos/aos/pkg/runtime/facade"
	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Server represents the Agent OS HTTP server.
type Server struct {
	config             *config.Config
	logger             *zap.Logger
	server             *http.Server
	metrics            *monitoring.Metrics
	healthChecker      *health.Checker
	scheduler          scheduler.Scheduler
	agentRepo          repository.AgentRepository
	uow                transaction.UnitOfWork
	outbox             outbox.Publisher
	workflowEngine     *workflow.WorkflowEngine
	stateMachineEngine *orchestration.StateMachineEngine
	healthStatus       string
	startTime          time.Time
	kernel             *kernel.Facade
	runtimeFacade      *runtimefacade.RuntimeFacade
}

// NewServer creates a new server instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	agentRepo := repository.AgentRepository(repository.NewInMemoryAgentRepository())
	unitOfWork := transaction.UnitOfWork(transaction.NewNoopUnitOfWork())
	outboxPublisher := outbox.Publisher(outbox.NewInMemoryPublisher())
	if dbRepo, dbUoW, dbOutbox, err := initPostgresDataComponents(cfg, logger); err == nil {
		agentRepo = dbRepo
		unitOfWork = dbUoW
		outboxPublisher = dbOutbox
		logger.Info("using PostgreSQL data components (repo+uow+outbox)")
	} else {
		logger.Warn("falling back to in-memory data components", zap.Error(err))
	}

	checker := health.NewChecker()
	checker.Register("server", "HTTP server", func(_ context.Context) error { return nil })

	sched := scheduler.NewDefaultScheduler()
	_ = sched.AddNode(context.Background(), scheduler.NodeState{
		NodeID:      "node-local-1",
		NodeName:    "local",
		CPUCores:    4,
		MemoryBytes: 8 * 1024 * 1024 * 1024,
		Health:      "healthy",
	})

	k := kernel.NewDefaultFacade()
	rf := runtimefacade.NewRuntimeFacade(runtimefacade.WithKernel(k))
	rtCfg := &types.RuntimeConfig{}
	if err := rf.Connect(context.Background(), runtimefacade.BackendGVisor, rtCfg); err != nil {
		logger.Warn("runtime facade not connected (container runtime optional)", zap.Error(err))
	}

	s := &Server{
		config:        cfg,
		logger:        logger,
		healthChecker: checker,
		scheduler:     sched,
		agentRepo:     agentRepo,
		uow:           unitOfWork,
		outbox:        outboxPublisher,
		healthStatus:  "healthy",
		startTime:     time.Now(),
		kernel:        k,
		runtimeFacade: rf,
	}

	// Initialize orchestration engines
	s.initializeOrchestration()

	// Initialize metrics
	metrics, err := monitoring.NewMetrics(&cfg.Monitoring)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}
	s.metrics = metrics

	// Create HTTP server with middleware chain
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	var handler http.Handler = mux

	// Wrap with logging middleware (always enabled)
	loggingMw := middleware.NewLoggingMiddleware(logger)
	handler = loggingMw.Handler(handler)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Health endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/live", s.handleLive)

	// API endpoints
	mux.HandleFunc("/api/v1/agents", s.handleAgents)
	mux.HandleFunc("/api/v1/agents/", s.handleAgent)

	// Workflow endpoints
	mux.HandleFunc("/api/v1/workflows", s.handleWorkflows)
	mux.HandleFunc("/api/v1/workflows/", s.handleWorkflow)
	mux.HandleFunc("/api/v1/workflows//start", s.handleStartWorkflow)
	mux.HandleFunc("/api/v1/instances/", s.handleWorkflowInstance)

	// State machine endpoints
	mux.HandleFunc("/api/v1/state-machines/", s.handleStateMachine)

	// Monitoring endpoints
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Root endpoint
	mux.HandleFunc("/", s.handleRoot)
}

// initializeOrchestration initializes the orchestration engines.
func (s *Server) initializeOrchestration() {
	// Initialize state machine engine
	stateMachineOpts := orchestration.StateMachineOptions{
		Persistence: nil, // Use in-memory for now
		Logger:      s.logger,
	}
	s.stateMachineEngine = orchestration.NewStateMachineEngine(stateMachineOpts)

	// Initialize workflow engine
	workflowOpts := workflow.EngineOptions{
		Registry:     nil, // Use default workflows
		StepRegistry: nil,
		Store:        nil, // Use in-memory store
		Logger:       s.logger,
	}
	s.workflowEngine = workflow.NewWorkflowEngine(workflowOpts)

	s.logger.Info("Orchestration engines initialized")
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server",
		zap.String("address", s.server.Addr),
		zap.String("mode", s.config.Server.Mode),
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", zap.Error(err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	s.healthStatus = "shutting_down"
	
	// Give some time for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.Server.GracefulShutdownTimeout)
	defer cancel()
	
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Graceful shutdown failed", zap.Error(err))
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	
	s.healthStatus = "stopped"
	s.logger.Info("Server shutdown complete")
	return nil
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.healthChecker.Check(r.Context())
	checkerStatus := s.healthChecker.Status()
	checkerStatus["version"] = version.GetVersion()
	checkerStatus["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := writeJSON(w, checkerStatus); err != nil {
		s.logger.Error("Failed to write health response", zap.Error(err))
	}
}

// handleReady handles readiness probe requests.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, always ready
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleLive handles liveness probe requests.
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, always live
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleMetrics handles metrics requests.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.metrics.GetMetrics()))
}

// handleRoot handles the root endpoint.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := map[string]interface{}{
		"name":        "Agent OS",
		"version":     version.GetVersion(),
		"description": "Operating System for AI Agents",
		"endpoints": map[string]string{
			"health":   "/health",
			"ready":    "/ready",
			"live":     "/live",
			"metrics":  "/metrics",
			"agents":   "/api/v1/agents",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := writeJSON(w, info); err != nil {
		s.logger.Error("Failed to write root response", zap.Error(err))
	}
}

// handleAgents handles agents collection endpoints.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAgents(w, r)
	case http.MethodPost:
		s.createAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAgent handles individual agent endpoints.
func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getAgent(w, r)
	case http.MethodDelete:
		s.deleteAgent(w, r)
	case http.MethodPut:
		s.updateAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listAgents lists all agents.
func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.agentRepo.List(r.Context())
	if err != nil {
		s.logger.Error("failed to list agents", zap.Error(err))
		s.metrics.IncAPIError(http.MethodGet, "/api/v1/agents", "list_failed")
		http.Error(w, "failed to list agents", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"agents": agents,
		"pagination": map[string]interface{}{
			"page":  1,
			"limit": 20,
			"total": len(agents),
		},
	}

	s.metrics.IncAPIRequest(http.MethodGet, "/api/v1/agents", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := writeJSON(w, response); err != nil {
		s.logger.Error("Failed to write agents list", zap.Error(err))
	}
}

// createAgent creates a new agent.
func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	createStart := time.Now()
	type createAgentRequest struct {
		Name      string            `json:"name"`
		Image     string            `json:"image"`
		Runtime   string            `json:"runtime,omitempty"`
		Resources map[string]string `json:"resources,omitempty"`
	}
	var req createAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents", "bad_request")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Image) == "" {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents", "validation_failed")
		http.Error(w, "name and image are required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	id := "agent-" + uuid.NewString()
	eventID := "evt-" + uuid.NewString()
	agent := &repository.Agent{
		ID:        id,
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   req.Runtime,
		Resources: req.Resources,
		Status:    repository.AgentStatusCreating,
		Metadata:  map[string]interface{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.uow.Do(r.Context(), func(ctx context.Context) error {
		if err := s.agentRepo.Create(ctx, agent); err != nil {
			return err
		}
		return s.outbox.Publish(ctx, outbox.Event{
			ID:        eventID,
			Type:      "agent.created",
			Payload:   map[string]interface{}{"agent_id": id, "name": req.Name},
			Status:    outbox.EventStatusPending,
			CreatedAt: now,
		})
	}); err != nil {
		s.logger.Error("failed to create agent", zap.Error(err))
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/agents", "create_failed")
		http.Error(w, "failed to create agent", http.StatusInternalServerError)
		return
	}

	if s.kernel != nil {
		ctx := r.Context()
		if g, err := s.kernel.Process.CreateGroup(ctx, id); err == nil && g != nil {
			agent.Metadata["kernel_group_id"] = g.GroupID
		} else if err != nil {
			s.logger.Debug("kernel process group skipped", zap.Error(err), zap.String("agent_id", id))
		}
		if ns, err := s.kernel.Process.CreateNamespace(ctx); err == nil && ns != nil {
			_ = s.kernel.Process.EnterNamespace(ctx, id, ns)
			agent.Metadata["kernel_namespace_id"] = ns.NamespaceID
		}
	}

	if nodeID, err := s.scheduleAgentBaseline(r.Context(), id); err == nil {
		agent.Metadata["scheduled_node"] = nodeID
		agent.UpdatedAt = time.Now().UTC()
		_ = s.agentRepo.Update(r.Context(), agent)
	} else {
		_ = s.outbox.DeadLetter(r.Context(), eventID)
		s.logger.Warn("baseline scheduling failed", zap.Error(err), zap.String("agent_id", id))
	}

	s.metrics.IncAgentCreated()
	s.metrics.ObserveAgentCreationDuration(time.Since(createStart).Seconds())
	s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/agents", http.StatusCreated)

	response := map[string]interface{}{
		"id":        agent.ID,
		"name":      agent.Name,
		"image":     agent.Image,
		"runtime":   agent.Runtime,
		"status":    agent.Status,
		"createdAt": agent.CreatedAt.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	
	if err := writeJSON(w, response); err != nil {
		s.logger.Error("Failed to write create agent response", zap.Error(err))
	}
}

// getAgent gets an agent by ID.
func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if strings.TrimSpace(id) == "" {
		s.metrics.IncAPIError(http.MethodGet, "/api/v1/agents/", "missing_id")
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}
	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodGet, "/api/v1/agents/", "not_found")
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	s.metrics.IncAPIRequest(http.MethodGet, "/api/v1/agents/", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := writeJSON(w, agent); err != nil {
		s.logger.Error("Failed to write get agent response", zap.Error(err))
	}
}

// deleteAgent deletes an agent.
func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if strings.TrimSpace(id) == "" {
		s.metrics.IncAPIError(http.MethodDelete, "/api/v1/agents/", "missing_id")
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}
	if err := s.agentRepo.Delete(r.Context(), id); err != nil {
		s.metrics.IncAPIError(http.MethodDelete, "/api/v1/agents/", "not_found")
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	s.metrics.IncAgentDeleted()
	s.metrics.IncAPIRequest(http.MethodDelete, "/api/v1/agents/", http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// updateAgent updates an agent.
func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if strings.TrimSpace(id) == "" {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/", "missing_id")
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}

	agent, err := s.agentRepo.Get(r.Context(), id)
	if err != nil {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/", "not_found")
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	type updateAgentRequest struct {
		Name      string                 `json:"name,omitempty"`
		Image     string                 `json:"image,omitempty"`
		Runtime   string                 `json:"runtime,omitempty"`
		Status    repository.AgentStatus `json:"status,omitempty"`
		Resources map[string]string      `json:"resources,omitempty"`
	}
	var req updateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/", "bad_request")
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
		s.metrics.IncAPIError(http.MethodPut, "/api/v1/agents/", "update_failed")
		http.Error(w, "failed to update agent", http.StatusInternalServerError)
		return
	}

	s.metrics.IncAPIRequest(http.MethodPut, "/api/v1/agents/", http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := writeJSON(w, agent); err != nil {
		s.logger.Error("Failed to write update agent response", zap.Error(err))
	}
}

// writeJSON writes JSON response.
func writeJSON(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}

// handleWorkflows handles workflow collection endpoints.
func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List available workflows
		workflows := s.workflowEngine.Stats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, workflows)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWorkflow handles individual workflow endpoints.
func (s *Server) handleWorkflow(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get workflow details
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, map[string]string{"status": "available"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStartWorkflow handles workflow start requests.
func (s *Server) handleStartWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type startWorkflowRequest struct {
		WorkflowID string                 `json:"workflow_id"`
		EntityID   string                 `json:"entity_id"`
		EntityType string                 `json:"entity_type"`
		Input      map[string]interface{} `json:"input,omitempty"`
	}

	var req startWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/workflows/start", "bad_request")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.WorkflowID == "" || req.EntityID == "" {
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/workflows/start", "validation_failed")
		http.Error(w, "workflow_id and entity_id are required", http.StatusBadRequest)
		return
	}

	instance, err := s.workflowEngine.StartWorkflow(r.Context(), req.WorkflowID, req.EntityID, req.EntityType, req.Input)
	if err != nil {
		s.logger.Error("failed to start workflow", zap.Error(err))
		s.metrics.IncAPIError(http.MethodPost, "/api/v1/workflows/start", "start_failed")
		http.Error(w, "failed to start workflow", http.StatusInternalServerError)
		return
	}

	s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/workflows/start", http.StatusAccepted)

	response := map[string]interface{}{
		"instance_id": instance.ID,
		"workflow_id": req.WorkflowID,
		"entity_id":   req.EntityID,
		"status":      instance.Status,
		"started_at":  instance.StartedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = writeJSON(w, response)
}

// handleWorkflowInstance handles workflow instance endpoints.
func (s *Server) handleWorkflowInstance(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/instances/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "instance ID is required", http.StatusBadRequest)
		return
	}

	instanceID := parts[0]

	switch r.Method {
	case http.MethodGet:
		// Get instance details
		instance, err := s.workflowEngine.GetInstance(r.Context(), instanceID)
		if err != nil {
			s.metrics.IncAPIError(http.MethodGet, "/api/v1/instances/", "not_found")
			http.Error(w, "instance not found", http.StatusNotFound)
			return
		}

		s.metrics.IncAPIRequest(http.MethodGet, "/api/v1/instances/", http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, instance)

	case http.MethodDelete:
		// Cancel workflow instance
		if err := s.workflowEngine.CancelWorkflow(r.Context(), instanceID); err != nil {
			s.metrics.IncAPIError(http.MethodDelete, "/api/v1/instances/", "cancel_failed")
			http.Error(w, "failed to cancel workflow", http.StatusInternalServerError)
			return
		}

		s.metrics.IncAPIRequest(http.MethodDelete, "/api/v1/instances/", http.StatusOK)
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, map[string]string{"status": "cancelled"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStateMachine handles state machine endpoints.
func (s *Server) handleStateMachine(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/state-machines/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "entity ID is required", http.StatusBadRequest)
		return
	}

	entityID := parts[0]

	switch r.Method {
	case http.MethodGet:
		// Get state machine status
		summary, err := s.stateMachineEngine.GetStateSummary(entityID)
		if err != nil {
			s.metrics.IncAPIError(http.MethodGet, "/api/v1/state-machines/", "not_found")
			http.Error(w, "state machine not found", http.StatusNotFound)
			return
		}

		s.metrics.IncAPIRequest(http.MethodGet, "/api/v1/state-machines/", http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, summary)

	case http.MethodPost:
		// Send event to state machine
		type sendEventRequest struct {
			Event string                 `json:"event"`
			Data  map[string]interface{} `json:"data,omitempty"`
		}

		var req sendEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.metrics.IncAPIError(http.MethodPost, "/api/v1/state-machines/", "bad_request")
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		result, err := s.stateMachineEngine.SendEvent(r.Context(), entityID, req.Event, req.Data)
		if err != nil {
			s.logger.Error("failed to send state machine event", zap.Error(err))
			s.metrics.IncAPIError(http.MethodPost, "/api/v1/state-machines/", "event_failed")
			http.Error(w, "failed to send event", http.StatusInternalServerError)
			return
		}

		s.metrics.IncAPIRequest(http.MethodPost, "/api/v1/state-machines/", http.StatusOK)

		response := map[string]interface{}{
			"success":    result.Success,
			"from_state": result.From,
			"to_state":   result.To,
			"duration_ms": result.Duration().Milliseconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = writeJSON(w, response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func initPostgresDataComponents(
	cfg *config.Config,
	logger *zap.Logger,
) (repository.AgentRepository, transaction.UnitOfWork, outbox.Publisher, error) {
	dbCfg := &database.Config{
		Host:              cfg.Database.Host,
		Port:              cfg.Database.Port,
		User:              cfg.Database.Username,
		Password:          cfg.Database.Password,
		Database:          cfg.Database.Name,
		SSLMode:           cfg.Database.SSLMode,
		MaxOpenConns:      cfg.Database.MaxOpenConns,
		MaxIdleConns:      cfg.Database.MaxIdleConns,
		ConnMaxLifetime:   cfg.Database.ConnMaxLifetime,
		ConnectionTimeout: 5 * time.Second,
	}
	if dbCfg.Host == "" || dbCfg.Port == 0 || dbCfg.Database == "" || dbCfg.User == "" {
		return nil, nil, nil, fmt.Errorf("database config incomplete")
	}

	db, err := database.NewDatabase(dbCfg, logger)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := db.CreateTables(context.Background()); err != nil {
		return nil, nil, nil, err
	}
	if err := db.CreateIndexes(context.Background()); err != nil {
		return nil, nil, nil, err
	}
	repo, err := repository.NewPostgresAgentRepository(db.DB)
	if err != nil {
		return nil, nil, nil, err
	}
	uow, err := transaction.NewDatabaseUnitOfWork(db)
	if err != nil {
		return nil, nil, nil, err
	}
	pub, err := outbox.NewPostgresPublisher(db.DB)
	if err != nil {
		return nil, nil, nil, err
	}
	return repo, uow, pub, nil
}