package closure

import (
	"context"
	"fmt"

	"github.com/agentos/aos/pkg/runtime/types"
)

// BindRequest carries scheduler output to a node/runtime.
type BindRequest struct {
	AgentID  string
	NodeID   string
	Priority int
	Spec     *ResourceSpec
	// AgentSpec is used when Binding is set.
	AgentSpec *types.AgentSpec
}

// Binder binds a scheduling decision: reserve → optional create agent → save binding (Saga compensation on failure).
type Binder struct {
	Reserver ResourceReserver
	Store    BindingStore
	// Binding provides Create+Delete for full runtime integration; nil skips container create.
	Binding BindingRuntime
}

// NewBinder creates a binder with in-memory reserver/store.
func NewBinder() *Binder {
	return &Binder{
		Reserver: NewMemoryReserver(),
		Store:    NewMemoryBindingStore(),
	}
}

// NewBinderWith injects dependencies.
func NewBinderWith(r ResourceReserver, s BindingStore, br BindingRuntime) *Binder {
	return &Binder{Reserver: r, Store: s, Binding: br}
}

// Bind executes reserve → optional create → save binding.
func (b *Binder) Bind(ctx context.Context, req *BindRequest) error {
	if req == nil || req.AgentID == "" || req.NodeID == "" {
		return fmt.Errorf("closure: invalid bind request")
	}
	if b.Reserver == nil || b.Store == nil {
		return fmt.Errorf("closure: reserver and store required")
	}
	spec := req.Spec
	if spec == nil {
		spec = &ResourceSpec{}
	}
	resID, err := b.Reserver.Reserve(ctx, req.NodeID, spec)
	if err != nil {
		return fmt.Errorf("reserve: %w", err)
	}
	release := func() { _ = b.Reserver.Release(ctx, resID) }

	var createdID string
	if b.Binding != nil && req.AgentSpec != nil {
		ag, err := b.Binding.CreateAgent(ctx, req.AgentSpec)
		if err != nil {
			release()
			return fmt.Errorf("create agent: %w", err)
		}
		if ag != nil && ag.ID != "" {
			createdID = ag.ID
		} else {
			createdID = req.AgentID
		}
	}

	if err := b.Store.SaveBinding(ctx, req.AgentID, req.NodeID); err != nil {
		if createdID != "" && b.Binding != nil {
			_ = b.Binding.DeleteAgent(ctx, createdID)
		}
		release()
		return fmt.Errorf("save binding: %w", err)
	}
	return nil
}

// BindSimple reserves and saves binding without runtime create.
func (b *Binder) BindSimple(ctx context.Context, agentID, nodeID string, spec *ResourceSpec) error {
	return b.Bind(ctx, &BindRequest{AgentID: agentID, NodeID: nodeID, Spec: spec})
}
