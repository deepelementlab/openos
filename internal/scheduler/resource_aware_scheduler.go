package scheduler

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/agentos/aos/internal/config"
)

// ScheduleStrategy defines the scheduling algorithm to use.
type ScheduleStrategy string

const (
	StrategyRoundRobin  ScheduleStrategy = "round-robin"
	StrategyBestFit     ScheduleStrategy = "best-fit"
	StrategyWorstFit    ScheduleStrategy = "worst-fit"
	StrategyLeastLoaded ScheduleStrategy = "least-loaded"
)

// ResourceAwareScheduler implements resource-aware scheduling algorithms.
// It supports multiple strategies and tracks actual resource allocation.
type ResourceAwareScheduler struct {
	mu                  sync.RWMutex
	nodes               map[string]*nodeAlloc
	allocations         map[string]string // taskID -> nodeID
	resourceAllocations map[string]TaskResourceAllocation // taskID -> resource details
	strategy            ScheduleStrategy
	cfg                 *config.Config
	totalScheduled      int64
	totalFailed         int64
	rrIndex             int
}

// TaskResourceAllocation stores detailed resource allocation for a task
type TaskResourceAllocation struct {
	NodeID    string
	CPU       float64
	Memory    int64
	GPU       int
	Disk      int64
	Timestamp time.Time
}

// nodeAlloc tracks per-node resource allocation state.
type nodeAlloc struct {
	state          NodeState
	allocatedCPU   float64 // millicores
	allocatedMem   int64   // bytes
	allocatedGPU   int
	allocatedDisk  int64
	agentCount     int
	lastUpdate     time.Time
}

// NewResourceAwareScheduler creates a scheduler with the given strategy.
func NewResourceAwareScheduler(strategy ScheduleStrategy) *ResourceAwareScheduler {
	return &ResourceAwareScheduler{
		nodes:               make(map[string]*nodeAlloc),
		allocations:         make(map[string]string),
		resourceAllocations: make(map[string]TaskResourceAllocation),
		strategy:            strategy,
	}
}

