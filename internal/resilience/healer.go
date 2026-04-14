package resilience

import (
	"context"
	"sync"
	"time"
)

// FailurePolicy selects how to react after repeated probe failures.
type FailurePolicy string

const (
	PolicyRestart   FailurePolicy = "restart"
	PolicyMigrate   FailurePolicy = "migrate"
	PolicyDegrade   FailurePolicy = "degrade"
	PolicyNoop      FailurePolicy = "noop"
)

// Healer restarts or migrates failed agents with exponential backoff between restart attempts.
type Healer struct {
	mu sync.Mutex

	Policy             FailurePolicy
	MaxRestartAttempts int
	// backoff maps agentID -> next delay
	backoff map[string]time.Duration
	// attempts counts restart tries per agent
	attempts map[string]int

	OnRestart func(ctx context.Context, agentID string, attempt int, delay time.Duration) error
	OnMigrate func(ctx context.Context, agentID, reason string) error
	OnDegrade func(ctx context.Context, agentID string) error
}

// NewHealer creates a healer with restart policy and bounded backoff.
func NewHealer() *Healer {
	return &Healer{
		Policy:             PolicyRestart,
		MaxRestartAttempts: 5,
		backoff:            make(map[string]time.Duration),
		attempts:           make(map[string]int),
	}
}

// OnFailure is invoked when probes fail repeatedly.
func (h *Healer) OnFailure(ctx context.Context, agentID string) error {
	if h == nil || agentID == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	switch h.Policy {
	case PolicyMigrate:
		if h.OnMigrate != nil {
			return h.OnMigrate(ctx, agentID, "probe failure")
		}
		return nil
	case PolicyDegrade:
		if h.OnDegrade != nil {
			return h.OnDegrade(ctx, agentID)
		}
		return nil
	case PolicyNoop:
		return nil
	default: // restart
		n := h.attempts[agentID] + 1
		h.attempts[agentID] = n
		if h.MaxRestartAttempts > 0 && n > h.MaxRestartAttempts {
			if h.OnMigrate != nil {
				return h.OnMigrate(ctx, agentID, "max restart attempts exceeded")
			}
			return nil
		}
		d := h.backoff[agentID]
		if d == 0 {
			d = time.Second
		}
		next := d * 2
		if next > 5*time.Minute {
			next = 5 * time.Minute
		}
		h.backoff[agentID] = next
		if h.OnRestart != nil {
			return h.OnRestart(ctx, agentID, n, d)
		}
		return nil
	}
}

// Reset clears backoff state for an agent after successful recovery.
func (h *Healer) Reset(agentID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	delete(h.backoff, agentID)
	delete(h.attempts, agentID)
	h.mu.Unlock()
}
