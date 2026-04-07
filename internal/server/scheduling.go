package server

import (
	"context"
	"fmt"

	"github.com/agentos/aos/internal/scheduler"
)

// scheduleAgentBaseline delegates scheduling to the Server's scheduler.
func (s *Server) scheduleAgentBaseline(ctx context.Context, agentID string) (string, error) {
	if agentID == "" {
		return "", fmt.Errorf("agent id is required")
	}
	if s.scheduler == nil {
		return "node-local-1", nil
	}
	result, err := s.scheduler.Schedule(ctx, scheduler.TaskRequest{
		TaskID:   agentID,
		TaskType: "agent",
	})
	if err != nil {
		return "", fmt.Errorf("scheduler error: %w", err)
	}
	if !result.Success {
		return "", fmt.Errorf("scheduling failed: %s", result.Message)
	}
	return result.NodeID, nil
}

