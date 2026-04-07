package kata

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// KataRuntime is a placeholder for Kata Containers runtime support.
type KataRuntime struct {
	config *types.RuntimeConfig
}

func NewRuntime(_ context.Context, _ *types.RuntimeConfig) (interfaces.Runtime, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) Initialize(_ context.Context, config *types.RuntimeConfig) error {
	r.config = config
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) GetRuntimeInfo() *types.RuntimeInfo {
	return &types.RuntimeInfo{
		Type:    types.RuntimeKata,
		Name:    "kata",
		Version: "0.0.1",
	}
}

func (r *KataRuntime) CreateAgent(_ context.Context, _ *types.AgentSpec) (*types.Agent, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) StartAgent(_ context.Context, _ string) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) StopAgent(_ context.Context, _ string, _ time.Duration) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) DeleteAgent(_ context.Context, _ string) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) GetAgent(_ context.Context, _ string) (*types.Agent, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) ListAgents(_ context.Context, _ *types.AgentFilter) ([]*types.Agent, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) ExecuteCommand(_ context.Context, _ string, _ *types.Command) (*types.CommandResult, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) GetAgentLogs(_ context.Context, _ string, _ *types.LogOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) GetAgentStats(_ context.Context, _ string) (*types.ResourceStats, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) UpdateAgent(_ context.Context, _ string, _ *types.AgentSpec) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) ResizeAgentTerminal(_ context.Context, _ string, _, _ uint) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) AttachAgent(_ context.Context, _ string, _ *types.AttachOptions) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) HealthCheck(_ context.Context) error {
	return fmt.Errorf("kata runtime is not yet supported")
}

func (r *KataRuntime) Cleanup(_ context.Context) error {
	return nil
}

var _ interfaces.Runtime = (*KataRuntime)(nil)
