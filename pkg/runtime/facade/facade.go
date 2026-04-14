// Package facade provides the AOS RuntimeFacade: a single entry point over
// containerd / gVisor / Kata backends using the existing interfaces.Runtime contract.
package facade

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/kernel"
	aosruntime "github.com/agentos/aos/pkg/runtime"
	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// Backend identifies which runtime implementation to use.
type Backend string

const (
	BackendContainerd Backend = "containerd"
	BackendGVisor     Backend = "gvisor"
	BackendKata       Backend = "kata"
)

// RuntimeFacade wraps interfaces.Runtime with AOS-oriented helpers and backend selection.
type RuntimeFacade struct {
	factory interfaces.RuntimeFactory
	rt      interfaces.Runtime
	backend Backend
	kernel  *kernel.Facade
}

// Option configures RuntimeFacade construction.
type Option func(*RuntimeFacade)

// WithKernel attaches the Agent Kernel facade so CreateAgent can register process groups/namespaces.
func WithKernel(k *kernel.Facade) Option {
	return func(f *RuntimeFacade) {
		f.kernel = k
	}
}

// NewRuntimeFacade creates a facade without an active runtime; call Connect first.
func NewRuntimeFacade(opts ...Option) *RuntimeFacade {
	f := &RuntimeFacade{
		factory: aosruntime.NewFactory(),
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// Kernel returns the kernel facade when configured with WithKernel.
func (f *RuntimeFacade) Kernel() *kernel.Facade {
	if f == nil {
		return nil
	}
	return f.kernel
}

// ConnectForSpec connects using backend hints from spec labels (aos.openos.dev/runtime).
func (f *RuntimeFacade) ConnectForSpec(ctx context.Context, spec *types.AgentSpec, cfg *types.RuntimeConfig) error {
	return f.Connect(ctx, SelectBackendForSpec(spec), cfg)
}

// Connect initializes the underlying runtime for the given backend.
func (f *RuntimeFacade) Connect(ctx context.Context, backend Backend, cfg *types.RuntimeConfig) error {
	if cfg == nil {
		cfg = &types.RuntimeConfig{}
	}
	rtType := string(types.RuntimeContainerd)
	switch backend {
	case BackendContainerd:
		rtType = string(types.RuntimeContainerd)
	case BackendGVisor:
		rtType = string(types.RuntimeGVisor)
	case BackendKata:
		rtType = string(types.RuntimeKata)
	default:
		return fmt.Errorf("facade: unsupported backend %q", backend)
	}
	rt, err := f.factory.CreateRuntime(ctx, rtType, cfg)
	if err != nil {
		return err
	}
	if err := rt.Initialize(ctx, cfg); err != nil {
		return err
	}
	f.rt = rt
	f.backend = backend
	return nil
}

// Runtime returns the active interfaces.Runtime or nil if not connected.
func (f *RuntimeFacade) Runtime() interfaces.Runtime {
	return f.rt
}

// Backend returns the selected backend after Connect.
func (f *RuntimeFacade) Backend() Backend {
	return f.backend
}

// SupportedBackends lists backends the factory can instantiate.
func (f *RuntimeFacade) SupportedBackends() []Backend {
	return []Backend{BackendContainerd, BackendGVisor, BackendKata}
}

// CreateAgent delegates to the connected runtime (Connect must have succeeded).
func (f *RuntimeFacade) CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	if f == nil || f.rt == nil {
		return nil, fmt.Errorf("facade: runtime not connected")
	}
	agent, err := f.rt.CreateAgent(ctx, spec)
	if err != nil || agent == nil || f.kernel == nil {
		return agent, err
	}
	agentID := agent.ID
	if agentID == "" && spec != nil {
		agentID = spec.ID
	}
	if agentID == "" {
		return agent, err
	}
	_, _ = f.kernel.Process.CreateGroup(ctx, agentID)
	if ns, nsErr := f.kernel.Process.CreateNamespace(ctx); nsErr == nil {
		_ = f.kernel.Process.EnterNamespace(ctx, agentID, ns)
	}
	return agent, err
}

// StartAgent starts an agent on the active backend.
func (f *RuntimeFacade) StartAgent(ctx context.Context, agentID string) error {
	if f == nil || f.rt == nil {
		return fmt.Errorf("facade: runtime not connected")
	}
	return f.rt.StartAgent(ctx, agentID)
}

// StopAgent stops an agent.
func (f *RuntimeFacade) StopAgent(ctx context.Context, agentID string, timeout time.Duration) error {
	if f == nil || f.rt == nil {
		return fmt.Errorf("facade: runtime not connected")
	}
	return f.rt.StopAgent(ctx, agentID, timeout)
}

// DeleteAgent removes an agent.
func (f *RuntimeFacade) DeleteAgent(ctx context.Context, agentID string) error {
	if f == nil || f.rt == nil {
		return fmt.Errorf("facade: runtime not connected")
	}
	return f.rt.DeleteAgent(ctx, agentID)
}

// SelectBackendForSpec picks a backend from labels/annotations (aos.openos.dev/runtime) or defaults to containerd.
func SelectBackendForSpec(spec *types.AgentSpec) Backend {
	if spec == nil || spec.Labels == nil {
		return BackendContainerd
	}
	switch spec.Labels["aos.openos.dev/runtime"] {
	case "gvisor", "runsc":
		return BackendGVisor
	case "kata":
		return BackendKata
	default:
		return BackendContainerd
	}
}
