package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeighted_Select_Single(t *testing.T) {
	b := NewWeightedBalancer()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", Weight: 100, HealthStatus: HealthStatusHealthy},
		},
	}

	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", selected.ID)
}

func TestWeighted_Select_Empty(t *testing.T) {
	b := NewWeightedBalancer()
	ss := &ServiceSet{Instances: []*ServiceInstance{}}
	_, err := b.Select(context.Background(), ss)
	assert.Equal(t, ErrEmptyServiceSet, err)
}

func TestWeighted_Select_NoHealthy_Fallback(t *testing.T) {
	b := NewWeightedBalancer()
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", Weight: 100, HealthStatus: HealthStatusUnhealthy},
		},
	}
	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", selected.ID)
}

func TestWeighted_Name(t *testing.T) {
	b := NewWeightedBalancer()
	assert.Equal(t, "weighted", b.Name())
}

func TestWeighted_Select_AllSameWeight(t *testing.T) {
	b := NewWeightedBalancer()
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", Weight: 100, HealthStatus: HealthStatusHealthy},
			{ID: "b", Weight: 100, HealthStatus: HealthStatusHealthy},
			{ID: "c", Weight: 100, HealthStatus: HealthStatusHealthy},
		},
	}

	ids := map[string]int{}
	for i := 0; i < 300; i++ {
		selected, err := b.Select(context.Background(), ss)
		require.NoError(t, err)
		ids[selected.ID]++
	}

	for _, id := range []string{"a", "b", "c"} {
		assert.True(t, ids[id] > 0, "expected %s to be selected at least once", id)
	}
}

func TestWeighted_Select_HigherWeightSelected(t *testing.T) {
	b := NewWeightedBalancer()
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "heavy", Weight: 900, HealthStatus: HealthStatusHealthy},
			{ID: "light", Weight: 100, HealthStatus: HealthStatusHealthy},
		},
	}

	heavyCount := 0
	for i := 0; i < 1000; i++ {
		selected, err := b.Select(context.Background(), ss)
		require.NoError(t, err)
		if selected.ID == "heavy" {
			heavyCount++
		}
	}

	assert.True(t, heavyCount > 500, "heavy instance should be selected more often, got %d/1000", heavyCount)
}

func TestWeighted_Select_ZeroWeight(t *testing.T) {
	b := NewWeightedBalancer()
	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", Weight: 0, HealthStatus: HealthStatusHealthy},
			{ID: "b", Weight: 0, HealthStatus: HealthStatusHealthy},
		},
	}

	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Contains(t, []string{"a", "b"}, selected.ID)
}
