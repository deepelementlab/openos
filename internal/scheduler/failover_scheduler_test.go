package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFailoverScheduler(t *testing.T) {
	fs := NewFailoverScheduler()
	assert.NotNil(t, fs)
}

func TestFailoverScheduler_Initialize(t *testing.T) {
	fs := NewFailoverScheduler()
	err := fs.Initialize(context.Background(), &config.Config{})
	require.NoError(t, err)
}

func TestFailoverScheduler_Schedule_Basic(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{
		NodeID: "node-1", NodeName: "primary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy",
	})

	result, err := fs.Schedule(ctx, TaskRequest{TaskID: "task-1", CPURequest: 500, MemoryRequest: 1024})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "node-1", result.NodeID)
}

func TestFailoverScheduler_Schedule_NoNodes(t *testing.T) {
	fs := NewFailoverScheduler()
	result, err := fs.Schedule(context.Background(), TaskRequest{TaskID: "task-1"})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "no_eligible_node", result.Reason)
}

func TestFailoverScheduler_PrimaryPreference(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "primary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.AddNode(ctx, NodeState{NodeID: "secondary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})

	result, err := fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "primary", result.NodeID)
}

func TestFailoverScheduler_NodeFailover(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "primary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.AddNode(ctx, NodeState{NodeID: "secondary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})

	fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})
	fs.Schedule(ctx, TaskRequest{TaskID: "task-2"})

	migrated, err := fs.MarkNodeFailed(ctx, "primary")
	require.NoError(t, err)
	assert.Equal(t, 2, migrated)

	fs.mu.RLock()
	node, _ := fs.nodes["primary"]
	assert.Equal(t, "unhealthy", node.state.Health)
	fs.mu.RUnlock()
}

func TestFailoverScheduler_CancelTask(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})

	err := fs.CancelTask(ctx, "task-1")
	require.NoError(t, err)

	fs.mu.RLock()
	_, exists := fs.allocations["task-1"]
	fs.mu.RUnlock()
	assert.False(t, exists)
}

func TestFailoverScheduler_CancelTask_NonExistent(t *testing.T) {
	fs := NewFailoverScheduler()
	err := fs.CancelTask(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestFailoverScheduler_RemoveNode(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.AddNode(ctx, NodeState{NodeID: "node-2", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})

	err := fs.RemoveNode(ctx, "node-1")
	require.NoError(t, err)

	nodes, _ := fs.ListNodes(ctx)
	assert.Len(t, nodes, 1)

	fs.mu.RLock()
	_, exists := fs.allocations["task-1"]
	fs.mu.RUnlock()
	assert.False(t, exists)
}

func TestFailoverScheduler_RemoveNode_NotFound(t *testing.T) {
	fs := NewFailoverScheduler()
	err := fs.RemoveNode(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestFailoverScheduler_UpdateNode(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})

	err := fs.UpdateNode(ctx, NodeState{NodeID: "node-1", CPUCores: 16, MemoryBytes: 32 * 1024 * 1024 * 1024, Health: "healthy"})
	require.NoError(t, err)

	nodes, _ := fs.ListNodes(ctx)
	assert.Equal(t, 16, nodes[0].CPUCores)
}

func TestFailoverScheduler_UpdateNode_NotFound(t *testing.T) {
	fs := NewFailoverScheduler()
	err := fs.UpdateNode(context.Background(), NodeState{NodeID: "nonexistent"})
	assert.Error(t, err)
}

func TestFailoverScheduler_HealthCheck(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	assert.NoError(t, fs.HealthCheck(ctx))

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 4, MemoryBytes: 8 * 1024 * 1024 * 1024, Health: "healthy"})
	assert.NoError(t, fs.HealthCheck(ctx))

	fs.UpdateNode(ctx, NodeState{NodeID: "node-1", CPUCores: 4, MemoryBytes: 8 * 1024 * 1024 * 1024, Health: "unhealthy"})
	assert.Error(t, fs.HealthCheck(ctx))
}

