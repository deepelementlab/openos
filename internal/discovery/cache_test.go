package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceCache_SetAndGet(t *testing.T) {
	c := NewServiceCache(5 * time.Minute)

	ss := &ServiceSet{ServiceName: "svc", Instances: []*ServiceInstance{{ID: "1"}}}
	c.Set("svc", ss)

	got, ok := c.Get("svc")
	assert.True(t, ok)
	assert.Equal(t, "svc", got.ServiceName)
}

func TestServiceCache_Get_Miss(t *testing.T) {
	c := NewServiceCache(5 * time.Minute)
	_, ok := c.Get("nonexistent")
	assert.False(t, ok)
}

func TestServiceCache_Get_Expired(t *testing.T) {
	c := NewServiceCache(10 * time.Millisecond)
	ss := &ServiceSet{ServiceName: "svc", Instances: []*ServiceInstance{{ID: "1"}}}
	c.Set("svc", ss)

	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("svc")
	assert.False(t, ok)
}

func TestServiceCache_Invalidate(t *testing.T) {
	c := NewServiceCache(5 * time.Minute)
	c.Set("svc", &ServiceSet{ServiceName: "svc"})
	c.Invalidate("svc")

	_, ok := c.Get("svc")
	assert.False(t, ok)
}

func TestServiceCache_InvalidateAll(t *testing.T) {
	c := NewServiceCache(5 * time.Minute)
	c.Set("a", &ServiceSet{ServiceName: "a"})
	c.Set("b", &ServiceSet{ServiceName: "b"})
	assert.Equal(t, 2, c.Size())

	c.InvalidateAll()
	assert.Equal(t, 0, c.Size())
}

func TestServiceCache_Size(t *testing.T) {
	c := NewServiceCache(5 * time.Minute)
	assert.Equal(t, 0, c.Size())

	c.Set("a", &ServiceSet{ServiceName: "a"})
	assert.Equal(t, 1, c.Size())
}

func TestServiceCache_CleanupExpired(t *testing.T) {
	c := NewServiceCache(10 * time.Millisecond)

	c.Set("old1", &ServiceSet{ServiceName: "old1"})
	c.Set("old2", &ServiceSet{ServiceName: "old2"})

	time.Sleep(20 * time.Millisecond)

	c.Set("fresh", &ServiceSet{ServiceName: "fresh"})

	removed := c.CleanupExpired()
	assert.Equal(t, 2, removed)
	assert.Equal(t, 1, c.Size())
}

func TestServiceCache_CleanupContext(t *testing.T) {
	c := NewServiceCache(10 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.StartCleanup(ctx, 50*time.Millisecond)
	c.Set("x", &ServiceSet{ServiceName: "x"})
	time.Sleep(30 * time.Millisecond)
	cancel()
}

func TestCachingRegistry_RegisterAndInvalidate(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	_, ok := cr.cache.Get("svc")
	assert.False(t, ok, "cache should be invalidated after register")
}

func TestCachingRegistry_GetService_CachesResult(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, inner.Register(context.Background(), inst))

	ss1, err := cr.GetService(context.Background(), "svc")
	require.NoError(t, err)

	_, ok := cr.cache.Get("svc")
	assert.True(t, ok, "result should be cached")

	ss2, err := cr.GetService(context.Background(), "svc")
	require.NoError(t, err)
	assert.Equal(t, ss1.ServiceName, ss2.ServiceName)
}

func TestCachingRegistry_Deregister(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	err := cr.Deregister(context.Background(), inst.ID)
	require.NoError(t, err)

	_, ok := cr.cache.Get("svc")
	assert.False(t, ok, "cache should be invalidated after deregister")
}

func TestCachingRegistry_Get(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	got, err := cr.Get(context.Background(), inst.ID)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, got.ID)
}

func TestCachingRegistry_Query(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	results, err := cr.Query(context.Background(), ServiceQuery{ServiceName: "svc"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCachingRegistry_Heartbeat(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	err := cr.Heartbeat(context.Background(), inst.ID)
	require.NoError(t, err)
}

func TestCachingRegistry_UpdateHealth(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, cr.Register(context.Background(), inst))

	err := cr.UpdateHealth(context.Background(), inst.ID, HealthStatusUnhealthy)
	require.NoError(t, err)

	_, ok := cr.cache.Get("svc")
	assert.False(t, ok, "cache should be invalidated after health update")
}

func TestCachingRegistry_ListServices(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	require.NoError(t, cr.Register(context.Background(), NewServiceInstance("a", "h", 80)))
	require.NoError(t, cr.Register(context.Background(), NewServiceInstance("b", "h", 81)))

	services, err := cr.ListServices(context.Background())
	require.NoError(t, err)
	assert.Len(t, services, 2)
}

func TestCachingRegistry_Watch(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := cr.Watch(ctx, "svc")
	require.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestCachingRegistry_CacheStats(t *testing.T) {
	inner := NewInMemoryRegistry(zap.NewNop())
	cr := NewCachingRegistry(inner, 5*time.Minute)

	stats := cr.CacheStats()
	assert.Equal(t, 0, stats["size"])
}
