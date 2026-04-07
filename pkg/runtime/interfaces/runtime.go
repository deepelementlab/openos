package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
)

// Runtime is the core interface for agent runtime implementations.
type Runtime interface {
	Initialize(ctx context.Context, config *types.RuntimeConfig) error
	GetRuntimeInfo() *types.RuntimeInfo
	CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error)
	StartAgent(ctx context.Context, agentID string) error
	StopAgent(ctx context.Context, agentID string, timeout time.Duration) error
	DeleteAgent(ctx context.Context, agentID string) error
	GetAgent(ctx context.Context, agentID string) (*types.Agent, error)
	ListAgents(ctx context.Context, filter *types.AgentFilter) ([]*types.Agent, error)
	ExecuteCommand(ctx context.Context, agentID string, cmd *types.Command) (*types.CommandResult, error)
	GetAgentLogs(ctx context.Context, agentID string, opts *types.LogOptions) (io.ReadCloser, error)
	GetAgentStats(ctx context.Context, agentID string) (*types.ResourceStats, error)
	UpdateAgent(ctx context.Context, agentID string, spec *types.AgentSpec) error
	ResizeAgentTerminal(ctx context.Context, agentID string, width, height uint) error
	AttachAgent(ctx context.Context, agentID string, opts *types.AttachOptions) (io.ReadWriteCloser, error)
	HealthCheck(ctx context.Context) error
	Cleanup(ctx context.Context) error
}

// SandboxRuntime extends Runtime with sandbox-specific capabilities.
type SandboxRuntime interface {
	Runtime
	CreateSandbox(ctx context.Context, spec *types.SandboxSpec) (*types.Sandbox, error)
	RemoveSandbox(ctx context.Context, sandboxID string) error
	JoinSandbox(ctx context.Context, agentID, sandboxID string) error
	LeaveSandbox(ctx context.Context, agentID, sandboxID string) error
	GetSandbox(ctx context.Context, sandboxID string) (*types.Sandbox, error)
	ListSandboxes(ctx context.Context) ([]*types.Sandbox, error)
	UpdateSandboxPolicy(ctx context.Context, sandboxID string, policy *types.SecurityPolicy) error
}

// NetworkRuntime extends Runtime with network management capabilities.
type NetworkRuntime interface {
	Runtime
	CreateNetwork(ctx context.Context, spec *types.NetworkSpec) (*types.Network, error)
	RemoveNetwork(ctx context.Context, networkID string) error
	ConnectAgentNetwork(ctx context.Context, agentID, networkID string) error
	DisconnectAgentNetwork(ctx context.Context, agentID, networkID string) error
	GetNetwork(ctx context.Context, networkID string) (*types.Network, error)
	ListNetworks(ctx context.Context) ([]*types.Network, error)
}

// VolumeRuntime extends Runtime with volume management capabilities.
type VolumeRuntime interface {
	Runtime
	CreateVolume(ctx context.Context, spec *types.VolumeSpec) (*types.Volume, error)
	RemoveVolume(ctx context.Context, volumeID string) error
	AttachVolume(ctx context.Context, agentID, volumeID string, mountPoint string) error
	DetachVolume(ctx context.Context, agentID, volumeID string) error
	GetVolume(ctx context.Context, volumeID string) (*types.Volume, error)
	ListVolumes(ctx context.Context) ([]*types.Volume, error)
}

// RuntimeFactory creates runtime instances based on configuration.
type RuntimeFactory interface {
	CreateRuntime(ctx context.Context, runtimeType string, config *types.RuntimeConfig) (Runtime, error)
	SupportedRuntimes() []string
}
