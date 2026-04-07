package resource

import (
	"context"
	"fmt"
	"sync"
)

// NodeResources describes a node's total and allocated resources.
type NodeResources struct {
	NodeID         string  `json:"node_id"`
	TotalCPU       float64 `json:"total_cpu"`
	TotalMemory    int64   `json:"total_memory"`
	TotalDisk      int64   `json:"total_disk"`
	AllocatedCPU   float64 `json:"allocated_cpu"`
	AllocatedMemory int64  `json:"allocated_memory"`
	AllocatedDisk  int64   `json:"allocated_disk"`
}

// AvailableCPU returns remaining CPU.
func (n *NodeResources) AvailableCPU() float64    { return n.TotalCPU - n.AllocatedCPU }
// AvailableMemory returns remaining memory.
func (n *NodeResources) AvailableMemory() int64    { return n.TotalMemory - n.AllocatedMemory }
// AvailableDisk returns remaining disk.
func (n *NodeResources) AvailableDisk() int64      { return n.TotalDisk - n.AllocatedDisk }

// AllocationRequest describes resources to allocate.
type AllocationRequest struct {
	AgentID string  `json:"agent_id"`
	NodeID  string  `json:"node_id"`
	CPU     float64 `json:"cpu"`
	Memory  int64   `json:"memory"`
	Disk    int64   `json:"disk"`
}

// ResourceManager tracks resource allocation across nodes.
type ResourceManager interface {
	RegisterNode(ctx context.Context, node NodeResources) error
	UnregisterNode(ctx context.Context, nodeID string) error
	Allocate(ctx context.Context, req AllocationRequest) error
	Release(ctx context.Context, agentID string) error
	GetNode(ctx context.Context, nodeID string) (*NodeResources, error)
	ListNodes(ctx context.Context) ([]NodeResources, error)
	CheckQuota(ctx context.Context, req AllocationRequest) (bool, string)
}

// InMemoryResourceManager is an in-memory ResourceManager.
type InMemoryResourceManager struct {
	mu          sync.RWMutex
	nodes       map[string]*NodeResources
	allocations map[string]AllocationRequest // keyed by agentID
}

func NewInMemoryResourceManager() *InMemoryResourceManager {
	return &InMemoryResourceManager{
		nodes:       make(map[string]*NodeResources),
		allocations: make(map[string]AllocationRequest),
	}
}

func (m *InMemoryResourceManager) RegisterNode(_ context.Context, node NodeResources) error {
	if node.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := node
	m.nodes[node.NodeID] = &cp
	return nil
}

func (m *InMemoryResourceManager) UnregisterNode(_ context.Context, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[nodeID]; !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	delete(m.nodes, nodeID)
	return nil
}

func (m *InMemoryResourceManager) Allocate(_ context.Context, req AllocationRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodes[req.NodeID]
	if !ok {
		return fmt.Errorf("node %s not found", req.NodeID)
	}

	if node.AvailableCPU() < req.CPU {
		return fmt.Errorf("insufficient CPU on node %s", req.NodeID)
	}
	if node.AvailableMemory() < req.Memory {
		return fmt.Errorf("insufficient memory on node %s", req.NodeID)
	}
	if node.AvailableDisk() < req.Disk {
		return fmt.Errorf("insufficient disk on node %s", req.NodeID)
	}

	node.AllocatedCPU += req.CPU
	node.AllocatedMemory += req.Memory
	node.AllocatedDisk += req.Disk
	m.allocations[req.AgentID] = req
	return nil
}

func (m *InMemoryResourceManager) Release(_ context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alloc, ok := m.allocations[agentID]
	if !ok {
		return fmt.Errorf("no allocation found for agent %s", agentID)
	}
	if node, nodeOK := m.nodes[alloc.NodeID]; nodeOK {
		node.AllocatedCPU -= alloc.CPU
		node.AllocatedMemory -= alloc.Memory
		node.AllocatedDisk -= alloc.Disk
	}
	delete(m.allocations, agentID)
	return nil
}

func (m *InMemoryResourceManager) GetNode(_ context.Context, nodeID string) (*NodeResources, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	cp := *n
	return &cp, nil
}

func (m *InMemoryResourceManager) ListNodes(_ context.Context) ([]NodeResources, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]NodeResources, 0, len(m.nodes))
	for _, n := range m.nodes {
		out = append(out, *n)
	}
	return out, nil
}

func (m *InMemoryResourceManager) CheckQuota(_ context.Context, req AllocationRequest) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	node, ok := m.nodes[req.NodeID]
	if !ok {
		return false, fmt.Sprintf("node %s not found", req.NodeID)
	}
	if node.AvailableCPU() < req.CPU {
		return false, "insufficient CPU"
	}
	if node.AvailableMemory() < req.Memory {
		return false, "insufficient memory"
	}
	if node.AvailableDisk() < req.Disk {
		return false, "insufficient disk"
	}
	return true, "ok"
}

var _ ResourceManager = (*InMemoryResourceManager)(nil)
