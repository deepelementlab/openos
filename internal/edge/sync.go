package edge

import (
	"context"
	"time"
)

// SyncMode controls edge-center data flow.
type SyncMode string

const (
	SyncRealtime SyncMode = "realtime"
	SyncBatch    SyncMode = "batch"
	SyncDeferred SyncMode = "deferred"
)

// SyncPlanner picks a mode based on connectivity.
func SyncPlanner(offline bool, latency time.Duration) SyncMode {
	if offline {
		return SyncDeferred
	}
	if latency > 200*time.Millisecond {
		return SyncBatch
	}
	return SyncRealtime
}

// Reconcile runs when link restores after partition.
func Reconcile(ctx context.Context, nodeID string) error {
	return nil
}
