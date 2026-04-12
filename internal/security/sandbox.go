package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SandboxType represents the type of security sandbox.
type SandboxType string

const (
	SandboxTypeContainerd SandboxType = "containerd"
	SandboxTypeGVisor     SandboxType = "gvisor"
	SandboxTypeKata       SandboxType = "kata"
	SandboxTypeNone       SandboxType = "none"
)

// SandboxConfig defines the configuration for an agent sandbox.
type SandboxConfig struct {
	ID             string            `json:"id"`
	AgentID        string            `json:"agent_id"`
	Type           SandboxType       `json:"type"`
	ReadOnlyRootFS bool              `json:"read_only_rootfs"`
	RunAsUser      int               `json:"run_as_user"`
	RunAsGroup     int               `json:"run_as_group"`
	Capabilities   []string          `json:"capabilities,omitempty"`
	SeccompProfile string            `json:"seccomp_profile,omitempty"`
	SELinuxLabel   string            `json:"selinux_label,omitempty"`
	ResourceLimits *ResourceLimits   `json:"resource_limits,omitempty"`
	MountPoints    []MountPoint      `json:"mount_points,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// ResourceLimits defines resource constraints for a sandbox.
type ResourceLimits struct {
	CPULimit    int64 `json:"cpu_limit"`     // millicores
	MemoryLimit int64 `json:"memory_limit"`  // bytes
	PidsLimit   int64 `json:"pids_limit"`    // max processes
	NoNewPrivs  bool  `json:"no_new_privs"`  // prevent privilege escalation
}

// MountPoint defines a volume mount in the sandbox.
type MountPoint struct {
	Source     string `json:"source"`
	Dest       string `json:"dest"`
	ReadOnly   bool   `json:"read_only"`
	FSType     string `json:"fs_type,omitempty"`
	Propagation string `json:"propagation,omitempty"`
}

// SandboxState represents the runtime state of a sandbox.
type SandboxState struct {
	ID        string      `json:"id"`
	AgentID   string      `json:"agent_id"`
	Status    string      `json:"status"` // creating, running, stopped, error
	PID       int         `json:"pid,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	ExitCode  int         `json:"exit_code,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// SandboxManager manages agent sandboxes.
type SandboxManager interface {
	CreateSandbox(ctx context.Context, config *SandboxConfig) (*SandboxState, error)
	DestroySandbox(ctx context.Context, sandboxID string) error
	GetSandboxState(ctx context.Context, sandboxID string) (*SandboxState, error)
	ListSandboxes(ctx context.Context) ([]*SandboxState, error)
	ExecInSandbox(ctx context.Context, sandboxID string, cmd []string) (int, error)
}

// InMemorySandboxManager is an in-memory sandbox manager for development.
type InMemorySandboxManager struct {
	mu        sync.RWMutex
	sandboxes map[string]*SandboxState
	configs   map[string]*SandboxConfig
	nextPID   int
}

// NewInMemorySandboxManager creates a new in-memory sandbox manager.
func NewInMemorySandboxManager() *InMemorySandboxManager {
	return &InMemorySandboxManager{
		sandboxes: make(map[string]*SandboxState),
		configs:   make(map[string]*SandboxConfig),
		nextPID:   10000,
	}
}

// CreateSandbox creates a new sandbox.
func (m *InMemorySandboxManager) CreateSandbox(_ context.Context, config *SandboxConfig) (*SandboxState, error) {
	if config.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if config.Type == "" {
		config.Type = SandboxTypeContainerd
	}

	// Validate sandbox type
	switch config.Type {
	case SandboxTypeContainerd, SandboxTypeGVisor, SandboxTypeKata:
		// valid
	default:
		return nil, fmt.Errorf("unsupported sandbox type: %s", config.Type)
	}

	// Validate resource limits
	if config.ResourceLimits != nil {
		if config.ResourceLimits.CPULimit > 0 && config.ResourceLimits.CPULimit < 100 {
			return nil, fmt.Errorf("cpu_limit must be at least 100m (0.1 core)")
		}
		if config.ResourceLimits.MemoryLimit > 0 && config.ResourceLimits.MemoryLimit < 65536 {
			return nil, fmt.Errorf("memory_limit must be at least 64KB")
		}
		if config.ResourceLimits.PidsLimit > 0 && config.ResourceLimits.PidsLimit < 1 {
			return nil, fmt.Errorf("pids_limit must be at least 1")
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := config.ID
	if id == "" {
		id = fmt.Sprintf("sandbox-%d", len(m.sandboxes)+1)
	}

	state := &SandboxState{
		ID:        id,
		AgentID:   config.AgentID,
		Status:    "running",
		PID:       m.nextPID,
		CreatedAt: time.Now().UTC(),
	}
	m.nextPID++

	m.sandboxes[id] = state
	m.configs[id] = config

	return state, nil
}

// DestroySandbox destroys a sandbox.
func (m *InMemorySandboxManager) DestroySandbox(_ context.Context, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}

	if state.Status != "running" {
		return fmt.Errorf("sandbox %s is not running (status: %s)", sandboxID, state.Status)
	}

	state.Status = "stopped"
	state.ExitCode = 0
	return nil
}

// GetSandboxState returns the state of a sandbox.
func (m *InMemorySandboxManager) GetSandboxState(_ context.Context, sandboxID string) (*SandboxState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sandboxes[sandboxID]
	if !ok {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID)
	}
	// Return a copy
	cp := *state
	return &cp, nil
}

// ListSandboxes lists all sandboxes.
func (m *InMemorySandboxManager) ListSandboxes(_ context.Context) ([]*SandboxState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SandboxState, 0, len(m.sandboxes))
	for _, state := range m.sandboxes {
		cp := *state
		result = append(result, &cp)
	}
	return result, nil
}

// ExecInSandbox executes a command inside a sandbox.
func (m *InMemorySandboxManager) ExecInSandbox(_ context.Context, sandboxID string, cmd []string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sandboxes[sandboxID]
	if !ok {
		return -1, fmt.Errorf("sandbox %s not found", sandboxID)
	}

	if state.Status != "running" {
		return -1, fmt.Errorf("sandbox %s is not running", sandboxID)
	}

	if len(cmd) == 0 {
		return -1, fmt.Errorf("command cannot be empty")
	}

	// Simulate command execution
	return 0, nil
}

var _ SandboxManager = (*InMemorySandboxManager)(nil)
