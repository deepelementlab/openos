package closure

import (
	"context"
	"fmt"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// ResourceReserver abstracts node capacity reservation (integrate with real scheduler later).
type ResourceReserver interface {
	Reserve(ctx context.Context, nodeID string, spec *ResourceSpec) (reservationID string, err error)
	Release(ctx context.Context, reservationID string) error
}

// ResourceSpec is a minimal resource request for binding.
type ResourceSpec struct {
	CPUMilli int64
	MemoryMB int64
}

// BindingStore persists agentID → nodeID (integrate with DB/etcd).
type BindingStore interface {
	SaveBinding(ctx context.Context, agentID, nodeID string) error
	GetNode(ctx context.Context, agentID string) (nodeID string, err error)
	DeleteBinding(ctx context.Context, agentID string) error
}

// BindingRuntime is used for saga: CreateAgent + DeleteAgent on rollback.
type BindingRuntime interface {
	CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error)
	DeleteAgent(ctx context.Context, agentID string) error
}

// FacadeRuntime adapts interfaces.Runtime to BindingRuntime.
type FacadeRuntime struct {
	RT interfaces.Runtime
}

// CreateAgent implements BindingRuntime.
func (f *FacadeRuntime) CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	if f == nil || f.RT == nil {
		return nil, fmt.Errorf("closure: runtime not connected")
	}
	return f.RT.CreateAgent(ctx, spec)
}

// DeleteAgent implements BindingRuntime.
func (f *FacadeRuntime) DeleteAgent(ctx context.Context, agentID string) error {
	if f == nil || f.RT == nil {
		return nil
	}
	return f.RT.DeleteAgent(ctx, agentID)
}

