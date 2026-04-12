// Package checkpoint provides agent state checkpointing (minimal API for AOS roadmap).
package checkpoint

import (
	"context"
	"time"
)

// SnapshotID identifies a checkpoint snapshot.
type SnapshotID string

// Manager coordinates checkpoint create/restore.
type Manager struct{}

// NewManager creates a checkpoint manager.
func NewManager() *Manager {
	return &Manager{}
}

// CreateCheckpoint records a logical checkpoint for an agent (implementation hooks to runtime).
func (m *Manager) CreateCheckpoint(ctx context.Context, agentID string) (SnapshotID, error) {
	_ = ctx
	_ = agentID
	return SnapshotID("stub-" + time.Now().Format("20060102150405")), nil
}
