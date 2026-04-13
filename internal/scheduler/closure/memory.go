package closure

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MemoryReserver is an in-process reservation tracker for tests and single-node deployments.
type MemoryReserver struct {
	mu           sync.Mutex
	reservations map[string]string // id -> nodeID
}

// NewMemoryReserver creates a reserver.
func NewMemoryReserver() *MemoryReserver {
	return &MemoryReserver{reservations: make(map[string]string)}
}

// Reserve records a reservation and returns an ID.
func (m *MemoryReserver) Reserve(ctx context.Context, nodeID string, spec *ResourceSpec) (string, error) {
	_ = ctx
	if nodeID == "" {
		return "", fmt.Errorf("closure: nodeID required")
	}
	id := uuid.NewString()
	m.mu.Lock()
	m.reservations[id] = nodeID
	m.mu.Unlock()
	return id, nil
}

// Release removes a reservation.
func (m *MemoryReserver) Release(ctx context.Context, reservationID string) error {
	_ = ctx
	m.mu.Lock()
	delete(m.reservations, reservationID)
	m.mu.Unlock()
	return nil
}

// MemoryBindingStore binds agents to nodes in memory.
type MemoryBindingStore struct {
	mu    sync.RWMutex
	agent map[string]string // agentID -> nodeID
}

// NewMemoryBindingStore creates a store.
func NewMemoryBindingStore() *MemoryBindingStore {
	return &MemoryBindingStore{agent: make(map[string]string)}
}

// SaveBinding records the binding.
func (s *MemoryBindingStore) SaveBinding(ctx context.Context, agentID, nodeID string) error {
	_ = ctx
	s.mu.Lock()
	s.agent[agentID] = nodeID
	s.mu.Unlock()
	return nil
}

// GetNode returns the node for an agent.
func (s *MemoryBindingStore) GetNode(ctx context.Context, agentID string) (string, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.agent[agentID]
	if !ok {
		return "", fmt.Errorf("closure: no binding for %s", agentID)
	}
	return n, nil
}

// DeleteBinding removes a binding.
func (s *MemoryBindingStore) DeleteBinding(ctx context.Context, agentID string) error {
	_ = ctx
	s.mu.Lock()
	delete(s.agent, agentID)
	s.mu.Unlock()
	return nil
}