// Initialize configures the scheduler.
func (s *ResourceAwareScheduler) Initialize(_ context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

// Schedule assigns a task to the best available node.
func (s *ResourceAwareScheduler) Schedule(ctx context.Context, req TaskRequest) (*ScheduleResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eligible := s.filterNodes(req)
	if len(eligible) == 0 {
		s.totalFailed++
		return &ScheduleResult{
			Success:     false,
			TaskID:      req.TaskID,
			Reason:      "no_eligible_node",
			Message:     "no node has enough available resources",
			ScheduledAt: time.Now(),
		}, nil
	}

	selected, score := s.selectNode(eligible, req)
	if selected == nil {
		s.totalFailed++
		return &ScheduleResult{
			Success:     false,
			TaskID:      req.TaskID,
			Reason:      "selection_failed",
			Message:     "failed to select a node",
			ScheduledAt: time.Now(),
		}, nil
	}

	// Apply allocation
	selected.allocatedCPU += req.CPURequest
	selected.allocatedMem += req.MemoryRequest
	selected.allocatedGPU += req.GPURequest
	selected.allocatedDisk += req.DiskRequest
	selected.agentCount++
	selected.lastUpdate = time.Now()
	s.allocations[req.TaskID] = selected.state.NodeID
	// Store resource allocation for cancellation
	if s.resourceAllocations == nil {
		s.resourceAllocations = make(map[string]TaskResourceAllocation)
	}
	s.resourceAllocations[req.TaskID] = TaskResourceAllocation{
		NodeID:    selected.state.NodeID,
		CPU:       req.CPURequest,
		Memory:    req.MemoryRequest,
		GPU:       req.GPURequest,
		Disk:      req.DiskRequest,
		Timestamp: time.Now(),
	}

	s.totalScheduled++

	return &ScheduleResult{
		Success:     true,
		NodeID:      selected.state.NodeID,
		TaskID:      req.TaskID,
		Message:     fmt.Sprintf("scheduled on %s (score: %.2f)", selected.state.NodeID, score),
		Score:       score,
		ScheduledAt: time.Now(),
	}, nil
}

// ScheduleBatch schedules multiple tasks.
func (s *ResourceAwareScheduler) ScheduleBatch(ctx context.Context, reqs []TaskRequest) ([]ScheduleResult, error) {
	results := make([]ScheduleResult, 0, len(reqs))
	for _, req := range reqs {
		res, err := s.Schedule(ctx, req)
		if err != nil {
			return nil, err
		}
		results = append(results, *res)
	}
	return results, nil
}

// Reschedule reassigns a task to a new node.
func (s *ResourceAwareScheduler) Reschedule(ctx context.Context, taskID string, reason RescheduleReason) (*ScheduleResult, error) {
	s.mu.Lock()
	if oldNodeID, ok := s.allocations[taskID]; ok {
		// Release resources before rescheduling
		if resourceAlloc, hasResources := s.resourceAllocations[taskID]; hasResources {
			if na, ok := s.nodes[oldNodeID]; ok {
				na.allocatedCPU -= resourceAlloc.CPU
				if na.allocatedCPU < 0 { na.allocatedCPU = 0 }
				na.allocatedMem -= resourceAlloc.Memory
				if na.allocatedMem < 0 { na.allocatedMem = 0 }
				na.allocatedGPU -= resourceAlloc.GPU
				if na.allocatedGPU < 0 { na.allocatedGPU = 0 }
				na.allocatedDisk -= resourceAlloc.Disk
				if na.allocatedDisk < 0 { na.allocatedDisk = 0 }
				na.agentCount--
				if na.agentCount < 0 { na.agentCount = 0 }
				na.lastUpdate = time.Now()
			}
		}
		// Clean up allocations
		delete(s.allocations, taskID)
		delete(s.resourceAllocations, taskID)
	}
	s.mu.Unlock()

	return s.Schedule(ctx, TaskRequest{TaskID: taskID})
}

// CancelTask removes a task allocation.
func (s *ResourceAwareScheduler) CancelTask(_ context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodeID, ok := s.allocations[taskID]
	if !ok {
		return nil // already cancelled
	}
	
	// Get resource allocation details
	resourceAlloc, hasResources := s.resourceAllocations[taskID]
	if na, ok := s.nodes[nodeID]; ok && hasResources {
		// Release allocated resources
		na.allocatedCPU -= resourceAlloc.CPU
		if na.allocatedCPU < 0 {
			na.allocatedCPU = 0
		}
		na.allocatedMem -= resourceAlloc.Memory
		if na.allocatedMem < 0 {
			na.allocatedMem = 0
		}
		na.allocatedGPU -= resourceAlloc.GPU
		if na.allocatedGPU < 0 {
			na.allocatedGPU = 0
		}
		na.allocatedDisk -= resourceAlloc.Disk
		if na.allocatedDisk < 0 {
			na.allocatedDisk = 0
		}
		
		na.agentCount--
		if na.agentCount < 0 {
			na.agentCount = 0
		}
		na.lastUpdate = time.Now()
	}
	
	// Clean up allocations
	delete(s.allocations, taskID)
	delete(s.resourceAllocations, taskID)
	return nil
}

// AddNode registers a new scheduling node.
func (s *ResourceAwareScheduler) AddNode(_ context.Context, node NodeState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if node.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	s.nodes[node.NodeID] = &nodeAlloc{
		state:      node,
		lastUpdate: time.Now(),
	}
	return nil
}

// RemoveNode removes a node and cancels its allocations.
func (s *ResourceAwareScheduler) RemoveNode(_ context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[nodeID]; !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	// Remove allocations pointing to this node
	for taskID, nid := range s.allocations {
		if nid == nodeID {
			delete(s.allocations, taskID)
		}
	}
	delete(s.nodes, nodeID)
	return nil
}

// UpdateNode updates the base state of a node.
func (s *ResourceAwareScheduler) UpdateNode(_ context.Context, node NodeState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	na, ok := s.nodes[node.NodeID]
	if !ok {
		return fmt.Errorf("node %s not found", node.NodeID)
	}
	na.state = node
	na.lastUpdate = time.Now()
	return nil
}

// ListNodes returns all nodes with their current allocation state.
func (s *ResourceAwareScheduler) ListNodes(_ context.Context) ([]NodeState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]NodeState, 0, len(s.nodes))
	for _, na := range s.nodes {
		nodes = append(nodes, na.toNodeState())
	}
	return nodes, nil
}

