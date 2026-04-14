package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/api/grpc/pb"
	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/internal/tenant"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RuntimeManager abstracts container runtime operations behind the gRPC surface.
type RuntimeManager interface {
	StartAgent(ctx context.Context, agentID string) error
	StopAgent(ctx context.Context, agentID string, graceful bool) error
	RestartAgent(ctx context.Context, agentID string) error
	GetLogs(ctx context.Context, agentID string, tail int, since time.Duration) ([]*pb.LogEntry, error)
	StreamLogs(ctx context.Context, agentID string, follow bool) (chan *pb.LogEntry, error)
	GetMetrics(ctx context.Context, agentID string) (*pb.AgentMetrics, error)
	StreamMetrics(ctx context.Context, agentID string, interval time.Duration) (chan *pb.AgentMetrics, error)
	ExecuteCommand(ctx context.Context, agentID string, command []string, timeout time.Duration) (*CommandResult, error)
}

// AgentServiceServer implements pb.AgentServiceServer.
type AgentServiceServer struct {
	pb.UnimplementedAgentServiceServer
	logger *zap.Logger
	repo   *grpcAgentRepo
	rt     RuntimeManager
}

// NewAgentServiceServer wires persistence and runtime for the Agent gRPC API.
func NewAgentServiceServer(logger *zap.Logger, repo repository.AgentRepository, runtime RuntimeManager) *AgentServiceServer {
	if runtime == nil {
		runtime = NoopRuntimeManager{}
	}
	return &AgentServiceServer{
		logger: logger,
		repo:   newGRPCAgentRepo(repo),
		rt:     runtime,
	}
}

// CreateAgent creates a new agent record and registers it with the kernel-facing metadata path via repository.
func (s *AgentServiceServer) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.Agent, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "agent name is required")
	}
	if req.Image == "" {
		return nil, status.Error(codes.InvalidArgument, "agent image is required")
	}

	now := time.Now().UTC()
	agent := &pb.Agent{
		Id:          uuid.New().String(),
		TenantId:    tenantID,
		Name:        req.Name,
		Image:       req.Image,
		Runtime:     req.Runtime,
		Status:      pb.Status_PENDING,
		Resources:   req.Resources,
		Environment: req.Environment,
		Labels:      req.Labels,
		Annotations: req.Annotations,
		CreatedAt:   timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now),
	}
	if req.SecurityContext != nil {
		agent.SecurityContext = req.SecurityContext
	}
	if req.NetworkPolicy != nil {
		agent.NetworkPolicy = req.NetworkPolicy
	}

	if err := s.repo.Create(ctx, agent); err != nil {
		s.logger.Error("failed to create agent", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create agent")
	}
	return agent, nil
}

// GetAgent returns an agent by ID.
func (s *AgentServiceServer) GetAgent(ctx context.Context, req *pb.GetAgentRequest) (*pb.Agent, error) {
	tenantID, _ := tenant.TenantFromContext(ctx)
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if tenantID != "" && agent.TenantId != "" && agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	return agent, nil
}

// ListAgents lists agents for the tenant.
func (s *AgentServiceServer) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	filter := &agentListFilter{Status: req.Status, Labels: req.Labels}
	agents, total, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list agents")
	}
	page, pageSize := int32(1), int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 && req.Pagination.PageSize <= 100 {
			pageSize = req.Pagination.PageSize
		}
	}
	pages := int32((total + int(pageSize) - 1) / int(pageSize))
	if pages == 0 {
		pages = 1
	}
	return &pb.ListAgentsResponse{
		Agents: agents,
		Pagination: &pb.PaginationResponse{
			Page:     page,
			PageSize: pageSize,
			Total:    int32(total),
			Pages:    pages,
		},
	}, nil
}

// UpdateAgent updates agent fields.
func (s *AgentServiceServer) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.Agent, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	curPB, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if curPB.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	if req.Name != "" {
		curPB.Name = req.Name
	}
	if req.Image != "" {
		curPB.Image = req.Image
	}
	if req.Runtime != "" {
		curPB.Runtime = req.Runtime
	}
	if req.Resources != nil {
		curPB.Resources = req.Resources
	}
	if req.Environment != nil {
		curPB.Environment = req.Environment
	}
	if req.Labels != nil {
		curPB.Labels = req.Labels
	}
	if req.Annotations != nil {
		curPB.Annotations = req.Annotations
	}
	if req.SecurityContext != nil {
		curPB.SecurityContext = req.SecurityContext
	}
	if req.NetworkPolicy != nil {
		curPB.NetworkPolicy = req.NetworkPolicy
	}
	curPB.UpdatedAt = timestamppb.New(time.Now().UTC())
	if err := s.repo.Replace(ctx, curPB); err != nil {
		return nil, status.Error(codes.Internal, "failed to update agent")
	}
	return s.repo.Get(ctx, req.AgentId)
}

// DeleteAgent deletes an agent.
func (s *AgentServiceServer) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*pb.Empty, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	if agent.Status == pb.Status_RUNNING {
		if err := s.rt.StopAgent(ctx, req.AgentId, !req.Force); err != nil && !req.Force {
			return nil, status.Error(codes.FailedPrecondition, "failed to stop agent")
		}
	}
	if err := s.repo.Delete(ctx, req.AgentId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete agent")
	}
	return &pb.Empty{}, nil
}

