package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultResourceTracker(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{
		MaxCPU:    8.0,
		MaxMemMB:  16384,
		MaxDiskMB: 100000,
	})
	assert.NotNil(t, tracker)
}

func TestResourceTracker_Allocate(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{
		MaxCPU:    8.0,
		MaxMemMB:  16384,
		MaxDiskMB: 100000,
	})

	err := tracker.Allocate(context.Background(), ResourceAllocation{
		AgentID:   "a1",
		CPUShares: 2.0,
		MemoryMB:  4096,
		DiskMB:    10000,
	})
	require.NoError(t, err)
}

func TestResourceTracker_Allocate_MissingAgentID(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	err := tracker.Allocate(context.Background(), ResourceAllocation{})
	assert.EqualError(t, err, "agent ID is required")
}

func TestResourceTracker_Allocate_Duplicate(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	alloc := ResourceAllocation{AgentID: "a1", CPUShares: 1}
	require.NoError(t, tracker.Allocate(context.Background(), alloc))
	err := tracker.Allocate(context.Background(), alloc)
	assert.EqualError(t, err, "allocation for agent a1 already exists")
}

func TestResourceTracker_Release(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 1}))
	require.NoError(t, tracker.Release(context.Background(), "a1"))
}

func TestResourceTracker_Release_NotFound(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	err := tracker.Release(context.Background(), "nonexistent")
	assert.EqualError(t, err, "no allocation for agent nonexistent")
}

func TestResourceTracker_Get(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{
		AgentID:   "a1",
		CPUShares: 2.5,
		MemoryMB:  1024,
	}))

	alloc, err := tracker.Get(context.Background(), "a1")
	require.NoError(t, err)
	assert.Equal(t, 2.5, alloc.CPUShares)
	assert.Equal(t, int64(1024), alloc.MemoryMB)
}

func TestResourceTracker_Get_NotFound(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	_, err := tracker.Get(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestResourceTracker_ListAllocations(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 1}))
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a2", CPUShares: 2}))

	allocs, err := tracker.ListAllocations(context.Background())
	require.NoError(t, err)
	assert.Len(t, allocs, 2)
}

func TestResourceTracker_TotalUsed(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8, MaxMemMB: 4096, MaxDiskMB: 1000})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 2, MemoryMB: 1024, DiskMB: 200}))
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a2", CPUShares: 3, MemoryMB: 2048, DiskMB: 300}))

	cpu, mem, disk := tracker.TotalUsed(context.Background())
	assert.Equal(t, 5.0, cpu)
	assert.Equal(t, int64(3072), mem)
	assert.Equal(t, int64(500), disk)
}

func TestResourceTracker_CheckQuota_OK(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 8, MaxMemMB: 4096, MaxDiskMB: 1000})
	err := tracker.CheckQuota(context.Background(), ResourceAllocation{AgentID: "new", CPUShares: 4, MemoryMB: 1024, DiskMB: 500})
	assert.NoError(t, err)
}

func TestResourceTracker_CheckQuota_CPUExceeded(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 4, MaxMemMB: 0, MaxDiskMB: 0})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 3}))
	err := tracker.CheckQuota(context.Background(), ResourceAllocation{AgentID: "new", CPUShares: 2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CPU quota exceeded")
}

func TestResourceTracker_CheckQuota_MemoryExceeded(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 0, MaxMemMB: 1024, MaxDiskMB: 0})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", MemoryMB: 512}))
	err := tracker.CheckQuota(context.Background(), ResourceAllocation{AgentID: "new", MemoryMB: 1024})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory quota exceeded")
}

func TestResourceTracker_CheckQuota_DiskExceeded(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 0, MaxMemMB: 0, MaxDiskMB: 500})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", DiskMB: 300}))
	err := tracker.CheckQuota(context.Background(), ResourceAllocation{AgentID: "new", DiskMB: 300})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk quota exceeded")
}

func TestResourceTracker_Allocate_EnforcesQuota(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{MaxCPU: 2, MaxMemMB: 0, MaxDiskMB: 0})
	require.NoError(t, tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 1.5}))
	err := tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a2", CPUShares: 1.0})
	assert.Error(t, err)
}

func TestResourceTracker_NoQuota(t *testing.T) {
	tracker := NewDefaultResourceTracker(QuotaConfig{})
	err := tracker.Allocate(context.Background(), ResourceAllocation{AgentID: "a1", CPUShares: 9999, MemoryMB: 99999, DiskMB: 99999})
	assert.NoError(t, err, "zero quotas should not enforce limits")
}
