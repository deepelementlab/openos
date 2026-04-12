package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestRegistry(t *testing.T) *InMemoryRegistry {
	t.Helper()
	return NewInMemoryRegistry(zap.NewNop())
}

func TestRegister_ValidInstance(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc-a", "10.0.0.1", 8080)

	err := r.Register(context.Background(), inst)
	require.NoError(t, err)

	got, err := r.Get(context.Background(), inst.ID)
	require.NoError(t, err)
	assert.Equal(t, "svc-a", got.ServiceName)
	assert.Equal(t, "10.0.0.1:8080", got.Address())
	assert.True(t, got.IsHealthy())
}

func TestRegister_MissingID(t *testing.T) {
	r := newTestRegistry(t)
	inst := &ServiceInstance{ServiceName: "svc", Host: "h", Port: 1}
	err := r.Register(context.Background(), inst)
	assert.EqualError(t, err, "instance ID is required")
}

func TestRegister_MissingServiceName(t *testing.T) {
	r := newTestRegistry(t)
	inst := &ServiceInstance{ID: "x", Host: "h", Port: 1}
	err := r.Register(context.Background(), inst)
	assert.EqualError(t, err, "service name is required")
}

func TestRegister_Duplicate_UpdatesInstance(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	inst.Port = 9090
	require.NoError(t, r.Register(context.Background(), inst))

	got, err := r.Get(context.Background(), inst.ID)
	require.NoError(t, err)
	assert.Equal(t, 9090, got.Port)
}

func TestDeregister(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	err := r.Deregister(context.Background(), inst.ID)
	require.NoError(t, err)

	_, err = r.Get(context.Background(), inst.ID)
	assert.Error(t, err)
}

func TestDeregister_NotFound(t *testing.T) {
	r := newTestRegistry(t)
	err := r.Deregister(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestDeregister_CleansUpEmptyService(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))
	require.NoError(t, r.Deregister(context.Background(), inst.ID))

	services, _ := r.ListServices(context.Background())
	assert.Empty(t, services)
}

func TestGet(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	got, err := r.Get(context.Background(), inst.ID)
	require.NoError(t, err)
	assert.Equal(t, inst.ID, got.ID)

	_, err = r.Get(context.Background(), "missing")
	assert.Error(t, err)
}

func TestGet_ReturnsCopy(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	copy, _ := r.Get(context.Background(), inst.ID)
	copy.Port = 9999

	original, _ := r.Get(context.Background(), inst.ID)
	assert.NotEqual(t, 9999, original.Port)
}

