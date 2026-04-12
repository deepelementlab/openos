package resilience

import "context"

// Healer restarts or migrates failed agents (policy stub).
type Healer struct{}

// NewHealer creates a healer.
func NewHealer() *Healer {
	return &Healer{}
}

// OnFailure is invoked when probes fail repeatedly.
func (h *Healer) OnFailure(ctx context.Context, agentID string) error {
	_ = ctx
	_ = agentID
	return nil
}
