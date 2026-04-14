package closure

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/resilience"
)

// VerifierConfig configures probe targets per phase (http(s):// URL, tcp host:port, exec:argv).
type VerifierConfig struct {
	StartupTarget   string
	LivenessTarget  string
	ReadinessTarget string
	StartupTimeout  time.Duration
	ProbeTimeout    time.Duration
}

// Verifier checks Startup → Liveness → Readiness using the shared Prober.
type Verifier struct {
	Prober *resilience.Prober
	Config VerifierConfig
}

// NewVerifier creates a verifier with default prober.
func NewVerifier() *Verifier {
	return &Verifier{
		Prober: resilience.NewProber(),
		Config: VerifierConfig{
			StartupTimeout: 60 * time.Second,
			ProbeTimeout:   5 * time.Second,
		},
	}
}

// NewVerifierWith injects a custom prober (e.g. tests).
func NewVerifierWith(p *resilience.Prober, cfg VerifierConfig) *Verifier {
	if p == nil {
		p = resilience.NewProber()
	}
	return &Verifier{Prober: p, Config: cfg}
}

// VerifyReady runs Startup (with retries until StartupTimeout), then Liveness, then Readiness.
func (v *Verifier) VerifyReady(ctx context.Context, agentID string) error {
	_ = agentID
	if v.Prober == nil {
		return nil
	}
	cfg := v.Config
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 60 * time.Second
	}
	if cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = 5 * time.Second
	}
	v.Prober.Timeout = cfg.ProbeTimeout

	if t := cfg.StartupTarget; t != "" {
		var last error
		deadline := time.Now().Add(cfg.StartupTimeout)
		for time.Now().Before(deadline) {
			sctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
			last = v.Prober.Run(sctx, resilience.ProbeStartup, t)
			cancel()
			if last == nil {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
		}
		if last != nil {
			return fmt.Errorf("startup probe: %w", last)
		}
	}

	if t := cfg.LivenessTarget; t != "" {
		sctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
		defer cancel()
		if err := v.Prober.Run(sctx, resilience.ProbeLiveness, t); err != nil {
			return fmt.Errorf("liveness probe: %w", err)
		}
	}
	if t := cfg.ReadinessTarget; t != "" {
		sctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
		defer cancel()
		if err := v.Prober.Run(sctx, resilience.ProbeReadiness, t); err != nil {
			return fmt.Errorf("readiness probe: %w", err)
		}
	}
	return nil
}