// HealthCheck verifies scheduler health.
func (s *ResourceAwareScheduler) HealthCheck(_ context.Context) error {
	return nil
}

// GetStats returns scheduler metrics.
func (s *ResourceAwareScheduler) GetStats(_ context.Context) (*SchedulerMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &SchedulerMetrics{
		TotalScheduled: s.totalScheduled,
		TotalFailed:    s.totalFailed,
		NodeCount:      len(s.nodes),
		QueueSize:      len(s.allocations),
		AlgorithmStats: map[string]int64{
			"strategy":          int64(len(s.strategy)),
			"active_allocations": int64(len(s.allocations)),
		},
	}, nil
}

// SetStrategy changes the scheduling strategy.
func (s *ResourceAwareScheduler) SetStrategy(strategy ScheduleStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy
}

// --- internal methods ---

// filterNodes returns nodes that have enough free resources for the request.
func (s *ResourceAwareScheduler) filterNodes(req TaskRequest) []*nodeAlloc {
	var eligible []*nodeAlloc
	for _, na := range s.nodes {
		// Health check
		if na.state.Health != "healthy" && na.state.Health != "" {
			continue
		}
		// CPU check (CPUCores * 1000 = total millicores, convert usage to allocated)
		totalCPU := float64(na.state.CPUCores) * 1000
		freeCPU := totalCPU - na.allocatedCPU - (na.state.CPUUsage / 100 * totalCPU)
		if req.CPURequest > 0 && freeCPU < req.CPURequest {
			continue
		}
		// Memory check
		freeMem := float64(na.state.MemoryBytes) - float64(na.allocatedMem) -
			(float64(na.state.MemoryBytes) * na.state.MemoryUsage / 100)
		if req.MemoryRequest > 0 && freeMem < float64(req.MemoryRequest) {
			continue
		}
		// GPU check
		freeGPU := na.state.GPUCount - na.allocatedGPU
		if req.GPURequest > 0 && freeGPU < req.GPURequest {
			continue
		}
		// Disk check
		freeDisk := float64(na.state.DiskBytes) - float64(na.allocatedDisk) -
			(float64(na.state.DiskBytes) * na.state.DiskUsage / 100)
		if req.DiskRequest > 0 && freeDisk < float64(req.DiskRequest) {
			continue
		}
		eligible = append(eligible, na)
	}
	return eligible
}

// selectNode picks the best node based on the current strategy.
func (s *ResourceAwareScheduler) selectNode(eligible []*nodeAlloc, req TaskRequest) (*nodeAlloc, float64) {
	switch s.strategy {
	case StrategyBestFit:
		return s.selectBestFit(eligible, req)
	case StrategyWorstFit:
		return s.selectWorstFit(eligible, req)
	case StrategyLeastLoaded:
		return s.selectLeastLoaded(eligible)
	case StrategyRoundRobin:
		fallthrough
	default:
		return s.selectRoundRobin(eligible)
	}
}

