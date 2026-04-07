//go:build legacy_containerd_stub
// +build legacy_containerd_stub

package containerd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// LegacyContainerdRuntime is the legacy stub kept behind a build tag.
type LegacyContainerdRuntime struct {
	config *types.RuntimeConfig
}

func NewLegacyContainerdRuntime() *LegacyContainerdRuntime {
	return &LegacyContainerdRuntime{}
}

func (r *LegacyContainerdRuntime) Initialize(_ context.Context, config *types.RuntimeConfig) error {
	r.config = config
	return nil
}

func (r *LegacyContainerdRuntime) GetRuntimeInfo() *types.RuntimeInfo {
	return &types.RuntimeInfo{
		Type:    types.RuntimeContainerd,
		Name:    "containerd-legacy-stub",
		Version: "0.0.1",
	}
}

func (r *LegacyContainerdRuntime) CreateAgent(_ context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	if spec == nil {
		return nil, fmt.Errorf("agent spec is required")
	}
	return &types.Agent{
		ID:        spec.ID,
		Name:      spec.Name,
		Image:     spec.Image,
		State:     types.AgentStateCreated,
		CreatedAt: time.Now(),
		Labels:    spec.Labels,
	}, nil
}

func (r *LegacyContainerdRuntime) StartAgent(_ context.Context, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *LegacyContainerdRuntime) StopAgent(_ context.Context, agentID string, _ time.Duration) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *LegacyContainerdRuntime) DeleteAgent(_ context.Context, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *LegacyContainerdRuntime) GetAgent(_ context.Context, agentID string) (*types.Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return &types.Agent{
		ID:        agentID,
		Name:      "mock-agent",
		Image:     "nginx:latest",
		State:     types.AgentStateRunning,
		CreatedAt: time.Now().Add(-time.Hour),
	}, nil
}

func (r *LegacyContainerdRuntime) ListAgents(_ context.Context, _ *types.AgentFilter) ([]*types.Agent, error) {
	return []*types.Agent{}, nil
}

func (r *LegacyContainerdRuntime) ExecuteCommand(_ context.Context, agentID string, cmd *types.Command) (*types.CommandResult, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	if cmd == nil {
		return nil, fmt.Errorf("command is required")
	}
	return &types.CommandResult{ExitCode: 0}, nil
}

func (r *LegacyContainerdRuntime) GetAgentLogs(_ context.Context, agentID string, _ *types.LogOptions) (io.ReadCloser, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return io.NopCloser(&legacyEmptyReader{}), nil
}

func (r *LegacyContainerdRuntime) GetAgentStats(_ context.Context, agentID string) (*types.ResourceStats, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return &types.ResourceStats{Timestamp: time.Now()}, nil
}

func (r *LegacyContainerdRuntime) UpdateAgent(_ context.Context, agentID string, _ *types.AgentSpec) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *LegacyContainerdRuntime) ResizeAgentTerminal(_ context.Context, agentID string, _, _ uint) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *LegacyContainerdRuntime) AttachAgent(_ context.Context, agentID string, _ *types.AttachOptions) (io.ReadWriteCloser, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return nil, fmt.Errorf("attach not implemented")
}

func (r *LegacyContainerdRuntime) HealthCheck(_ context.Context) error { return nil }
func (r *LegacyContainerdRuntime) Cleanup(_ context.Context) error    { return nil }

type legacyEmptyReader struct{}

func (r *legacyEmptyReader) Read(_ []byte) (int, error) { return 0, io.EOF }

var _ interfaces.Runtime = (*LegacyContainerdRuntime)(nil)
