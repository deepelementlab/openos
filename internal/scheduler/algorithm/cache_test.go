package algorithm

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/internal/scheduler"
	"github.com/stretchr/testify/require"
)

func TestNodeStateCache_UpsertSnapshot(t *testing.T) {
	c := NewNodeStateCache(time.Minute)
	c.Upsert(scheduler.NodeState{NodeID: "a", CPUCores: 4})
	require.Equal(t, 1, c.Len())
	snap := c.Snapshot()
	require.Len(t, snap, 1)
	require.False(t, c.Stale())
}

func TestScoreCache_WithBestFit(t *testing.T) {
	alg := scheduler.NewBestFitAlgorithm()
	cache := NewScoreCache()
	agent := scheduler.AgentSpec{CPURequest: 1, MemoryRequest: 128, DiskRequest: 1}
	nodes := []scheduler.NodeState{
		{
			NodeID: "n1", CPUCores: 8, MemoryBytes: 1 << 30, DiskBytes: 1 << 40,
			Health: "healthy",
		},
	}
	ctx := context.Background()
	m1, err := ScoreNodesWithCache(alg, ctx, nodes, agent, "worker", cache)
	require.NoError(t, err)
	m2, err := ScoreNodesWithCache(alg, ctx, nodes, agent, "worker", cache)
	require.NoError(t, err)
	require.Equal(t, m1, m2)
	cache.InvalidateNode("n1")
	_, err = ScoreNodesWithCache(alg, ctx, nodes, agent, "worker", cache)
	require.NoError(t, err)
}
