package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
)

// SandboxManager manages sandbox lifecycle and an in-memory sandbox registry.
type SandboxManager struct {
	mu        sync.RWMutex
	sandboxes map[string]*types.Sandbox
	agents    map[string]string // agentID -> sandboxID
}

func NewSandboxManager() *SandboxManager {
	return &SandboxManager{
		sandboxes: make(map[string]*types.Sandbox),
		agents:    make(map[string]string),
	}
}

// Create creates a new sandbox from a spec.
func (m *SandboxManager) Create(_ context.Context, spec *types.SandboxSpec) (*types.Sandbox, error) {
	if spec.ID == "" {
		return nil, fmt.Errorf("sandbox ID is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sandboxes[spec.ID]; exists {
		return nil, fmt.Errorf("sandbox %s already exists", spec.ID)
	}

	sb := &types.Sandbox{
		ID:             spec.ID,
		Name:           spec.Name,
		Type:           spec.Type,
		SecurityPolicy: spec.SecurityPolicy,
		AgentCount:     0,
		CreatedAt:      time.Now().UTC(),
		Labels:         spec.Labels,
		Annotations:    spec.Annotations,
	}
	m.sandboxes[spec.ID] = sb
	cp := *sb
	return &cp, nil
}

// Remove removes a sandbox. Fails if agents are still attached.
func (m *SandboxManager) Remove(_ context.Context, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sb, ok := m.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}
	if sb.AgentCount > 0 {
		return fmt.Errorf("sandbox %s still has %d agents attached", sandboxID, sb.AgentCount)
	}
	delete(m.sandboxes, sandboxID)
	return nil
}

// Join attaches an agent to a sandbox.
func (m *SandboxManager) Join(_ context.Context, agentID, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sb, ok := m.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}
	if existing, joined := m.agents[agentID]; joined {
		return fmt.Errorf("agent %s already in sandbox %s", agentID, existing)
	}
	m.agents[agentID] = sandboxID
	sb.AgentCount++
	return nil
}

// Leave detaches an agent from its sandbox.
func (m *SandboxManager) Leave(_ context.Context, agentID, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.agents[agentID]
	if !ok || current != sandboxID {
		return fmt.Errorf("agent %s is not in sandbox %s", agentID, sandboxID)
	}
	delete(m.agents, agentID)
	if sb, exists := m.sandboxes[sandboxID]; exists {
		sb.AgentCount--
	}
	return nil
}

// Get returns sandbox info.
func (m *SandboxManager) Get(_ context.Context, sandboxID string) (*types.Sandbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sb, ok := m.sandboxes[sandboxID]
	if !ok {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID)
	}
	cp := *sb
	return &cp, nil
}

// List returns all sandboxes.
func (m *SandboxManager) List(_ context.Context) ([]*types.Sandbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*types.Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		cp := *sb
		out = append(out, &cp)
	}
	return out, nil
}

// UpdatePolicy updates the security policy of a sandbox.
func (m *SandboxManager) UpdatePolicy(_ context.Context, sandboxID string, policy *types.SecurityPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sb, ok := m.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}
	sb.SecurityPolicy = policy
	return nil
}
