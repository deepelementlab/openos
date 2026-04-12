// Package quota provides hard enforcement hooks on top of tenant.QuotaManager (cgroups binding TBD).
package quota

import (
	"context"
	"fmt"
)

// HardLimit describes maximum resources for a tenant pool.
type HardLimit struct {
	TenantID   string
	MaxCPUNano int64
	MaxMemory  int64
}

// Enforcer applies hard limits (stub: integrate with internal/resource/enforcer).
type Enforcer struct {
	limits map[string]*HardLimit
}

// NewEnforcer creates a quota hard-limit registry.
func NewEnforcer() *Enforcer {
	return &Enforcer{limits: make(map[string]*HardLimit)}
}

// SetHardLimit registers limits for a tenant.
func (e *Enforcer) SetHardLimit(l *HardLimit) {
	if l == nil || l.TenantID == "" {
		return
	}
	e.limits[l.TenantID] = l
}

// CheckWithinHardLimit returns an error if usage exceeds hard limit (stub values).
func (e *Enforcer) CheckWithinHardLimit(ctx context.Context, tenantID string, cpuNano, memBytes int64) error {
	_ = ctx
	l := e.limits[tenantID]
	if l == nil {
		return nil
	}
	if l.MaxMemory > 0 && memBytes > l.MaxMemory {
		return fmt.Errorf("quota: tenant %s exceeds memory hard limit", tenantID)
	}
	if l.MaxCPUNano > 0 && cpuNano > l.MaxCPUNano {
		return fmt.Errorf("quota: tenant %s exceeds CPU hard limit", tenantID)
	}
	return nil
}
