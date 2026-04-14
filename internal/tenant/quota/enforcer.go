// Package quota provides hard enforcement hooks on top of tenant.QuotaManager (cgroups binding TBD).
package quota

import (
	"context"
	"errors"
	"fmt"

	"github.com/agentos/aos/internal/resource/enforcer"
)

// HardLimit describes maximum resources for a tenant pool.
type HardLimit struct {
	TenantID   string
	MaxCPUNano int64
	MaxMemory  int64
	// CgroupGroup is optional relative path under cgroup v2 root (linux).
	CgroupGroup string
}

// Enforcer applies hard limits and optional cgroup writes.
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

// CheckWithinHardLimit returns ErrResourceExhausted when usage exceeds registered hard limits.
func (e *Enforcer) CheckWithinHardLimit(ctx context.Context, tenantID string, cpuNano, memBytes int64) error {
	_ = ctx
	l := e.limits[tenantID]
	if l == nil {
		return nil
	}
	if l.MaxMemory > 0 && memBytes > l.MaxMemory {
		return fmt.Errorf("%w: tenant %s memory %d > limit %d", ErrResourceExhausted, tenantID, memBytes, l.MaxMemory)
	}
	if l.MaxCPUNano > 0 && cpuNano > l.MaxCPUNano {
		return fmt.Errorf("%w: tenant %s cpu %d > limit %d", ErrResourceExhausted, tenantID, cpuNano, l.MaxCPUNano)
	}
	return nil
}

// ApplyCgroupForTenant writes cgroup v2 limits when CgroupGroup is set (Linux + unified v2).
func (e *Enforcer) ApplyCgroupForTenant(tenantID string, cpuMax, memMax, ioMax string) error {
	l := e.limits[tenantID]
	if l == nil || l.CgroupGroup == "" {
		return nil
	}
	return enforcer.ApplyCgroupV2Limits(l.CgroupGroup, enforcer.CgroupLimits{
		CPUMax:    cpuMax,
		MemoryMax: memMax,
		IOMax:     ioMax,
	})
}

// IsResourceExhausted reports whether err is a quota exhaustion.
func IsResourceExhausted(err error) bool {
	return err != nil && errors.Is(err, ErrResourceExhausted)
}
