// Package slo collects SLI metrics for release gates (see architecture/slo-release-gate.md).
package slo

import "sync"

// AgentStartSample records one start attempt outcome and latency ms.
type AgentStartSample struct {
	Success   bool
	LatencyMS int64
}

// Collector holds rolling samples (in-memory stub; wire to Prometheus in production).
type Collector struct {
	mu      sync.Mutex
	starts  []AgentStartSample
	apiErrs int64
}

// NewCollector creates a collector.
func NewCollector() *Collector {
	return &Collector{starts: make([]AgentStartSample, 0)}
}

// RecordAgentStart appends a sample (cap in real impl).
func (c *Collector) RecordAgentStart(s AgentStartSample) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.starts = append(c.starts, s)
}

// StartSuccessRate returns fraction of successful starts in recorded samples.
func (c *Collector) StartSuccessRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.starts) == 0 {
		return 1.0
	}
	var ok int
	for _, x := range c.starts {
		if x.Success {
			ok++
		}
	}
	return float64(ok) / float64(len(c.starts))
}
