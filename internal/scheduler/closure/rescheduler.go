package closure

import "context"

// Rescheduler triggers reschedule on failure.
type Rescheduler struct{}

// NewRescheduler creates a rescheduler.
func NewRescheduler() *Rescheduler {
	return &Rescheduler{}
}

// Reschedule requests a new placement for agentID (stub).
func (r *Rescheduler) Reschedule(ctx context.Context, agentID, reason string) error {
	_ = ctx
	_ = agentID
	_ = reason
	return nil
}
