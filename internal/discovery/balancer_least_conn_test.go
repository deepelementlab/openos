package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeastConnection_Select_MinConnections(t *testing.T) {
	b := NewLeastConnectionBalancer()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
		},
	}

	b.IncrementConnections("a")
	b.IncrementConnections("a")
	b.IncrementConnections("a")

	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "b", selected.ID)
}

func TestLeastConnection_Select_Empty(t *testing.T) {
	b := NewLeastConnectionBalancer()

	ss := &ServiceSet{Instances: []*ServiceInstance{}}
	_, err := b.Select(context.Background(), ss)
	assert.Equal(t, ErrEmptyServiceSet, err)
}

func TestLeastConnection_Select_NoHealthy_Fallback(t *testing.T) {
	b := NewLeastConnectionBalancer()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusUnhealthy},
		},
	}
	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "a", selected.ID)
}

func TestLeastConnection_Name(t *testing.T) {
	b := NewLeastConnectionBalancer()
	assert.Equal(t, "least_connection", b.Name())
}

func TestLeastConnection_ConnectionTracking(t *testing.T) {
	b := NewLeastConnectionBalancer()

	b.IncrementConnections("a")
	b.IncrementConnections("a")
	b.IncrementConnections("b")

	assert.Equal(t, int64(2), b.GetConnectionCount("a"))
	assert.Equal(t, int64(1), b.GetConnectionCount("b"))
	assert.Equal(t, int64(0), b.GetConnectionCount("c"))
}

func TestLeastConnection_DecrementConnections(t *testing.T) {
	b := NewLeastConnectionBalancer()

	b.IncrementConnections("a")
	b.IncrementConnections("a")
	b.DecrementConnections("a")

	assert.Equal(t, int64(1), b.GetConnectionCount("a"))
}

func TestLeastConnection_Decrement_Zero(t *testing.T) {
	b := NewLeastConnectionBalancer()
	b.DecrementConnections("a")
	assert.Equal(t, int64(0), b.GetConnectionCount("a"))
}

func TestLeastConnection_Select_PicksLowest(t *testing.T) {
	b := NewLeastConnectionBalancer()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
			{ID: "c", HealthStatus: HealthStatusHealthy},
		},
	}

	b.IncrementConnections("a")
	b.IncrementConnections("c")
	b.IncrementConnections("c")
	b.IncrementConnections("c")

	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Equal(t, "b", selected.ID)
}

func TestRandomBalancer_Select(t *testing.T) {
	b := NewRandomBalancer()

	ss := &ServiceSet{
		Instances: []*ServiceInstance{
			{ID: "a", HealthStatus: HealthStatusHealthy},
			{ID: "b", HealthStatus: HealthStatusHealthy},
		},
	}

	selected, err := b.Select(context.Background(), ss)
	require.NoError(t, err)
	assert.Contains(t, []string{"a", "b"}, selected.ID)
}

func TestRandomBalancer_Empty(t *testing.T) {
	b := NewRandomBalancer()
	ss := &ServiceSet{Instances: []*ServiceInstance{}}
	_, err := b.Select(context.Background(), ss)
	assert.Equal(t, ErrEmptyServiceSet, err)
}

func TestRandomBalancer_Name(t *testing.T) {
	b := NewRandomBalancer()
	assert.Equal(t, "random", b.Name())
}