func TestQuery_ByServiceName(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc-a", "10.0.0.1", 80)
	b := NewServiceInstance("svc-b", "10.0.0.2", 80)
	require.NoError(t, r.Register(context.Background(), a))
	require.NoError(t, r.Register(context.Background(), b))

	results, err := r.Query(context.Background(), ServiceQuery{ServiceName: "svc-a"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "svc-a", results[0].ServiceName)
}

func TestQuery_HealthyOnly(t *testing.T) {
	r := newTestRegistry(t)
	h := NewServiceInstance("svc", "10.0.0.1", 80)
	u := NewServiceInstance("svc", "10.0.0.2", 81)
	u.HealthStatus = HealthStatusUnhealthy
	require.NoError(t, r.Register(context.Background(), h))
	require.NoError(t, r.Register(context.Background(), u))

	results, err := r.Query(context.Background(), ServiceQuery{ServiceName: "svc", HealthyOnly: true})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestQuery_ByTags(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc", "10.0.0.1", 80)
	a.Tags = []string{"v2", "prod"}
	b := NewServiceInstance("svc", "10.0.0.2", 81)
	b.Tags = []string{"v1"}
	require.NoError(t, r.Register(context.Background(), a))
	require.NoError(t, r.Register(context.Background(), b))

	results, err := r.Query(context.Background(), ServiceQuery{Tags: []string{"prod"}})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestQuery_ByMetadata(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc", "10.0.0.1", 80)
	a.Metadata["version"] = "2.0"
	require.NoError(t, r.Register(context.Background(), a))

	results, err := r.Query(context.Background(), ServiceQuery{Metadata: map[string]string{"version": "2.0"}})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestQuery_ByZone(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc", "10.0.0.1", 80)
	a.Zone = "us-east-1a"
	b := NewServiceInstance("svc", "10.0.0.2", 81)
	b.Zone = "us-west-1a"
	require.NoError(t, r.Register(context.Background(), a))
	require.NoError(t, r.Register(context.Background(), b))

	results, err := r.Query(context.Background(), ServiceQuery{Zone: "us-east-1a"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestQuery_ByRegion(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc", "10.0.0.1", 80)
	a.Region = "us-east"
	require.NoError(t, r.Register(context.Background(), a))

	results, err := r.Query(context.Background(), ServiceQuery{Region: "us-east"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestGetService(t *testing.T) {
	r := newTestRegistry(t)
	a := NewServiceInstance("svc", "10.0.0.1", 80)
	b := NewServiceInstance("svc", "10.0.0.2", 81)
	require.NoError(t, r.Register(context.Background(), a))
	require.NoError(t, r.Register(context.Background(), b))

	ss, err := r.GetService(context.Background(), "svc")
	require.NoError(t, err)
	assert.Len(t, ss.Instances, 2)

	_, err = r.GetService(context.Background(), "missing")
	assert.Error(t, err)
}

func TestHeartbeat(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	beforeHB := inst.LastHeartbeat
	time.Sleep(10 * time.Millisecond)
	err := r.Heartbeat(context.Background(), inst.ID)
	require.NoError(t, err)

	got, _ := r.Get(context.Background(), inst.ID)
	assert.True(t, got.LastHeartbeat.After(beforeHB))
}

func TestHeartbeat_NotFound(t *testing.T) {
	r := newTestRegistry(t)
	err := r.Heartbeat(context.Background(), "missing")
	assert.Error(t, err)
}

func TestUpdateHealth(t *testing.T) {
	r := newTestRegistry(t)
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	err := r.UpdateHealth(context.Background(), inst.ID, HealthStatusUnhealthy)
	require.NoError(t, err)

	got, _ := r.Get(context.Background(), inst.ID)
	assert.Equal(t, HealthStatusUnhealthy, got.HealthStatus)
}

func TestUpdateHealth_NotFound(t *testing.T) {
	r := newTestRegistry(t)
	err := r.UpdateHealth(context.Background(), "missing", HealthStatusHealthy)
	assert.Error(t, err)
}

func TestListServices(t *testing.T) {
	r := newTestRegistry(t)
	require.NoError(t, r.Register(context.Background(), NewServiceInstance("svc-a", "h", 80)))
	require.NoError(t, r.Register(context.Background(), NewServiceInstance("svc-b", "h", 81)))

	services, err := r.ListServices(context.Background())
	require.NoError(t, err)
	assert.Len(t, services, 2)
}

func TestWatch(t *testing.T) {
	r := newTestRegistry(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := r.Watch(ctx, "svc")
	require.NoError(t, err)

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	select {
	case ss := <-ch:
		assert.Equal(t, "svc", ss.ServiceName)
		assert.Len(t, ss.Instances, 1)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for watch notification")
	}
}

func TestCleanupExpired(t *testing.T) {
	r := newTestRegistry(t)
	r.heartbeatTimeout = 50 * time.Millisecond

	inst := &ServiceInstance{
		ID:            "old",
		ServiceName:   "svc",
		Host:          "10.0.0.1",
		Port:          80,
		HealthStatus:  HealthStatusHealthy,
		LastHeartbeat: time.Now().UTC().Add(-200 * time.Millisecond),
		RegisteredAt:  time.Now().UTC(),
		Metadata:      map[string]string{},
		Tags:          []string{},
	}
	require.NoError(t, r.Register(context.Background(), inst))

	removed := r.CleanupExpired()
	assert.Equal(t, 1, removed)

	_, err := r.Get(context.Background(), "old")
	assert.Error(t, err)
}

func TestCleanupExpired_KeepsAlive(t *testing.T) {
	r := newTestRegistry(t)
	r.heartbeatTimeout = 5 * time.Second

	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, r.Register(context.Background(), inst))

	removed := r.CleanupExpired()
	assert.Equal(t, 0, removed)
}
