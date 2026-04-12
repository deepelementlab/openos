package slo

import "fmt"

// Gate evaluates release readiness against documented SLO floors (stub thresholds).
type Gate struct {
	MinStartSuccessRate float64
}

// NewGate creates a release gate with default 0.99 start success rate floor.
func NewGate() *Gate {
	return &Gate{MinStartSuccessRate: 0.99}
}

// Evaluate returns an error if SLO data fails the gate.
func (g *Gate) Evaluate(c *Collector) error {
	if c == nil {
		return fmt.Errorf("slo: nil collector")
	}
	if c.StartSuccessRate() < g.MinStartSuccessRate {
		return fmt.Errorf("slo: start success rate below threshold")
	}
	return nil
}
