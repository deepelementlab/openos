// Package governance implements data retention, compliance checks, and billing helpers.
package governance

import (
	"context"
	"fmt"
	"time"
)

// RetentionPolicy describes table TTL enforcement.
type RetentionPolicy struct {
	Table          string
	MaxAge         time.Duration
	DeleteBatchSize int
}

// Enforcer applies retention (call from cron job).
type Enforcer struct{}

// Enforce is a stub; wire to repository deletes.
func (e Enforcer) Enforce(ctx context.Context, p RetentionPolicy) (int64, error) {
	if p.Table == "" || p.MaxAge <= 0 {
		return 0, fmt.Errorf("governance: invalid retention policy")
	}
	return 0, nil
}
