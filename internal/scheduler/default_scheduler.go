package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/internal/config"
)

// DefaultScheduler is a round-robin, in-memory scheduler implementation.
type DefaultScheduler struct {
	mu    sync.RWMutex
	nodes map[string]NodeState
	cfg   *config.Config

	totalScheduled int64
	totalFailed    int64
	rrIndex        int
}

// NewDefaultScheduler returns a ready-to-use DefaultScheduler.
func NewDefaultScheduler() *DefaultScheduler {
	return &DefaultScheduler{
		nodes: make(map[string]NodeState),
	}
}

func (s *DefaultScheduler) Initialize(_ context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

func (s *DefaultScheduler) Schedule(_ context.Context, req TaskRequest) (*ScheduleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := s.scheduleLocked(req)
	return &res, nil
}

// scheduleLocked performs one scheduling decision; caller must hold s.mu.
func (s *DefaultScheduler) scheduleLocked(req TaskRequest) ScheduleResult {
	eligible := s.eligibleNodes(req)
	if len(eligible) == 0 {
		s.totalFailed++
		return ScheduleResult{
			Success: false,
			TaskID:  req.TaskID,
			Reason:  "no_eligible_node",
			Message: "no node has enough resources or is healthy",
		}
	}

	selected := eligible[s.rrIndex%len(eligible)]
	s.rrIndex++

	if n, ok := s.nodes[selected.NodeID]; ok {
		n.AgentCount++
		n.LastUpdate = time.Now()
		s.nodes[selected.NodeID] = n
	}

	s.totalScheduled++
	return ScheduleResult{
		Success:     true,
		NodeID:      selected.NodeID,
		TaskID:      req.TaskID,
		Message:     fmt.Sprintf("scheduled on %s", selected.NodeID),
		Score:       1.0,
		ScheduledAt: time.Now(),
	}
}

func (s *DefaultScheduler) ScheduleBatch(ctx context.Context, reqs []TaskRequest) ([]ScheduleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]ScheduleResult, 0, len(reqs))
	for _, req := range reqs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		results = append(results, s.scheduleLocked(req))
	}
	return results, nil
}

func (s *DefaultScheduler) Reschedule(ctx context.Context, taskID string, _ RescheduleReason) (*ScheduleResult, error) {
	return s.Schedule(ctx, TaskRequest{TaskID: taskID})
}

func (s *DefaultScheduler) CancelTask(_ context.Context, _ string) error {
	return nil
}

func (s *DefaultScheduler) AddNode(_ context.Context, node NodeState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if node.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	node.LastUpdate = time.Now()
	s.nodes[node.NodeID] = node
	return nil
}

func (s *DefaultScheduler) RemoveNode(_ context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[nodeID]; !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	delete(s.nodes, nodeID)
	return nil
}

func (s *DefaultScheduler) UpdateNode(_ context.Context, node NodeState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[node.NodeID]; !ok {
		return fmt.Errorf("node %s not found", node.NodeID)
	}
	node.LastUpdate = time.Now()
	s.nodes[node.NodeID] = node
	return nil
}

func (s *DefaultScheduler) ListNodes(_ context.Context) ([]NodeState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]NodeState, 0, len(s.nodes))
	for _, n := range s.nodes {
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (s *DefaultScheduler) HealthCheck(_ context.Context) error {
	return nil
}

func (s *DefaultScheduler) GetStats(_ context.Context) (*SchedulerMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &SchedulerMetrics{
		TotalScheduled: s.totalScheduled,
		TotalFailed:    s.totalFailed,
		NodeCount:      len(s.nodes),
	}, nil
}

// eligibleNodes filters nodes that are healthy and have capacity.
// Must be called while holding s.mu.
func (s *DefaultScheduler) eligibleNodes(req TaskRequest) []NodeState {
	var result []NodeState
	for _, n := range s.nodes {
		if n.Health != "healthy" && n.Health != "" {
			continue
		}
		remainCPU := float64(n.CPUCores) * (1 - n.CPUUsage/100)
		remainMem := float64(n.MemoryBytes) * (1 - n.MemoryUsage/100)
		if req.CPURequest > 0 && remainCPU < req.CPURequest {
			continue
		}
		if req.MemoryRequest > 0 && remainMem < float64(req.MemoryRequest) {
			continue
		}
		result = append(result, n)
	}
	return result
}

var _ Scheduler = (*DefaultScheduler)(nil)