func TestFailoverScheduler_GetStats(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 4, MemoryBytes: 8 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})

	stats, err := fs.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalScheduled)
	assert.Equal(t, 1, stats.NodeCount)
}

func TestFailoverScheduler_ScheduleBatch(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 32, MemoryBytes: 64 * 1024 * 1024 * 1024, Health: "healthy"})

	reqs := []TaskRequest{
		{TaskID: "t1", CPURequest: 100},
		{TaskID: "t2", CPURequest: 200},
		{TaskID: "t3", CPURequest: 300},
	}

	results, err := fs.ScheduleBatch(ctx, reqs)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	for _, r := range results {
		assert.True(t, r.Success)
	}
}

func TestFailoverScheduler_Reschedule(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.AddNode(ctx, NodeState{NodeID: "node-2", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})
	fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})

	result, err := fs.Reschedule(ctx, "task-1", RescheduleReasonOptimization)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestFailoverScheduler_AddNode_EmptyID(t *testing.T) {
	fs := NewFailoverScheduler()
	err := fs.AddNode(context.Background(), NodeState{NodeID: ""})
	assert.Error(t, err)
}

func TestFailoverScheduler_MarkNodeFailed_NotFound(t *testing.T) {
	fs := NewFailoverScheduler()
	_, err := fs.MarkNodeFailed(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestFailoverScheduler_EnqueueAndProcess(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "node-1", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})

	fs.EnqueueTask(TaskRequest{TaskID: "t1", Priority: 5})
	fs.EnqueueTask(TaskRequest{TaskID: "t2", Priority: 10})
	fs.EnqueueTask(TaskRequest{TaskID: "t3", Priority: 1})

	results, err := fs.ProcessQueue(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, "t2", results[0].TaskID)
}

func TestPriorityQueue_DefaultOrder(t *testing.T) {
	pq := NewPriorityQueue(DefaultLess)

	pq.Enqueue(TaskRequest{TaskID: "low", Priority: 1})
	pq.Enqueue(TaskRequest{TaskID: "high", Priority: 10})
	pq.Enqueue(TaskRequest{TaskID: "med", Priority: 5})

	item, ok := pq.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "high", item.Task.TaskID)

	item, ok = pq.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "med", item.Task.TaskID)

	item, ok = pq.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "low", item.Task.TaskID)

	_, ok = pq.Dequeue()
	assert.False(t, ok)
}

func TestPriorityQueue_Size(t *testing.T) {
	pq := NewPriorityQueue(DefaultLess)
	assert.Equal(t, 0, pq.Size())

	pq.Enqueue(TaskRequest{TaskID: "t1"})
	pq.Enqueue(TaskRequest{TaskID: "t2"})
	assert.Equal(t, 2, pq.Size())
}

func TestPriorityQueue_FIFO_SamePriority(t *testing.T) {
	pq := NewPriorityQueue(DefaultLess)

	pq.Enqueue(TaskRequest{TaskID: "first", Priority: 5})
	time.Sleep(time.Millisecond)
	pq.Enqueue(TaskRequest{TaskID: "second", Priority: 5})

	item, _ := pq.Dequeue()
	assert.Equal(t, "first", item.Task.TaskID)

	item, _ = pq.Dequeue()
	assert.Equal(t, "second", item.Task.TaskID)
}

func TestFailoverScheduler_PrimaryFailoverSelection(t *testing.T) {
	fs := NewFailoverScheduler()
	ctx := context.Background()

	fs.AddNode(ctx, NodeState{NodeID: "primary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "unhealthy"})
	fs.AddNode(ctx, NodeState{NodeID: "secondary", CPUCores: 8, MemoryBytes: 16 * 1024 * 1024 * 1024, Health: "healthy"})

	result, err := fs.Schedule(ctx, TaskRequest{TaskID: "task-1"})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "secondary", result.NodeID)
}
