package validation

import (
	"fmt"
	"time"
)

// RolloutPhase for blue/green or shadow traffic validation.
type RolloutPhase string

const (
	PhaseShadow   RolloutPhase = "shadow"
	PhaseCanary   RolloutPhase = "canary"
	PhaseFull     RolloutPhase = "full"
)

// RolloutPlan describes progressive delivery.
type RolloutPlan struct {
	NewVersion   string
	ShadowPercent int
	CanaryPercent int
	AutoRollback bool
}

// Validate checks plan invariants.
func (p RolloutPlan) Validate() error {
	if p.NewVersion == "" {
		return fmt.Errorf("validation: version required")
	}
	if p.ShadowPercent < 0 || p.ShadowPercent > 100 {
		return fmt.Errorf("validation: shadow percent invalid")
	}
	return nil
}

// DisasterRecoveryCheckpoint is a marker for DR drills.
type DisasterRecoveryCheckpoint struct {
	Timestamp time.Time
	Note      string
}
