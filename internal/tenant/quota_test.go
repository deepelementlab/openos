package tenant

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultQuotaManager_CheckAgentQuota(t *testing.T) {
	logger := zap.NewNop()
	repo := NewInMemoryTenantRepository()
	cache := NewInMemoryQuotaCache()
	qm := NewDefaultQuotaManager(logger, repo, cache)

	ctx := context.Background()

	// Create a tenant with quota
	tenant := &Tenant{
		ID:        "tenant-1",
		Name:      "Test Tenant",
		Status:    TenantActive,
		Plan:      "basic",
		Quota:     ResourceQuota{MaxAgents: 5},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.Create(ctx, tenant)

	// Test: should allow creation when under quota
	err := qm.CheckAgentQuota(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("expected no error when under quota, got: %v", err)
	}

	// Simulate usage at limit
	qm.usageByTenant["tenant-1"] = &ResourceUsage{AgentsCount: 5}

	// Test: should deny creation when at quota limit
	err = qm.CheckAgentQuota(ctx, "tenant-1")
	if err == nil {
		t.Fatal("expected error when at quota limit")
	}

	// Test: non-existent tenant
	err = qm.CheckAgentQuota(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
}

func TestDefaultQuotaManager_GetUsage(t *testing.T) {
	logger := zap.NewNop()
	repo := NewInMemoryTenantRepository()
	cache := NewInMemoryQuotaCache()
	qm := NewDefaultQuotaManager(logger, repo, cache)

	ctx := context.Background()

	// Test: empty usage for new tenant
	usage, err := qm.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.AgentsCount != 0 {
		t.Fatalf("expected 0 agents, got %d", usage.AgentsCount)
	}

	// Increment usage
	qm.IncrementAgentUsage(ctx, "tenant-1")

	// Test: should reflect incremented usage
	usage, err = qm.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.AgentsCount != 1 {
		t.Fatalf("expected 1 agent, got %d", usage.AgentsCount)
	}
}

func TestDefaultQuotaManager_IncrementDecrement(t *testing.T) {
	logger := zap.NewNop()
	repo := NewInMemoryTenantRepository()
	cache := NewInMemoryQuotaCache()
	qm := NewDefaultQuotaManager(logger, repo, cache)

	ctx := context.Background()

	// Increment
	err := qm.IncrementAgentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage := qm.usageByTenant["tenant-1"]
	if usage.AgentsCount != 1 {
		t.Fatalf("expected 1 agent, got %d", usage.AgentsCount)
	}

	// Increment again
	err = qm.IncrementAgentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage = qm.usageByTenant["tenant-1"]
	if usage.AgentsCount != 2 {
		t.Fatalf("expected 2 agents, got %d", usage.AgentsCount)
	}

	// Decrement
	err = qm.DecrementAgentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage = qm.usageByTenant["tenant-1"]
	if usage.AgentsCount != 1 {
		t.Fatalf("expected 1 agent after decrement, got %d", usage.AgentsCount)
	}

	// Decrement to zero
	err = qm.DecrementAgentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage = qm.usageByTenant["tenant-1"]
	if usage.AgentsCount != 0 {
		t.Fatalf("expected 0 agents after decrement, got %d", usage.AgentsCount)
	}

	// Decrement below zero (should stay at 0)
	err = qm.DecrementAgentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage = qm.usageByTenant["tenant-1"]
	if usage.AgentsCount != 0 {
		t.Fatalf("expected 0 agents after underflow, got %d", usage.AgentsCount)
	}
}

func TestDefaultQuotaManager_CheckQuota(t *testing.T) {
	logger := zap.NewNop()
	repo := NewInMemoryTenantRepository()
	cache := NewInMemoryQuotaCache()
	qm := NewDefaultQuotaManager(logger, repo, cache)

	ctx := context.Background()

	// Create a tenant with quota
	tenant := &Tenant{
		ID:     "tenant-1",
		Name:   "Test Tenant",
		Status: TenantActive,
		Plan:   "basic",
		Quota: ResourceQuota{
			MaxAgents:   5,
			MaxCPU:      10,
			MaxMemoryGB: 20,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.Create(ctx, tenant)

	// Set current usage
	qm.usageByTenant["tenant-1"] = &ResourceUsage{
		AgentsCount:  3,
		CPUCoresUsed: 5,
		MemoryGBUsed: 10,
	}

	// Test: request within limits
	req := &ResourceRequirements{
		Agents:   1,
		CPUCores: 3,
		MemoryGB: 8,
	}
	err := qm.CheckQuota(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("expected no error within limits, got: %v", err)
	}

	// Test: request exceeding agent quota
	req = &ResourceRequirements{
		Agents: 3, // 3 + 3 > 5
	}
	err = qm.CheckQuota(ctx, "tenant-1", req)
	if err == nil {
		t.Fatal("expected error for exceeding agent quota")
	}

	// Test: request exceeding CPU quota
	req = &ResourceRequirements{
		CPUCores: 10, // 5 + 10 > 10
	}
	err = qm.CheckQuota(ctx, "tenant-1", req)
	if err == nil {
		t.Fatal("expected error for exceeding CPU quota")
	}

	// Test: non-existent tenant
	err = qm.CheckQuota(ctx, "non-existent", req)
	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
}

func TestInMemoryQuotaCache(t *testing.T) {
	cache := NewInMemoryQuotaCache()
	ctx := context.Background()

	// Test Set and Get
	err := cache.Set(ctx, "key1", "value1", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value1" {
		t.Fatalf("expected 'value1', got %s", val)
	}

	// Test Get non-existent key
	_, err = cache.Get(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}

	// Test Incr
	count, err := cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}

	count, err = cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}

	// Test Decr
	count, err = cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}

	// Test Decr to zero
	count, err = cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	// Test Decr below zero
	count, err = cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 (no underflow), got %d", count)
	}

	// Test expired entry
	err = cache.Set(ctx, "expire-test", "value", time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(time.Millisecond * 10)

	_, err = cache.Get(ctx, "expire-test")
	if err == nil {
		t.Fatal("expected error for expired key")
	}
}
