package grpc

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/tenant"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServiceServer implements the gRPC AgentService
type AgentServiceServer struct {
	logger *zap.Logger
	repo   AgentRepository
	// Runtime manager for agent lifecycle
	runtimeManager RuntimeManager
}

// AgentRepository defines the interface for agent storage
type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	Get(ctx context.Context, id string) (*Agent, error)
	List(ctx context.Context, tenantID string, filter *AgentListFilter) ([]*Agent, int, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status string, message string) error
}

// RuntimeManager defines the interface for agent runtime operations
type RuntimeManager interface {
	StartAgent(ctx context.Context, agentID string) error
	StopAgent(ctx context.Context, agentID string, graceful bool) error
	RestartAgent(ctx context.Context, agentID string) error
	GetLogs(ctx context.Context, agentID string, tail int, since time.Duration) ([]LogEntry, error)
	StreamLogs(ctx context.Context, agentID string, follow bool) (chan LogEntry, error)
	GetMetrics(ctx context.Context, agentID string) (*AgentMetrics, error)
	StreamMetrics(ctx context.Context, agentID string, interval time.Duration) (chan *AgentMetrics, error)
	ExecuteCommand(ctx context.Context, agentID string, command []string, timeout time.Duration) (*CommandResult, error)
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// NewAgentServiceServer creates a new agent service server
func NewAgentServiceServer(logger *zap.Logger, repo AgentRepository, runtime RuntimeManager) *AgentServiceServer {
	return &AgentServiceServer{
		logger:         logger,
		repo:           repo,
		runtimeManager: runtime,
	}
}

// CreateAgent creates a new agent
func (s *AgentServiceServer) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*Agent, error) {
	logger := s.logger.With(zap.String("method", "CreateAgent"))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	logger = logger.With(zap.String("tenant_id", tenantID))

	// Validate request
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "agent name is required")
	}
	if req.Image == "" {
		return nil, status.Error(codes.InvalidArgument, "agent image is required")
	}

	// Create agent
	agent := &Agent{
		Id:        uuid.New().String(),
		TenantId:  tenantID,
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   req.Runtime,
		Status:    Status_PENDING,
		CreatedAt: timeToProto(time.Now()),
		UpdatedAt: timeToProto(time.Now()),
	}

	// Copy resources
	if req.Resources != nil {
		agent.Resources = &Resource{
			Cpu:     req.Resources.Cpu,
			Memory:  req.Resources.Memory,
			Storage: req.Resources.Storage,
			Gpu:     req.Resources.Gpu,
		}
	}

	// Copy environment
	if req.Environment != nil {
		agent.Environment = req.Environment
	}

	// Copy labels
	if req.Labels != nil {
		agent.Labels = req.Labels
	}

	// Copy security context
	if req.SecurityContext != nil {
		agent.SecurityContext = &SecurityContext{
			RunAsUser:               req.SecurityContext.RunAsUser,
			RunAsGroup:              req.SecurityContext.RunAsGroup,
			ReadOnlyRootFs:          req.SecurityContext.ReadOnlyRootFs,
			AllowPrivilegeEscalation: req.SecurityContext.AllowPrivilegeEscalation,
			SandboxType:             req.SecurityContext.SandboxType,
			SeccompProfile:          req.SecurityContext.SeccompProfile,
		}
		if req.SecurityContext.CapabilitiesAdd != nil {
			agent.SecurityContext.CapabilitiesAdd = req.SecurityContext.CapabilitiesAdd
		}
		if req.SecurityContext.CapabilitiesDrop != nil {
			agent.SecurityContext.CapabilitiesDrop = req.SecurityContext.CapabilitiesDrop
		}
	}

	// Save to repository
	if err := s.repo.Create(ctx, agent); err != nil {
		logger.Error("failed to create agent", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create agent")
	}

	logger.Info("agent created", zap.String("agent_id", agent.Id))
	return agent, nil
}

