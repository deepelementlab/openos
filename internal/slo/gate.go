package slo

import (
	"fmt"
	"time"
)

// Gate evaluates release readiness against documented SLO floors.
type Gate struct {
	MinStartSuccessRate float64
	MaxP99StartLatency  time.Duration
	MaxAPIErrorRatio    float64
}

// NewGate creates a release gate with default 0.99 start success rate floor.
func NewGate() *Gate {
	return &Gate{
		MinStartSuccessRate: 0.99,
		MaxP99StartLatency:  5 * time.Second,
		MaxAPIErrorRatio:    0.01,
	}
}

// Evaluate returns an error if SLO data fails the gate.
func (g *Gate) Evaluate(c *Collector) error {
	if c == nil {
		return fmt.Errorf("slo: nil collector")
	}
	if g.MinStartSuccessRate > 0 && c.StartSuccessRate() < g.MinStartSuccessRate {
		return fmt.Errorf("slo: start success rate below threshold")
	}
	if g.MaxP99StartLatency > 0 {
		if p99 := c.StartLatencyP99(); p99 > g.MaxP99StartLatency {
			return fmt.Errorf("slo: p99 start latency %v exceeds %v", p99, g.MaxP99StartLatency)
		}
	}
	if g.MaxAPIErrorRatio >= 0 && c.APIErrorRatio() > g.MaxAPIErrorRatio {
		return fmt.Errorf("slo: API error ratio above threshold")
	}
	return nil
}
