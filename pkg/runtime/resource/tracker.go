package resource

import (
	"context"
	"fmt"
	"sync"
)

// ResourceAllocation describes resources assigned to one agent.
type ResourceAllocation struct {
	AgentID    string  `json:"agent_id"`
	CPUShares  float64 `json:"cpu_shares"`
	MemoryMB   int64   `json:"memory_mb"`
	DiskMB     int64   `json:"disk_mb"`
}

// QuotaConfig defines resource limits for tracking.
type QuotaConfig struct {
	MaxCPU    float64 `json:"max_cpu"`
	MaxMemMB  int64   `json:"max_mem_mb"`
	MaxDiskMB int64   `json:"max_disk_mb"`
}

// ResourceTracker monitors runtime-level resource allocation.
type ResourceTracker interface {
	Allocate(ctx context.Context, alloc ResourceAllocation) error
	Release(ctx context.Context, agentID string) error
	Get(ctx context.Context, agentID string) (*ResourceAllocation, error)
	ListAllocations(ctx context.Context) ([]ResourceAllocation, error)
	TotalUsed(ctx context.Context) (cpu float64, memMB int64, diskMB int64)
	CheckQuota(ctx context.Context, alloc ResourceAllocation) error
}

// DefaultResourceTracker is an in-memory resource tracker.
type DefaultResourceTracker struct {
	mu          sync.RWMutex
	allocations map[string]ResourceAllocation
	quota       QuotaConfig
}

func NewDefaultResourceTracker(quota QuotaConfig) *DefaultResourceTracker {
	return &DefaultResourceTracker{
		allocations: make(map[string]ResourceAllocation),
		quota:       quota,
	}
}

func (t *DefaultResourceTracker) Allocate(_ context.Context, alloc ResourceAllocation) error {
	if alloc.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.allocations[alloc.AgentID]; exists {
		return fmt.Errorf("allocation for agent %s already exists", alloc.AgentID)
	}

	if err := t.checkQuotaLocked(alloc); err != nil {
		return err
	}

	t.allocations[alloc.AgentID] = alloc
	return nil
}

func (t *DefaultResourceTracker) Release(_ context.Context, agentID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.allocations[agentID]; !ok {
		return fmt.Errorf("no allocation for agent %s", agentID)
	}
	delete(t.allocations, agentID)
	return nil
}

func (t *DefaultResourceTracker) Get(_ context.Context, agentID string) (*ResourceAllocation, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	alloc, ok := t.allocations[agentID]
	if !ok {
		return nil, fmt.Errorf("no allocation for agent %s", agentID)
	}
	return &alloc, nil
}

func (t *DefaultResourceTracker) ListAllocations(_ context.Context) ([]ResourceAllocation, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]ResourceAllocation, 0, len(t.allocations))
	for _, a := range t.allocations {
		out = append(out, a)
	}
	return out, nil
}

func (t *DefaultResourceTracker) TotalUsed(_ context.Context) (float64, int64, int64) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var cpu float64
	var mem, disk int64
	for _, a := range t.allocations {
		cpu += a.CPUShares
		mem += a.MemoryMB
		disk += a.DiskMB
	}
	return cpu, mem, disk
}

func (t *DefaultResourceTracker) CheckQuota(_ context.Context, alloc ResourceAllocation) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.checkQuotaLocked(alloc)
}

func (t *DefaultResourceTracker) checkQuotaLocked(alloc ResourceAllocation) error {
	var usedCPU float64
	var usedMem, usedDisk int64
	for _, a := range t.allocations {
		usedCPU += a.CPUShares
		usedMem += a.MemoryMB
		usedDisk += a.DiskMB
	}

	if t.quota.MaxCPU > 0 && usedCPU+alloc.CPUShares > t.quota.MaxCPU {
		return fmt.Errorf("CPU quota exceeded: used=%.2f requested=%.2f max=%.2f", usedCPU, alloc.CPUShares, t.quota.MaxCPU)
	}
	if t.quota.MaxMemMB > 0 && usedMem+alloc.MemoryMB > t.quota.MaxMemMB {
		return fmt.Errorf("memory quota exceeded: used=%dMB requested=%dMB max=%dMB", usedMem, alloc.MemoryMB, t.quota.MaxMemMB)
	}
	if t.quota.MaxDiskMB > 0 && usedDisk+alloc.DiskMB > t.quota.MaxDiskMB {
		return fmt.Errorf("disk quota exceeded: used=%dMB requested=%dMB max=%dMB", usedDisk, alloc.DiskMB, t.quota.MaxDiskMB)
	}
	return nil
}

var _ ResourceTracker = (*DefaultResourceTracker)(nil)
