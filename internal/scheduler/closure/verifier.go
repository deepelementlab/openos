package closure

import "context"

// Verifier checks that a bound agent reaches Ready.
type Verifier struct{}

// NewVerifier creates a verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// VerifyReady returns nil when the agent passes readiness (stub).
func (v *Verifier) VerifyReady(ctx context.Context, agentID string) error {
	_ = ctx
	_ = agentID
	return nil
}
