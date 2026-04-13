// Package prediction hosts predictive heuristics for proactive remediation.
package prediction

import (
	"time"
)

// NodeSignal is a lightweight health/resource signal.
type NodeSignal struct {
	NodeID       string
	CPUUtil      float64
	MemUtil      float64
	ErrorRate    float64
	NetworkLoss  float64
	ObservedTime time.Time
}

// FaultRisk estimates likelihood of imminent failure (0-1).
func FaultRisk(s NodeSignal) float64 {
	risk := 0.0
	if s.CPUUtil > 0.92 {
		risk += 0.25
	}
	if s.MemUtil > 0.94 {
		risk += 0.35
	}
	if s.ErrorRate > 0.05 {
		risk += 0.25
	}
	if s.NetworkLoss > 0.02 {
		risk += 0.15
	}
	if risk > 1 {
		return 1
	}
	return risk
}

// ShouldEvacuate returns true if risk exceeds threshold.
func ShouldEvacuate(s NodeSignal, threshold float64) bool {
	return FaultRisk(s) >= threshold
}

// MaintenanceWindow is a planned outage interval for automatic migration triggers.
type MaintenanceWindow struct {
	NodeID    string
	Start     time.Time
	End       time.Time
	DrainOnly bool
}

// Active reports whether t falls in the window.
func (m MaintenanceWindow) Active(t time.Time) bool {
	return !t.Before(m.Start) && !t.After(m.End)
}
