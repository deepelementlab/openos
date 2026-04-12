// Package resilience implements health probes and basic healing policies.
package resilience

import (
	"context"
	"time"
)

// ProbeType mirrors k8s probe kinds.
type ProbeType string

const (
	ProbeLiveness  ProbeType = "Liveness"
	ProbeReadiness ProbeType = "Readiness"
	ProbeStartup   ProbeType = "Startup"
)

// Prober runs HTTP/TCP/exec probes (stub: always success).
type Prober struct {
	Timeout time.Duration
}

// NewProber creates a prober.
func NewProber() *Prober {
	return &Prober{Timeout: 5 * time.Second}
}

// Run executes a probe (stub implementation).
func (p *Prober) Run(ctx context.Context, t ProbeType, target string) error {
	_ = ctx
	_ = t
	_ = target
	return nil
}
