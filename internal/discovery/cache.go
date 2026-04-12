package discovery

import (
	"context"
	"sync"
	"time"
)

// ServiceCache provides caching for service discovery results.
type ServiceCache struct {
	cache     map[string]*cacheEntry
	ttl       time.Duration
	mu        sync.RWMutex
}

// cacheEntry represents a cached service set.
type cacheEntry struct {
	serviceSet *ServiceSet
	cachedAt   time.Time
}

// NewServiceCache creates a new service cache.
func NewServiceCache(ttl time.Duration) *ServiceCache {
	return &ServiceCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a cached service set.
func (c *ServiceCache) Get(serviceName string) (*ServiceSet, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[serviceName]
	if !exists {
		return nil, false
	}

	if time.Since(entry.cachedAt) > c.ttl {
		return nil, false
	}

	return entry.serviceSet, true
}

// Set caches a service set.
func (c *ServiceCache) Set(serviceName string, serviceSet *ServiceSet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[serviceName] = &cacheEntry{
		serviceSet: serviceSet,
		cachedAt:   time.Now().UTC(),
	}
}

// Invalidate removes a cached entry.
func (c *ServiceCache) Invalidate(serviceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, serviceName)
}

// InvalidateAll clears all cached entries.
func (c *ServiceCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

// Size returns the number of cached entries.
func (c *ServiceCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// CleanupExpired removes expired cache entries.
func (c *ServiceCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	now := time.Now().UTC()

	for name, entry := range c.cache {
		if now.Sub(entry.cachedAt) > c.ttl {
			delete(c.cache, name)
			removed++
		}
	}

	return removed
}

// StartCleanup starts a background cleanup task.
func (c *ServiceCache) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				c.CleanupExpired()
			}
		}
	}()
}

// CachingRegistry wraps a registry with caching.
type CachingRegistry struct {
	registry ServiceRegistry
	cache    *ServiceCache
}

// NewCachingRegistry creates a new caching registry.
func NewCachingRegistry(registry ServiceRegistry, ttl time.Duration) *CachingRegistry {
	cr := &CachingRegistry{
		registry: registry,
		cache:    NewServiceCache(ttl),
	}

	// Start cache cleanup
	ctx, _ := context.WithCancel(context.Background())
	cr.cache.StartCleanup(ctx, ttl)

	return cr
}

// Register implements ServiceRegistry.
func (cr *CachingRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	err := cr.registry.Register(ctx, instance)
	if err != nil {
		return err
	}

	// Invalidate cache for this service
	cr.cache.Invalidate(instance.ServiceName)

	return nil
}

// Deregister implements ServiceRegistry.
func (cr *CachingRegistry) Deregister(ctx context.Context, instanceID string) error {
	// Need to get service name before deregistering
	instance, err := cr.registry.Get(ctx, instanceID)
	if err != nil {
		return err
	}

	err = cr.registry.Deregister(ctx, instanceID)
	if err != nil {
		return err
	}

	// Invalidate cache for this service
	cr.cache.Invalidate(instance.ServiceName)

	return nil
}

// Get implements ServiceRegistry.
func (cr *CachingRegistry) Get(ctx context.Context, instanceID string) (*ServiceInstance, error) {
	return cr.registry.Get(ctx, instanceID)
}

// Query implements ServiceRegistry.
func (cr *CachingRegistry) Query(ctx context.Context, query ServiceQuery) ([]*ServiceInstance, error) {
	return cr.registry.Query(ctx, query)
}

// GetService implements ServiceRegistry with caching.
func (cr *CachingRegistry) GetService(ctx context.Context, serviceName string) (*ServiceSet, error) {
	// Try cache first
	if cached, ok := cr.cache.Get(serviceName); ok {
		return cached, nil
	}

	// Fetch from underlying registry
	serviceSet, err := cr.registry.GetService(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	cr.cache.Set(serviceName, serviceSet)

	return serviceSet, nil
}

// Heartbeat implements ServiceRegistry.
func (cr *CachingRegistry) Heartbeat(ctx context.Context, instanceID string) error {
	return cr.registry.Heartbeat(ctx, instanceID)
}

// UpdateHealth implements ServiceRegistry with cache invalidation.
func (cr *CachingRegistry) UpdateHealth(ctx context.Context, instanceID string, status HealthStatus) error {
	instance, err := cr.registry.Get(ctx, instanceID)
	if err != nil {
		return err
	}

	err = cr.registry.UpdateHealth(ctx, instanceID, status)
	if err != nil {
		return err
	}

	// Invalidate cache
	cr.cache.Invalidate(instance.ServiceName)

	return nil
}

// ListServices implements ServiceRegistry.
func (cr *CachingRegistry) ListServices(ctx context.Context) ([]string, error) {
	return cr.registry.ListServices(ctx)
}

// Watch implements ServiceRegistry.
func (cr *CachingRegistry) Watch(ctx context.Context, serviceName string) (<-chan *ServiceSet, error) {
	return cr.registry.Watch(ctx, serviceName)
}

// CacheStats returns cache statistics.
func (cr *CachingRegistry) CacheStats() map[string]interface{} {
	return map[string]interface{}{
		"size": cr.cache.Size(),
		"ttl":  cr.cache.ttl.Seconds(),
	}
}
