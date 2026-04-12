package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QuotaManager defines the interface for quota management
type QuotaManager interface {
	// CheckAgentQuota checks if tenant can create more agents
	CheckAgentQuota(ctx context.Context, tenantID string) error
	// IncrementAgentUsage increments the agent usage counter
	IncrementAgentUsage(ctx context.Context, tenantID string) error
	// DecrementAgentUsage decrements the agent usage counter
	DecrementAgentUsage(ctx context.Context, tenantID string) error
	// GetUsage returns current resource usage for a tenant
	GetUsage(ctx context.Context, tenantID string) (*ResourceUsage, error)
	// CheckQuota checks all quotas for a tenant
	CheckQuota(ctx context.Context, tenantID string, requirements *ResourceRequirements) error
}

// ResourceRequirements defines resource requirements for an operation
type ResourceRequirements struct {
	Agents   int
	CPUCores int
	MemoryGB int
	StorageGB int
	GPU      int
}

// DefaultQuotaManager is the default implementation of QuotaManager
type DefaultQuotaManager struct {
	logger *zap.Logger
	repo   TenantRepository
	cache  QuotaCache

	// Current usage tracking
	mu             sync.RWMutex
	usageByTenant  map[string]*ResourceUsage
}

// QuotaCache defines the interface for quota caching
type QuotaCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
}

// NewDefaultQuotaManager creates a new default quota manager
func NewDefaultQuotaManager(logger *zap.Logger, repo TenantRepository, cache QuotaCache) *DefaultQuotaManager {
	return &DefaultQuotaManager{
		logger:        logger,
		repo:          repo,
		cache:         cache,
		usageByTenant: make(map[string]*ResourceUsage),
	}
}

// CheckAgentQuota checks if tenant can create more agents
func (qm *DefaultQuotaManager) CheckAgentQuota(ctx context.Context, tenantID string) error {
	tenant, err := qm.repo.Get(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get current usage
	usage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	// Check agent quota
	if tenant.Quota.MaxAgents > 0 && usage.AgentsCount >= tenant.Quota.MaxAgents {
		return fmt.Errorf("agent quota exceeded: %d/%d", usage.AgentsCount, tenant.Quota.MaxAgents)
	}

	return nil
}

// IncrementAgentUsage increments the agent usage counter
func (qm *DefaultQuotaManager) IncrementAgentUsage(ctx context.Context, tenantID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	usage, ok := qm.usageByTenant[tenantID]
	if !ok {
		usage = &ResourceUsage{}
		qm.usageByTenant[tenantID] = usage
	}

	usage.AgentsCount++

	// Also update cache if available
	if qm.cache != nil {
		key := fmt.Sprintf("quota:%s:agents", tenantID)
		if _, err := qm.cache.Incr(ctx, key); err != nil {
			qm.logger.Warn("failed to increment cache counter", zap.Error(err))
		}
	}

	return nil
}

// DecrementAgentUsage decrements the agent usage counter
func (qm *DefaultQuotaManager) DecrementAgentUsage(ctx context.Context, tenantID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	usage, ok := qm.usageByTenant[tenantID]
	if !ok {
		return nil
	}

	if usage.AgentsCount > 0 {
		usage.AgentsCount--
	}

	// Also update cache if available
	if qm.cache != nil {
		key := fmt.Sprintf("quota:%s:agents", tenantID)
		if _, err := qm.cache.Decr(ctx, key); err != nil {
			qm.logger.Warn("failed to decrement cache counter", zap.Error(err))
		}
	}

	return nil
}

// GetUsage returns current resource usage for a tenant
func (qm *DefaultQuotaManager) GetUsage(ctx context.Context, tenantID string) (*ResourceUsage, error) {
	qm.mu.RLock()
	usage, ok := qm.usageByTenant[tenantID]
	qm.mu.RUnlock()

	if ok {
		// Return a copy
		return &ResourceUsage{
			AgentsCount:  usage.AgentsCount,
			CPUCoresUsed: usage.CPUCoresUsed,
			MemoryGBUsed: usage.MemoryGBUsed,
			StorageGBUsed: usage.StorageGBUsed,
			GPUUsed:      usage.GPUUsed,
		}, nil
	}

	// Try to get from cache
	if qm.cache != nil {
		key := fmt.Sprintf("quota:%s:agents", tenantID)
		val, err := qm.cache.Get(ctx, key)
		if err == nil && val != "" {
			var count int
			fmt.Sscanf(val, "%d", &count)
			return &ResourceUsage{
				AgentsCount: count,
			}, nil
		}
	}

	// Initialize empty usage
	return &ResourceUsage{}, nil
}

// CheckQuota checks all quotas for a tenant
func (qm *DefaultQuotaManager) CheckQuota(ctx context.Context, tenantID string, requirements *ResourceRequirements) error {
	tenant, err := qm.repo.Get(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	currentUsage, err := qm.GetUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get current usage: %w", err)
	}

	quota := tenant.Quota

	// Check each resource
	if quota.MaxAgents > 0 && currentUsage.AgentsCount+requirements.Agents > quota.MaxAgents {
		return fmt.Errorf("agent quota exceeded: current %d, requested %d, limit %d",
			currentUsage.AgentsCount, requirements.Agents, quota.MaxAgents)
	}

	if quota.MaxCPU > 0 && currentUsage.CPUCoresUsed+requirements.CPUCores > quota.MaxCPU {
		return fmt.Errorf("CPU quota exceeded: current %d, requested %d, limit %d",
			currentUsage.CPUCoresUsed, requirements.CPUCores, quota.MaxCPU)
	}

	if quota.MaxMemoryGB > 0 && currentUsage.MemoryGBUsed+requirements.MemoryGB > quota.MaxMemoryGB {
		return fmt.Errorf("memory quota exceeded: current %d, requested %d, limit %d",
			currentUsage.MemoryGBUsed, requirements.MemoryGB, quota.MaxMemoryGB)
	}

	return nil
}

// InMemoryQuotaCache is a simple in-memory implementation of QuotaCache
type InMemoryQuotaCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewInMemoryQuotaCache creates a new in-memory quota cache
func NewInMemoryQuotaCache() *InMemoryQuotaCache {
	return &InMemoryQuotaCache{
		data: make(map[string]cacheEntry),
	}
}

// Get retrieves a value from the cache
func (c *InMemoryQuotaCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("key not found")
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return "", fmt.Errorf("key expired")
	}

	return entry.value, nil
}

// Set stores a value in the cache
func (c *InMemoryQuotaCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Incr increments a counter in the cache
func (c *InMemoryQuotaCache) Incr(ctx context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	var current int64
	if ok && time.Now().Before(entry.expiresAt) {
		fmt.Sscanf(entry.value, "%d", &current)
	}

	current++
	c.data[key] = cacheEntry{
		value:     fmt.Sprintf("%d", current),
		expiresAt: time.Now().Add(time.Hour),
	}

	return current, nil
}

// Decr decrements a counter in the cache
func (c *InMemoryQuotaCache) Decr(ctx context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	var current int64
	if ok && time.Now().Before(entry.expiresAt) {
		fmt.Sscanf(entry.value, "%d", &current)
	}

	if current > 0 {
		current--
	}
	c.data[key] = cacheEntry{
		value:     fmt.Sprintf("%d", current),
		expiresAt: time.Now().Add(time.Hour),
	}

	return current, nil
}
