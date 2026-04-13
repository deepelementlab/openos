package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager is the virtual memory / checkpoint interface for the Agent Kernel.
type Manager interface {
	MapRegion(ctx context.Context, agentID string, spec MemoryRegionSpec) (*MemoryRegion, error)
	UnmapRegion(ctx context.Context, regionID string) error
	ProtectRegion(ctx context.Context, regionID string, prot ProtectionFlags) error

	LoadWorkingSet(ctx context.Context, agentID string, set WorkingSet) error
	EvictWorkingSet(ctx context.Context, agentID string) error

	Checkpoint(ctx context.Context, agentID string) (*Checkpoint, error)
	Restore(ctx context.Context, cp *Checkpoint) (restoredAgentID string, err error)
	Migrate(ctx context.Context, agentID, targetNode string) error
}

// InMemoryManager tracks regions and checkpoints in memory.
type InMemoryManager struct {
	mu        sync.RWMutex
	regions   map[string]*MemoryRegion // regionID -> region
	byAgent   map[string][]string      // agentID -> region IDs
	checksums map[string]*Checkpoint   // checkpoint id
}

// NewInMemoryManager creates a manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		regions:   make(map[string]*MemoryRegion),
		byAgent:   make(map[string][]string),
		checksums: make(map[string]*Checkpoint),
	}
}

func (m *InMemoryManager) MapRegion(ctx context.Context, agentID string, spec MemoryRegionSpec) (*MemoryRegion, error) {
	if agentID == "" || spec.Size <= 0 {
		return nil, fmt.Errorf("memory: invalid map request")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rid := uuid.NewString()
	r := &MemoryRegion{
		RegionID:    rid,
		AgentID:     agentID,
		Start:       0,
		Size:        spec.Size,
		Protection:  spec.Protection,
		BackingType: spec.BackingType,
		BackingRef:  spec.BackingRef,
		Dirty:       false,
	}
	m.regions[rid] = r
	m.byAgent[agentID] = append(m.byAgent[agentID], rid)
	return cloneRegion(r), nil
}

func (m *InMemoryManager) UnmapRegion(ctx context.Context, regionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.regions[regionID]
	if !ok {
		return fmt.Errorf("memory: unknown region %s", regionID)
	}
	delete(m.regions, regionID)
	ids := m.byAgent[r.AgentID]
	out := ids[:0]
	for _, id := range ids {
		if id != regionID {
			out = append(out, id)
		}
	}
	m.byAgent[r.AgentID] = out
	return nil
}

func (m *InMemoryManager) ProtectRegion(ctx context.Context, regionID string, prot ProtectionFlags) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.regions[regionID]
	if !ok {
		return fmt.Errorf("memory: unknown region %s", regionID)
	}
	r.Protection = prot
	return nil
}

func (m *InMemoryManager) LoadWorkingSet(ctx context.Context, agentID string, set WorkingSet) error {
	// Stub: mark regions matching refs as hot
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rid := range m.byAgent[agentID] {
		r := m.regions[rid]
		if r == nil {
			continue
		}
		for _, ref := range set.Refs {
			if r.BackingRef == ref {
				r.Dirty = false
			}
		}
	}
	return nil
}

func (m *InMemoryManager) EvictWorkingSet(ctx context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rid := range m.byAgent[agentID] {
		if r := m.regions[rid]; r != nil {
			r.Dirty = false
		}
	}
	return nil
}

func (m *InMemoryManager) Checkpoint(ctx context.Context, agentID string) (*Checkpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cid := uuid.NewString()
	cp := &Checkpoint{
		ID:      cid,
		AgentID: agentID,
		Payload: []byte(fmt.Sprintf(`{"agent_id":%q,"ts":%q}`, agentID, time.Now().UTC().Format(time.RFC3339))),
	}
	m.checksums[cid] = cp
	return cp, nil
}

func (m *InMemoryManager) Restore(ctx context.Context, cp *Checkpoint) (string, error) {
	if cp == nil {
		return "", fmt.Errorf("memory: nil checkpoint")
	}
	return cp.AgentID, nil
}

func (m *InMemoryManager) Migrate(ctx context.Context, agentID, targetNode string) error {
	if targetNode == "" {
		return fmt.Errorf("memory: empty target node")
	}
	return nil
}

func cloneRegion(r *MemoryRegion) *MemoryRegion {
	cp := *r
	return &cp
}