// GetAgent retrieves an agent by ID
func (s *AgentServiceServer) GetAgent(ctx context.Context, req *GetAgentRequest) (*Agent, error) {
	logger := s.logger.With(zap.String("method", "GetAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context for validation
	tenantID, _ := tenant.TenantFromContext(ctx)

	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if tenantID != "" && agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return agent, nil
}

// ListAgents lists agents with filtering
func (s *AgentServiceServer) ListAgents(ctx context.Context, req *ListAgentsRequest) (*ListAgentsResponse, error) {
	logger := s.logger.With(zap.String("method", "ListAgents"))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	logger = logger.With(zap.String("tenant_id", tenantID))

	// Build filter
	filter := &AgentListFilter{
		Status: req.Status,
	}
	if req.Labels != nil {
		filter.Labels = req.Labels
	}

	// Set pagination defaults
	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 && req.Pagination.PageSize <= 100 {
			pageSize = req.Pagination.PageSize
		}
	}

	// Query repository
	agents, total, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		logger.Error("failed to list agents", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list agents")
	}

	// Calculate pagination
	pages := (total + int(pageSize) - 1) / int(pageSize)
	if pages == 0 {
		pages = 1
	}

	return &ListAgentsResponse{
		Agents: agents,
		Total:  int32(total),
		Page:   page,
		Pages:  int32(pages),
	}, nil
}

// UpdateAgent updates an existing agent
func (s *AgentServiceServer) UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*Agent, error) {
	logger := s.logger.With(zap.String("method", "UpdateAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Update fields
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
	if req.Environment != nil {
		agent.Environment = req.Environment
	}
	if req.Labels != nil {
		agent.Labels = req.Labels
	}

	agent.UpdatedAt = timeToProto(time.Now())

	// Save changes
	if err := s.repo.Update(ctx, agent); err != nil {
		logger.Error("failed to update agent", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update agent")
	}

	logger.Info("agent updated")
	return agent, nil
}

// DeleteAgent deletes an agent
func (s *AgentServiceServer) DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*Empty, error) {
	logger := s.logger.With(zap.String("method", "DeleteAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Stop agent if running
	if agent.Status == Status_RUNNING {
		if err := s.runtimeManager.StopAgent(ctx, req.AgentId, !req.Force); err != nil {
			if !req.Force {
				logger.Error("failed to stop agent before deletion", zap.Error(err))
				return nil, status.Error(codes.FailedPrecondition, "failed to stop agent")
			}
		}
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, req.AgentId); err != nil {
		logger.Error("failed to delete agent", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete agent")
	}

	logger.Info("agent deleted")
	return &Empty{}, nil
}

// StartAgent starts an agent
func (s *AgentServiceServer) StartAgent(ctx context.Context, req *StartAgentRequest) (*AgentActionResponse, error) {
	logger := s.logger.With(zap.String("method", "StartAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	oldStatus := agent.Status

	// Start the agent via runtime manager
	if err := s.runtimeManager.StartAgent(ctx, req.AgentId); err != nil {
		logger.Error("failed to start agent", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to start agent: %v", err))
	}

	// Update agent status
	if err := s.repo.UpdateStatus(ctx, req.AgentId, "running", ""); err != nil {
		logger.Warn("failed to update agent status", zap.Error(err))
	}

	return &AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "start",
		OldStatus: oldStatus,
		NewStatus: Status_RUNNING,
		Message:   "Agent started successfully",
	}, nil
}

// StopAgent stops an agent
func (s *AgentServiceServer) StopAgent(ctx context.Context, req *StopAgentRequest) (*AgentActionResponse, error) {
	logger := s.logger.With(zap.String("method", "StopAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	oldStatus := agent.Status

	// Stop the agent via runtime manager
	if err := s.runtimeManager.StopAgent(ctx, req.AgentId, req.Graceful); err != nil {
		logger.Error("failed to stop agent", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to stop agent: %v", err))
	}

	// Update agent status
	if err := s.repo.UpdateStatus(ctx, req.AgentId, "stopped", ""); err != nil {
		logger.Warn("failed to update agent status", zap.Error(err))
	}

	return &AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "stop",
		OldStatus: oldStatus,
		NewStatus: Status_STOPPED,
		Message:   "Agent stopped successfully",
	}, nil
}

// RestartAgent restarts an agent
func (s *AgentServiceServer) RestartAgent(ctx context.Context, req *RestartAgentRequest) (*AgentActionResponse, error) {
	logger := s.logger.With(zap.String("method", "RestartAgent"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	oldStatus := agent.Status

	// Restart the agent via runtime manager
	if err := s.runtimeManager.RestartAgent(ctx, req.AgentId); err != nil {
		logger.Error("failed to restart agent", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to restart agent: %v", err))
	}

	// Update agent status
	if err := s.repo.UpdateStatus(ctx, req.AgentId, "running", ""); err != nil {
		logger.Warn("failed to update agent status", zap.Error(err))
	}

	return &AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "restart",
		OldStatus: oldStatus,
		NewStatus: Status_RUNNING,
		Message:   "Agent restarted successfully",
	}, nil
}

// GetAgentLogs retrieves agent logs
func (s *AgentServiceServer) GetAgentLogs(ctx context.Context, req *GetAgentLogsRequest) (*GetAgentLogsResponse, error) {
	logger := s.logger.With(zap.String("method", "GetAgentLogs"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Parse since duration
	since := time.Hour // default 1 hour
	if req.Since != "" {
		d, err := time.ParseDuration(req.Since)
		if err == nil {
			since = d
		}
	}

	// Get logs from runtime manager
	logs, err := s.runtimeManager.GetLogs(ctx, req.AgentId, int(req.Tail), since)
	if err != nil {
		logger.Error("failed to get agent logs", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get logs")
	}

	return &GetAgentLogsResponse{
		AgentId: req.AgentId,
		Entries: logs,
	}, nil
}

// StreamAgentLogs streams agent logs (server-side streaming)
func (s *AgentServiceServer) StreamAgentLogs(req *StreamAgentLogsRequest, stream AgentService_StreamAgentLogsServer) error {
	logger := s.logger.With(zap.String("method", "StreamAgentLogs"), zap.String("agent_id", req.AgentId))

	ctx := stream.Context()

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	// Stream logs from runtime manager
	logChan, err := s.runtimeManager.StreamLogs(ctx, req.AgentId, req.Follow)
	if err != nil {
		logger.Error("failed to stream agent logs", zap.Error(err))
		return status.Error(codes.Internal, "failed to stream logs")
	}

	for entry := range logChan {
		if err := stream.Send(&entry); err != nil {
			logger.Error("failed to send log entry", zap.Error(err))
			return err
		}
	}

	return nil
}

// GetAgentMetrics retrieves agent metrics
func (s *AgentServiceServer) GetAgentMetrics(ctx context.Context, req *GetAgentMetricsRequest) (*AgentMetrics, error) {
	logger := s.logger.With(zap.String("method", "GetAgentMetrics"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Get metrics from runtime manager
	metrics, err := s.runtimeManager.GetMetrics(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent metrics", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get metrics")
	}

	return metrics, nil
}

// StreamAgentMetrics streams agent metrics (server-side streaming)
func (s *AgentServiceServer) StreamAgentMetrics(req *StreamAgentMetricsRequest, stream AgentService_StreamAgentMetricsServer) error {
	logger := s.logger.With(zap.String("method", "StreamAgentMetrics"), zap.String("agent_id", req.AgentId))

	ctx := stream.Context()

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	// Stream metrics from runtime manager
	interval := time.Duration(req.IntervalSeconds) * time.Second
	if interval < time.Second {
		interval = time.Second * 5
	}

	metricsChan, err := s.runtimeManager.StreamMetrics(ctx, req.AgentId, interval)
	if err != nil {
		logger.Error("failed to stream agent metrics", zap.Error(err))
		return status.Error(codes.Internal, "failed to stream metrics")
	}

	for metrics := range metricsChan {
		if err := stream.Send(metrics); err != nil {
			logger.Error("failed to send metrics", zap.Error(err))
			return err
		}
	}

	return nil
}

// ExecuteCommand executes a command in an agent
func (s *AgentServiceServer) ExecuteCommand(ctx context.Context, req *ExecuteCommandRequest) (*ExecuteCommandResponse, error) {
	logger := s.logger.With(zap.String("method", "ExecuteCommand"), zap.String("agent_id", req.AgentId))

	// Get tenant from context
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}

	// Get existing agent
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		logger.Error("failed to get agent", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	// Verify tenant access
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Verify agent is running
	if agent.Status != Status_RUNNING {
		return nil, status.Error(codes.FailedPrecondition, "agent is not running")
	}

	// Execute command via runtime manager
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}

	result, err := s.runtimeManager.ExecuteCommand(ctx, req.AgentId, req.Command, timeout)
	if err != nil {
		logger.Error("failed to execute command", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to execute command: %v", err))
	}

	return &ExecuteCommandResponse{
		AgentId:   req.AgentId,
		ExitCode:  int32(result.ExitCode),
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
	}, nil
}

// GetAgentEvents retrieves agent events
func (s *AgentServiceServer) GetAgentEvents(ctx context.Context, req *GetAgentEventsRequest) (*GetAgentEventsResponse, error) {
	// This would typically query an event store
	// For now, return empty list
	return &GetAgentEventsResponse{
		AgentId: req.AgentId,
		Events:  []*AgentEvent{},
	}, nil
}

// StreamAgentEvents streams agent events (server-side streaming)
func (s *AgentServiceServer) StreamAgentEvents(req *StreamAgentEventsRequest, stream AgentService_StreamAgentEventsServer) error {
	// This would typically connect to an event bus
	// For now, just return
	return nil
}

// Helper types and interfaces

type AgentListFilter struct {
	Status string
	Labels map[string]string
}

// timeToProto converts time.Time to proto timestamp
func timeToProto(t time.Time) *Timestamp {
	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// From agent.proto types

// LogEntry type alias
type LogEntry = LogEntry

// Timestamp type
type Timestamp struct {
	Seconds int64
	Nanos   int32
}