// selectBestFit picks the node with the least remaining resources after allocation.
func (s *ResourceAwareScheduler) selectBestFit(eligible []*nodeAlloc, req TaskRequest) (*nodeAlloc, float64) {
	type scored struct {
		na    *nodeAlloc
		score float64
	}
	var candidates []scored
	for _, na := range eligible {
		totalCPU := float64(na.state.CPUCores) * 1000
		usedCPU := na.allocatedCPU + (na.state.CPUUsage / 100 * totalCPU)
		totalMem := float64(na.state.MemoryBytes)
		usedMem := float64(na.allocatedMem) + (totalMem * na.state.MemoryUsage / 100)

		// Score: lower remaining = better fit
		cpuAfterAlloc := (usedCPU + req.CPURequest) / totalCPU
		memAfterAlloc := (usedMem + float64(req.MemoryRequest)) / totalMem

		// Average utilization score (higher = better fit)
		score := (cpuAfterAlloc + memAfterAlloc) / 2.0
		candidates = append(candidates, scored{na: na, score: score})
	}
	if len(candidates) == 0 {
		return nil, 0
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score // highest utilization = best fit
	})
	return candidates[0].na, candidates[0].score
}

// selectWorstFit picks the node with the most remaining resources after allocation.
func (s *ResourceAwareScheduler) selectWorstFit(eligible []*nodeAlloc, req TaskRequest) (*nodeAlloc, float64) {
	type scored struct {
		na    *nodeAlloc
		score float64
	}
	var candidates []scored
	for _, na := range eligible {
		totalCPU := float64(na.state.CPUCores) * 1000
		usedCPU := na.allocatedCPU + (na.state.CPUUsage / 100 * totalCPU)
		totalMem := float64(na.state.MemoryBytes)
		usedMem := float64(na.allocatedMem) + (totalMem * na.state.MemoryUsage / 100)

		// Score: higher remaining = better (spread load)
		cpuFree := 1.0 - (usedCPU+req.CPURequest)/totalCPU
		memFree := 1.0 - (usedMem+float64(req.MemoryRequest))/totalMem

		score := (cpuFree + memFree) / 2.0
		candidates = append(candidates, scored{na: na, score: score})
	}
	if len(candidates) == 0 {
		return nil, 0
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score // most free = worst fit for bin-packing
	})
	return candidates[0].na, candidates[0].score
}

// selectLeastLoaded picks the node with the lowest current load.
func (s *ResourceAwareScheduler) selectLeastLoaded(eligible []*nodeAlloc) (*nodeAlloc, float64) {
	var best *nodeAlloc
	bestLoad := math.MaxFloat64
	for _, na := range eligible {
		totalCPU := float64(na.state.CPUCores) * 1000
		usedCPU := na.allocatedCPU + (na.state.CPUUsage / 100 * totalCPU)
		load := usedCPU / totalCPU
		if load < bestLoad {
			bestLoad = load
			best = na
		}
	}
	if best == nil {
		return nil, 0
	}
	return best, 1.0 - bestLoad
}

// selectRoundRobin picks nodes in round-robin fashion.
func (s *ResourceAwareScheduler) selectRoundRobin(eligible []*nodeAlloc) (*nodeAlloc, float64) {
	selected := eligible[s.rrIndex%len(eligible)]
	s.rrIndex++
	return selected, 1.0
}

// toNodeState converts internal allocation state to NodeState.
func (na *nodeAlloc) toNodeState() NodeState {
	totalCPU := float64(na.state.CPUCores) * 1000
	return NodeState{
		NodeID:      na.state.NodeID,
		NodeName:    na.state.NodeName,
		Address:     na.state.Address,
		CPUUsage:    na.state.CPUUsage + (na.allocatedCPU / totalCPU * 100),
		MemoryUsage: na.state.MemoryUsage + (float64(na.allocatedMem) / float64(na.state.MemoryBytes) * 100),
		DiskUsage:   na.state.DiskUsage,
		CPUCores:    na.state.CPUCores,
		MemoryBytes: na.state.MemoryBytes,
		DiskBytes:   na.state.DiskBytes,
		GPUCount:    na.state.GPUCount,
		AgentCount:  na.agentCount,
		Health:      na.state.Health,
		LastUpdate:  na.lastUpdate,
	}
}

// Ensure interface compliance.
var _ Scheduler = (*ResourceAwareScheduler)(nil)
