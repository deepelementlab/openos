package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobin_Select_Sequential(t *testing.T) {
	b := &RoundRobinBalancer{}
	b.Init()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
			{ID: "c", HealthStatus: HealthStatusHealthy},
		},
	}

	first, err := b.Select(context.Background(), ss)
	require.NoError(t, err)

	second, err := b.Select(context.Background(), ss)
	require.NoError(t, err)

	third, err := b.Select(context.Background(), ss)
	require.NoError(t, err)

	assert.Equal(t, "a", first.ID)
	assert.Equal(t, "b", second.ID)
	assert.Equal(t, "c", third.ID)
}

func TestRoundRobin_Select_Wraps(t *testing.T) {
	b := &RoundRobinBalancer{}
	b.Init()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
		},
	}

	b.Select(context.Background(), ss)
	b.Select(context.Background(), ss)

	third, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", third.ID)
}

func TestRoundRobin_Select_Empty(t *testing.T) {
	b := &RoundRobinBalancer{}
	b.Init()

	ss := &ServiceSet{Instances: []*ServiceInstance{}}
	_, err := b.Select(context.Background(), ss)
	assert.Equal(t, ErrEmptyServiceSet, err)
}

func TestRoundRobin_Select_NoHealthy_Fallback(t *testing.T) {
	b := &RoundRobinBalancer{}
	b.Init()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusUnhealthy},
		},
	}
	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", selected.ID)
}

func TestRoundRobin_Name(t *testing.T) {
	b := &RoundRobinBalancer{}
	assert.Equal(t, "round_robin", b.Name())
}

func TestRoundRobin_Reset(t *testing.T) {
	b := &RoundRobinBalancer{}
	b.Init()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
		},
	}
	b.Select(context.Background(), ss)
	b.Reset()

	inst, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", inst.ID)
}
