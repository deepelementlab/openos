// Package slo collects SLI metrics for release gates (see architecture/slo-release-gate.md).
package slo

import (
	"sort"
	"sync"
	"time"
)

// AgentStartSample records one start attempt outcome and latency ms.
type AgentStartSample struct {
	Success   bool
	LatencyMS int64
}

// Collector holds rolling samples (in-memory stub; wire to Prometheus in production).
type Collector struct {
	mu       sync.Mutex
	starts   []AgentStartSample
	apiErrs  int64
	apiTotal int64
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

// StartLatencyP99 returns a simple P99 estimate from recorded start latencies (ms).
func (c *Collector) StartLatencyP99() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.starts) == 0 {
		return 0
	}
	lat := make([]int64, 0, len(c.starts))
	for _, x := range c.starts {
		if x.Success {
			lat = append(lat, x.LatencyMS)
		}
	}
	if len(lat) == 0 {
		return 0
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	idx := (len(lat) * 99 / 100)
	if idx >= len(lat) {
		idx = len(lat) - 1
	}
	return time.Duration(lat[idx]) * time.Millisecond
}

// RecordAPI records API call outcome for error ratio SLI.
func (c *Collector) RecordAPI(success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiTotal++
	if !success {
		c.apiErrs++
	}
}

// APIErrorRatio returns apiErrs/apiTotal (0 if no samples).
func (c *Collector) APIErrorRatio() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.apiTotal == 0 {
		return 0
	}
	return float64(c.apiErrs) / float64(c.apiTotal)
}