// StartAgent starts an agent via the runtime manager.
func (s *AgentServiceServer) StartAgent(ctx context.Context, req *pb.StartAgentRequest) (*pb.AgentActionResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	old := agent.Status
	if err := s.rt.StartAgent(ctx, req.AgentId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to start agent: %v", err))
	}
	_ = s.repo.UpdateStatus(ctx, req.AgentId, repository.AgentStatusRunning, "")
	return &pb.AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "start",
		OldStatus: old,
		NewStatus: pb.Status_RUNNING,
		Message:   "Agent started successfully",
	}, nil
}

// StopAgent stops a running agent.
func (s *AgentServiceServer) StopAgent(ctx context.Context, req *pb.StopAgentRequest) (*pb.AgentActionResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	old := agent.Status
	if err := s.rt.StopAgent(ctx, req.AgentId, req.Graceful); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to stop agent: %v", err))
	}
	_ = s.repo.UpdateStatus(ctx, req.AgentId, repository.AgentStatusStopped, "")
	return &pb.AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "stop",
		OldStatus: old,
		NewStatus: pb.Status_STOPPED,
		Message:   "Agent stopped successfully",
	}, nil
}

// RestartAgent restarts an agent.
func (s *AgentServiceServer) RestartAgent(ctx context.Context, req *pb.RestartAgentRequest) (*pb.AgentActionResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	old := agent.Status
	if err := s.rt.RestartAgent(ctx, req.AgentId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to restart agent: %v", err))
	}
	_ = s.repo.UpdateStatus(ctx, req.AgentId, repository.AgentStatusRunning, "")
	return &pb.AgentActionResponse{
		AgentId:   req.AgentId,
		Action:    "restart",
		OldStatus: old,
		NewStatus: pb.Status_RUNNING,
		Message:   "Agent restarted successfully",
	}, nil
}

// GetAgentLogs returns recent log lines.
func (s *AgentServiceServer) GetAgentLogs(ctx context.Context, req *pb.GetAgentLogsRequest) (*pb.GetAgentLogsResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	since := time.Hour
	if req.Since != "" {
		if d, err := time.ParseDuration(req.Since); err == nil {
			since = d
		}
	}
	logs, err := s.rt.GetLogs(ctx, req.AgentId, int(req.Tail), since)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get logs")
	}
	return &pb.GetAgentLogsResponse{AgentId: req.AgentId, Entries: logs}, nil
}

// StreamAgentLogs streams log entries.
func (s *AgentServiceServer) StreamAgentLogs(req *pb.StreamAgentLogsRequest, stream pb.AgentService_StreamAgentLogsServer) error {
	ctx := stream.Context()
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	ch, err := s.rt.StreamLogs(ctx, req.AgentId, req.Follow)
	if err != nil {
		return status.Error(codes.Internal, "failed to stream logs")
	}
	for entry := range ch {
		if entry == nil {
			continue
		}
		if err := stream.Send(entry); err != nil {
			return err
		}
	}
	return nil
}

// GetAgentMetrics returns metrics for an agent.
func (s *AgentServiceServer) GetAgentMetrics(ctx context.Context, req *pb.GetAgentMetricsRequest) (*pb.AgentMetrics, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	return s.rt.GetMetrics(ctx, req.AgentId)
}

// StreamAgentMetrics streams metrics samples.
func (s *AgentServiceServer) StreamAgentMetrics(req *pb.StreamAgentMetricsRequest, stream pb.AgentService_StreamAgentMetricsServer) error {
	ctx := stream.Context()
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	interval := time.Duration(req.IntervalSeconds) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	ch, err := s.rt.StreamMetrics(ctx, req.AgentId, interval)
	if err != nil {
		return status.Error(codes.Internal, "failed to stream metrics")
	}
	for m := range ch {
		if m == nil {
			continue
		}
		if err := stream.Send(m); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteCommand runs a command inside the agent environment.
func (s *AgentServiceServer) ExecuteCommand(ctx context.Context, req *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	tenantID, ok := tenant.TenantFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "tenant ID required")
	}
	agent, err := s.repo.Get(ctx, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agent.TenantId != tenantID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	if agent.Status != pb.Status_RUNNING {
		return nil, status.Error(codes.FailedPrecondition, "agent is not running")
	}
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}
	result, err := s.rt.ExecuteCommand(ctx, req.AgentId, req.Command, timeout)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to execute command: %v", err))
	}
	return &pb.ExecuteCommandResponse{
		AgentId:  req.AgentId,
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

// GetAgentEvents returns recent events (stub).
func (s *AgentServiceServer) GetAgentEvents(ctx context.Context, req *pb.GetAgentEventsRequest) (*pb.GetAgentEventsResponse, error) {
	return &pb.GetAgentEventsResponse{AgentId: req.AgentId, Events: nil}, nil
}

// StreamAgentEvents streams events (stub).
func (s *AgentServiceServer) StreamAgentEvents(req *pb.StreamAgentEventsRequest, stream pb.AgentService_StreamAgentEventsServer) error {
	return nil
}
