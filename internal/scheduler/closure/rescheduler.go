package closure

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Rescheduler tracks node heartbeats and triggers migration when a node is considered failed.
type Rescheduler struct {
	mu sync.RWMutex

	HeartbeatTimeout time.Duration
	lastBeat         map[string]time.Time
	pending          map[string]string

	// OnReschedule is invoked when an agent must move (integrate with scheduler queue).
	OnReschedule func(ctx context.Context, agentID, fromNode, reason string) error
}

// NewRescheduler creates a rescheduler with default 30s heartbeat timeout (per plan).
func NewRescheduler() *Rescheduler {
	return &Rescheduler{
		HeartbeatTimeout: 30 * time.Second,
		lastBeat:         make(map[string]time.Time),
		pending:          make(map[string]string),
	}
}

func (r *Rescheduler) ensureMaps() {
	if r == nil {
		return
	}
	if r.lastBeat == nil {
		r.lastBeat = make(map[string]time.Time)
	}
	if r.pending == nil {
		r.pending = make(map[string]string)
	}
}

// RecordHeartbeat updates last-seen time for a node (called by node agent / control plane).
func (r *Rescheduler) RecordHeartbeat(nodeID string, at time.Time) {
	if r == nil || nodeID == "" {
		return
	}
	r.ensureMaps()
	r.mu.Lock()
	r.lastBeat[nodeID] = at
	r.mu.Unlock()
}

// IsNodeHealthy returns false if no heartbeat or heartbeat older than HeartbeatTimeout.
func (r *Rescheduler) IsNodeHealthy(nodeID string) bool {
	if r == nil {
		return false
	}
	r.ensureMaps()
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.lastBeat[nodeID]
	if !ok {
		return false
	}
	timeout := r.HeartbeatTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return time.Since(t) < timeout
}

// MarkNodeFailed clears heartbeat so IsNodeHealthy is false (simulated failure).
func (r *Rescheduler) MarkNodeFailed(nodeID string) {
	if r == nil {
		return
	}
	r.ensureMaps()
	r.mu.Lock()
	delete(r.lastBeat, nodeID)
	r.mu.Unlock()
}

// Reschedule records a pending migration for agentID and optionally invokes OnReschedule.
func (r *Rescheduler) Reschedule(ctx context.Context, agentID, reason string) error {
	if r == nil || agentID == "" {
		return fmt.Errorf("closure: invalid reschedule")
	}
	r.ensureMaps()
	r.mu.Lock()
	r.pending[agentID] = reason
	r.mu.Unlock()
	if r.OnReschedule != nil {
		return r.OnReschedule(ctx, agentID, "", reason)
	}
	return nil
}

// RescheduleFromStore looks up the agent's node from BindingStore and enqueues reschedule.
func (r *Rescheduler) RescheduleFromStore(ctx context.Context, agentID string, store BindingStore, reason string) error {
	if r == nil {
		return fmt.Errorf("closure: nil rescheduler")
	}
	r.ensureMaps()
	var fromNode string
	if store != nil {
		if n, err := store.GetNode(ctx, agentID); err == nil {
			fromNode = n
		}
	}
	r.mu.Lock()
	r.pending[agentID] = reason
	r.mu.Unlock()
	if r.OnReschedule != nil {
		return r.OnReschedule(ctx, agentID, fromNode, reason)
	}
	return nil
}

// PendingReason returns reason if agent is pending reschedule.
func (r *Rescheduler) PendingReason(agentID string) (string, bool) {
	r.ensureMaps()
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.pending[agentID]
	return s, ok
}

// ClearPending removes pending state after successful rebind.
func (r *Rescheduler) ClearPending(agentID string) {
	r.ensureMaps()
	r.mu.Lock()
	delete(r.pending, agentID)
	r.mu.Unlock()
}
