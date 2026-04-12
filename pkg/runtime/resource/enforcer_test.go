package resource

import (
	"testing"
	"time"

	"github.com/agentos/aos/pkg/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestTracker() ResourceTracker {
	return NewDefaultResourceTracker(QuotaConfig{MaxCPU: 100, MaxMemMB: 1024, MaxDiskMB: 500 * 1024})
}

func TestNewResourceEnforcer(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	assert.NotNil(t, re)
}

func TestResourceEnforcer_SetQuota(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())

	err := re.SetQuota(&ResourceQuota{
		AgentID:      "agent-1",
		MaxCPUShares: 1000,
		MaxMemoryMB:  512,
	})
	require.NoError(t, err)

	quota, err := re.GetQuota("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), quota.MaxCPUShares)
	assert.Equal(t, int64(512), quota.MaxMemoryMB)
}

func TestResourceEnforcer_SetQuota_EmptyAgentID(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	err := re.SetQuota(&ResourceQuota{AgentID: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

func TestResourceEnforcer_GetQuota_NotFound(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	_, err := re.GetQuota("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota not found")
}

func TestResourceEnforcer_RemoveQuota(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{AgentID: "agent-1", MaxMemoryMB: 512})
	re.RemoveQuota("agent-1")
	_, err := re.GetQuota("agent-1")
	assert.Error(t, err)
}

func TestResourceEnforcer_CheckAndEnforce_Allow(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{
		AgentID:      "agent-1",
		MaxCPUShares: 2000,
		MaxMemoryMB:  1024,
	})

	spec := &types.AgentSpec{
		Resources: &types.ResourceRequirements{
			CPULimit:    1000,
			MemoryLimit: 512 * 1024 * 1024,
		},
	}

	action, err := re.CheckAndEnforce(nil, "agent-1", spec)
	assert.NoError(t, err)
	assert.Equal(t, EnforcementAllow, action)
}

func TestResourceEnforcer_CheckAndEnforce_MemoryExceeded(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{
		AgentID:     "agent-1",
		MaxMemoryMB: 256,
	})

	spec := &types.AgentSpec{
		Resources: &types.ResourceRequirements{
			MemoryLimit: 512 * 1024 * 1024,
		},
	}

	action, err := re.CheckAndEnforce(nil, "agent-1", spec)
	assert.Error(t, err)
	assert.Equal(t, EnforcementKill, action)
}

func TestResourceEnforcer_CheckAndEnforce_CPUExceeded(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{
		AgentID:      "agent-1",
		MaxCPUShares: 500,
	})

	spec := &types.AgentSpec{
		Resources: &types.ResourceRequirements{
			CPULimit: 1000,
		},
	}

	action, err := re.CheckAndEnforce(nil, "agent-1", spec)
	assert.Error(t, err)
	assert.Equal(t, EnforcementThrottle, action)
}

func TestResourceEnforcer_CheckAndEnforce_NoQuota(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	action, err := re.CheckAndEnforce(nil, "agent-no-quota", &types.AgentSpec{})
	assert.NoError(t, err)
	assert.Equal(t, EnforcementAllow, action)
}

func TestResourceEnforcer_CheckAndEnforce_NilResources(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{AgentID: "agent-1", MaxMemoryMB: 512})
	action, err := re.CheckAndEnforce(nil, "agent-1", &types.AgentSpec{})
	assert.NoError(t, err)
	assert.Equal(t, EnforcementAllow, action)
}

func TestResourceEnforcer_Violations(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{AgentID: "agent-1", MaxMemoryMB: 256})

	spec := &types.AgentSpec{
		Resources: &types.ResourceRequirements{MemoryLimit: 512 * 1024 * 1024},
	}
	re.CheckAndEnforce(nil, "agent-1", spec)

	violations := re.GetViolations("agent-1")
	assert.Len(t, violations, 1)
	assert.Equal(t, "memory", violations[0].Resource)
	assert.Equal(t, EnforcementKill, violations[0].Action)
}

func TestResourceEnforcer_ClearViolations(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{AgentID: "agent-1", MaxMemoryMB: 256})

	spec := &types.AgentSpec{
		Resources: &types.ResourceRequirements{MemoryLimit: 512 * 1024 * 1024},
	}
	re.CheckAndEnforce(nil, "agent-1", spec)
	assert.Len(t, re.GetViolations("agent-1"), 1)

	re.ClearViolations("agent-1")
	assert.Empty(t, re.GetViolations("agent-1"))
}

func TestResourceEnforcer_ListQuotas(t *testing.T) {
	re := NewResourceEnforcer(makeTestTracker())
	re.SetQuota(&ResourceQuota{AgentID: "a1", MaxMemoryMB: 512})
	re.SetQuota(&ResourceQuota{AgentID: "a2", MaxMemoryMB: 1024})

	quotas := re.ListQuotas()
	assert.Len(t, quotas, 2)
}

func TestQuotaViolation_Fields(t *testing.T) {
	now := time.Now()
	v := QuotaViolation{
		AgentID:   "agent-1",
		Resource:  "cpu",
		Requested: 2000,
		Limit:     1000,
		Action:    EnforcementThrottle,
		Timestamp: now,
	}
	assert.Equal(t, "agent-1", v.AgentID)
	assert.Equal(t, "cpu", v.Resource)
	assert.Equal(t, int64(2000), v.Requested)
	assert.Equal(t, int64(1000), v.Limit)
	assert.Equal(t, EnforcementThrottle, v.Action)
}

func TestResourceQuota_AllFields(t *testing.T) {
	q := ResourceQuota{
		AgentID:        "agent-1",
		MaxCPUShares:   2000,
		MaxMemoryMB:    1024,
		MaxDiskMB:      10000,
		MaxPIDs:        100,
		MaxNetworkKbps: 10000,
	}
	assert.Equal(t, int64(2000), q.MaxCPUShares)
	assert.Equal(t, int64(1024), q.MaxMemoryMB)
	assert.Equal(t, int64(10000), q.MaxDiskMB)
	assert.Equal(t, int64(100), q.MaxPIDs)
	assert.Equal(t, int64(10000), q.MaxNetworkKbps)
}
