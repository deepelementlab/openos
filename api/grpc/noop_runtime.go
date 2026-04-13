package grpc

import (
	"context"
	"time"

	"github.com/agentos/aos/api/grpc/pb"
)

// NoopRuntimeManager satisfies RuntimeManager with safe stubs (dev / tests).
type NoopRuntimeManager struct{}

func (NoopRuntimeManager) StartAgent(ctx context.Context, agentID string) error { return nil }

func (NoopRuntimeManager) StopAgent(ctx context.Context, agentID string, graceful bool) error {
	return nil
}

func (NoopRuntimeManager) RestartAgent(ctx context.Context, agentID string) error { return nil }

func (NoopRuntimeManager) GetLogs(ctx context.Context, agentID string, tail int, since time.Duration) ([]*pb.LogEntry, error) {
	return nil, nil
}

func (NoopRuntimeManager) StreamLogs(ctx context.Context, agentID string, follow bool) (chan *pb.LogEntry, error) {
	ch := make(chan *pb.LogEntry)
	close(ch)
	return ch, nil
}

func (NoopRuntimeManager) GetMetrics(ctx context.Context, agentID string) (*pb.AgentMetrics, error) {
	return &pb.AgentMetrics{AgentId: agentID}, nil
}

func (NoopRuntimeManager) StreamMetrics(ctx context.Context, agentID string, interval time.Duration) (chan *pb.AgentMetrics, error) {
	ch := make(chan *pb.AgentMetrics)
	close(ch)
	return ch, nil
}

func (NoopRuntimeManager) ExecuteCommand(ctx context.Context, agentID string, command []string, timeout time.Duration) (*CommandResult, error) {
	return &CommandResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
}
