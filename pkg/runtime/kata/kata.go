package kata

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// KataRuntime is an optional Kata Containers integration (containerd + kata shim is typical in production).
type KataRuntime struct {
	config   *types.RuntimeConfig
	kataPath string
}

// NewRuntime constructs a Kata runtime handle (operations delegate to shim/containerd when wired).
func NewRuntime(_ context.Context, _ *types.RuntimeConfig) (interfaces.Runtime, error) {
	return &KataRuntime{}, nil
}

func resolveKataBin(cfg *types.RuntimeConfig) string {
	if cfg != nil && cfg.Options != nil {
		if p, ok := cfg.Options["kata_runtime_path"].(string); ok && p != "" {
			return p
		}
	}
	if p, err := exec.LookPath("kata-runtime"); err == nil {
		return p
	}
	if p, err := exec.LookPath("kata-qemu"); err == nil {
		return p
	}
	return ""
}

func (r *KataRuntime) Initialize(_ context.Context, config *types.RuntimeConfig) error {
	r.config = config
	r.kataPath = resolveKataBin(config)
	return nil
}

func (r *KataRuntime) GetRuntimeInfo() *types.RuntimeInfo {
	return &types.RuntimeInfo{
		Type:    types.RuntimeKata,
		Name:    "kata",
		Version: "0.1.0",
	}
}

func (r *KataRuntime) CreateAgent(_ context.Context, _ *types.AgentSpec) (*types.Agent, error) {
	return nil, fmt.Errorf("kata: create agent requires containerd+kata shim wiring (set kata_runtime_path)")
}

func (r *KataRuntime) StartAgent(_ context.Context, _ string) error {
	return fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) StopAgent(_ context.Context, _ string, _ time.Duration) error {
	return fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) DeleteAgent(_ context.Context, _ string) error {
	return fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) GetAgent(_ context.Context, _ string) (*types.Agent, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) ListAgents(_ context.Context, _ *types.AgentFilter) ([]*types.Agent, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) ExecuteCommand(_ context.Context, _ string, _ *types.Command) (*types.CommandResult, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) GetAgentLogs(_ context.Context, _ string, _ *types.LogOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) GetAgentStats(_ context.Context, _ string) (*types.ResourceStats, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) UpdateAgent(_ context.Context, _ string, _ *types.AgentSpec) error {
	return fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) ResizeAgentTerminal(_ context.Context, _ string, _, _ uint) error {
	return fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) AttachAgent(_ context.Context, _ string, _ *types.AttachOptions) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("kata: not wired")
}

func (r *KataRuntime) HealthCheck(ctx context.Context) error {
	if r.kataPath == "" {
		return fmt.Errorf("kata: kata-runtime not found in PATH")
	}
	cmd := exec.CommandContext(ctx, r.kataPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kata: %w", err)
	}
	return nil
}

func (r *KataRuntime) Cleanup(_ context.Context) error {
	return nil
}

var _ interfaces.Runtime = (*KataRuntime)(nil)
