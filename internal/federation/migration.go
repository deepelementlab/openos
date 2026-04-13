package federation

import (
	"context"
	"fmt"
	"time"
)

// MigrationState tracks cross-cluster agent moves.
type MigrationState string

const (
	MigrationPending   MigrationState = "pending"
	MigrationRunning   MigrationState = "running"
	MigrationCompleted MigrationState = "completed"
	MigrationFailed    MigrationState = "failed"
)

// AgentMigration describes a relocation between clusters.
type AgentMigration struct {
	AgentID    string
	FromCluster string
	ToCluster   string
	State      MigrationState
	Started    time.Time
	Updated    time.Time
}

// MigrationController coordinates checkpoints + cutover (integration points for runtime).
type MigrationController struct {
	reg *Registry
}

// NewMigrationController creates a controller.
func NewMigrationController(reg *Registry) *MigrationController {
	return &MigrationController{reg: reg}
}

// Start schedules a migration (validates clusters exist).
func (m *MigrationController) Start(ctx context.Context, agentID, from, to string) (*AgentMigration, error) {
	if _, ok := m.reg.Get(from); !ok {
		return nil, fmt.Errorf("migration: unknown source cluster %s", from)
	}
	if _, ok := m.reg.Get(to); !ok {
		return nil, fmt.Errorf("migration: unknown target cluster %s", to)
	}
	now := time.Now()
	return &AgentMigration{
		AgentID:     agentID,
		FromCluster: from,
		ToCluster:   to,
		State:       MigrationPending,
		Started:     now,
		Updated:     now,
	}, nil
}
