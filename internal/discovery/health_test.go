package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewHealthChecker(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())

	assert.Equal(t, 10*time.Second, hc.interval)
	assert.Equal(t, 5*time.Second, hc.timeout)
	assert.Equal(t, 3, hc.threshold)
}

func TestHealthChecker_GetHealthStatus_NotFound(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())

	_, exists := hc.GetHealthStatus("nonexistent")
	assert.False(t, exists)
}

func TestHealthChecker_RemoveInstance(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())

	hc.checks["test-id"] = &healthCheck{instanceID: "test-id"}
	hc.RemoveInstance("test-id")

	_, exists := hc.GetHealthStatus("test-id")
	assert.False(t, exists)
}

func TestHealthChecker_StartStop(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())
	hc.interval = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	hc.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	hc.Stop()
}

func TestHealthChecker_TCPCheck_Default(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	hc.checkInstance(context.Background(), inst)

	check, exists := hc.GetHealthStatus(inst.ID)
	assert.True(t, exists)
	assert.True(t, check.lastCheck.After(time.Time{}))
}

func TestHealthChecker_TCPCheck_SuccessThreshold(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())
	hc.threshold = 1

	inst := &ServiceInstance{
		ID:            "test",
		ServiceName:   "svc",
		Host:          "10.0.0.1",
		Port:          80,
		HealthStatus:  HealthStatusUnknown,
		LastHeartbeat: time.Now().UTC(),
		RegisteredAt:  time.Now().UTC(),
		Metadata:      map[string]string{},
		Tags:          []string{},
	}
	require.NoError(t, r.Register(context.Background(), inst))

	hc.checkInstance(context.Background(), inst)

	got, _ := r.Get(context.Background(), inst.ID)
	assert.Equal(t, HealthStatusHealthy, got.HealthStatus)
}

func TestHealthChecker_RemovedInstanceTracking(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())

	hc.checks["i-1"] = &healthCheck{instanceID: "i-1", lastCheck: time.Now()}
	hc.RemoveInstance("i-1")

	_, exists := hc.GetHealthStatus("i-1")
	assert.False(t, exists)
}

func TestHealthChecker_ConsecutiveTracking(t *testing.T) {
	r := NewInMemoryRegistry(zap.NewNop())
	hc := NewHealthChecker(r, zap.NewNop())
	hc.threshold = 3

	inst := &ServiceInstance{
		ID:            "test",
		ServiceName:   "svc",
		Host:          "10.0.0.1",
		Port:          80,
		HealthStatus:  HealthStatusUnknown,
		LastHeartbeat: time.Now().UTC(),
		RegisteredAt:  time.Now().UTC(),
		Metadata:      map[string]string{},
		Tags:          []string{},
	}
	require.NoError(t, r.Register(context.Background(), inst))

	for i := 0; i < 3; i++ {
		hc.checkInstance(context.Background(), inst)
	}

	check, exists := hc.GetHealthStatus(inst.ID)
	assert.True(t, exists)
	assert.Equal(t, 3, check.consecutiveSuccesses)
	assert.Equal(t, 0, check.consecutiveFailures)
}
