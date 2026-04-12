// Package facade provides the AOS RuntimeFacade: a single entry point over
// containerd / gVisor / Kata backends using the existing interfaces.Runtime contract.
package facade

import (
	"context"
	"fmt"

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
}

// NewRuntimeFacade creates a facade without an active runtime; call Connect first.
func NewRuntimeFacade() *RuntimeFacade {
	return &RuntimeFacade{
		factory: aosruntime.NewFactory(),
	}
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
