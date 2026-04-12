package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoadBalancerRegistry_GetAll(t *testing.T) {
	reg := NewLoadBalancerRegistry()

	balancers := reg.List()
	assert.Contains(t, balancers, "round_robin")
	assert.Contains(t, balancers, "random")
	assert.Contains(t, balancers, "weighted")
	assert.Contains(t, balancers, "least_connection")
}

func TestLoadBalancerRegistry_Get(t *testing.T) {
	reg := NewLoadBalancerRegistry()

	lb, err := reg.Get("round_robin")
	require.NoError(t, err)
	assert.Equal(t, "round_robin", lb.Name())

	_, err = reg.Get("nonexistent")
	assert.Error(t, err)
}

func TestLoadBalancerRegistry_GetDefault(t *testing.T) {
	reg := NewLoadBalancerRegistry()
	lb := reg.GetDefault()
	assert.Equal(t, "round_robin", lb.Name())
}

func TestLoadBalancerRegistry_SetDefault(t *testing.T) {
	reg := NewLoadBalancerRegistry()
	err := reg.SetDefault("weighted")
	require.NoError(t, err)
	assert.Equal(t, "weighted", reg.GetDefault().Name())
}

func TestLoadBalancerRegistry_SetDefault_NotFound(t *testing.T) {
	reg := NewLoadBalancerRegistry()
	err := reg.SetDefault("nonexistent")
	assert.Error(t, err)
}

func TestLoadBalancerRegistry_Register(t *testing.T) {
	reg := NewLoadBalancerRegistry()
	reg.Register("custom", &RoundRobinBalancer{})
	lb, err := reg.Get("custom")
	require.NoError(t, err)
	assert.NotNil(t, lb)
}

func TestBaseBalancer_GetCandidates_HealthyOnly(t *testing.T) {
	b := &BaseBalancer{healthyOnly: true}
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusHealthy},
			{ID: "2", HealthStatus: HealthStatusUnhealthy},
		},
	}
	candidates := b.GetCandidates(ss)
	assert.Len(t, candidates, 1)
}

func TestBaseBalancer_GetCandidates_HealthyOnlyFallback(t *testing.T) {
	b := &BaseBalancer{healthyOnly: true}
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusUnhealthy},
		},
	}
	candidates := b.GetCandidates(ss)
	assert.Len(t, candidates, 1, "should fall back to all instances when no healthy ones")
}

func TestBaseBalancer_GetCandidates_All(t *testing.T) {
	b := &BaseBalancer{healthyOnly: false}
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "1", HealthStatus: HealthStatusHealthy},
			{ID: "2", HealthStatus: HealthStatusUnhealthy},
		},
	}
	candidates := b.GetCandidates(ss)
	assert.Len(t, candidates, 2)
}

func TestClientSideLoadBalancer_Select(t *testing.T) {
	registry := NewInMemoryRegistry(newNopLogger())
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, registry.Register(context.Background(), inst))

	cslb := NewClientSideLoadBalancer(registry)

	selected, err := cslb.Select(context.Background(), "svc", "round_robin")
	require.NoError(t, err)
	assert.Equal(t, inst.ID, selected.ID)
}

func TestClientSideLoadBalancer_Select_DefaultLB(t *testing.T) {
	registry := NewInMemoryRegistry(newNopLogger())
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, registry.Register(context.Background(), inst))

	cslb := NewClientSideLoadBalancer(registry)
	selected, err := cslb.Select(context.Background(), "svc", "nonexistent_lb")
	require.NoError(t, err)
	assert.NotNil(t, selected)
}

func TestClientSideLoadBalancer_SelectWithQuery(t *testing.T) {
	registry := NewInMemoryRegistry(newNopLogger())
	inst := NewServiceInstance("svc", "10.0.0.1", 80)
	require.NoError(t, registry.Register(context.Background(), inst))

	cslb := NewClientSideLoadBalancer(registry)
	selected, err := cslb.SelectWithQuery(context.Background(), ServiceQuery{
		ServiceName: "svc",
		HealthyOnly: true,
	}, "round_robin")
	require.NoError(t, err)
	assert.NotNil(t, selected)
}

func TestClientSideLoadBalancer_SelectWithQuery_NoInstances(t *testing.T) {
	registry := NewInMemoryRegistry(newNopLogger())
	cslb := NewClientSideLoadBalancer(registry)

	_, err := cslb.SelectWithQuery(context.Background(), ServiceQuery{
		ServiceName: "missing",
	}, "round_robin")
	assert.Error(t, err)
}

func newNopLogger() *zap.Logger {
	return zap.NewNop()
}
