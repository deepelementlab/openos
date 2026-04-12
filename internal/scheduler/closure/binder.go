// Package closure implements schedule → bind → verify steps for AOS scheduling closure.
package closure

import "context"

// BindRequest carries scheduler output to a node/runtime.
type BindRequest struct {
	AgentID  string
	NodeID   string
	Priority int
}

// Binder binds a scheduling decision to concrete node resources.
type Binder struct{}

// NewBinder creates a binder.
func NewBinder() *Binder {
	return &Binder{}
}

// Bind records the binding (stub: integrate with node agent / kubelet later).
func (b *Binder) Bind(ctx context.Context, req *BindRequest) error {
	_ = ctx
	_ = req
	return nil
}
