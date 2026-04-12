package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/agentos/aos/internal/config"
)

func TestNewDefaultScheduler(t *testing.T) {
	s := NewDefaultScheduler()
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestDefaultScheduler_Initialize(t *testing.T) {
	s := NewDefaultScheduler()
	cfg := &config.Config{}
	if err := s.Initialize(context.Background(), cfg); err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

func TestDefaultScheduler_HealthCheck(t *testing.T) {
	s := NewDefaultScheduler()
	if err := s.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check: %v", err)
	}
}

func TestDefaultScheduler_AddNode_Valid(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	err := s.AddNode(ctx, NodeState{
		NodeID:   "n1",
		NodeName: "node-1",
		Health:   "healthy",
		CPUCores: 4,
	})
	if err != nil {
		t.Fatalf("add node: %v", err)
	}

	nodes, err := s.ListNodes(ctx)
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].NodeID != "n1" {
		t.Errorf("expected node n1, got %s", nodes[0].NodeID)
	}
}

func TestDefaultScheduler_AddNode_EmptyID(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	err := s.AddNode(ctx, NodeState{NodeID: ""})
	if err == nil {
		t.Fatal("expected error for empty node ID")
	}
}

func TestDefaultScheduler_AddNode_Multiple(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := s.AddNode(ctx, NodeState{
			NodeID:   fmt.Sprintf("n%d", i),
			NodeName: fmt.Sprintf("node-%d", i),
			Health:   "healthy",
			CPUCores: 4,
		})
		if err != nil {
			t.Fatalf("add node %d: %v", i, err)
		}
	}

	nodes, _ := s.ListNodes(ctx)
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(nodes))
	}
}

func TestDefaultScheduler_RemoveNode_Existing(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 4})

	if err := s.RemoveNode(ctx, "n1"); err != nil {
		t.Fatalf("remove node: %v", err)
	}

	nodes, _ := s.ListNodes(ctx)
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestDefaultScheduler_RemoveNode_NonExisting(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	err := s.RemoveNode(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error removing non-existent node")
	}
}

func TestDefaultScheduler_Schedule_NoNodes(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	res, err := s.Schedule(ctx, TaskRequest{
		TaskID:        "t1",
		CPURequest:    1.0,
		MemoryRequest: 1 * GiB,
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure with no nodes")
	}
	if res.Reason != "no_eligible_node" {
		t.Errorf("expected reason=no_eligible_node, got %s", res.Reason)
	}
	if res.TaskID != "t1" {
		t.Errorf("expected TaskID=t1, got %s", res.TaskID)
	}
}

func TestDefaultScheduler_Schedule_SingleNode(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    4,
		MemoryBytes: 8 * GiB,
	})

	res, err := s.Schedule(ctx, TaskRequest{
		TaskID:        "t1",
		CPURequest:    1.0,
		MemoryRequest: 1 * GiB,
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Message)
	}
	if res.NodeID != "n1" {
		t.Errorf("expected NodeID=n1, got %s", res.NodeID)
	}
	if res.TaskID != "t1" {
		t.Errorf("expected TaskID=t1, got %s", res.TaskID)
	}
}

func TestDefaultScheduler_Schedule_MultipleNodes(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	for _, id := range []string{"n1", "n2", "n3"} {
		s.AddNode(ctx, NodeState{
			NodeID:      id,
			Health:      "healthy",
			CPUCores:    4,
			MemoryBytes: 8 * GiB,
		})
	}

	res, err := s.Schedule(ctx, TaskRequest{
		TaskID:        "t1",
		CPURequest:    1.0,
		MemoryRequest: 1 * GiB,
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !res.Success {
		t.Fatal("expected success")
	}
	if res.NodeID == "" {
		t.Error("expected non-empty NodeID")
	}
}

func TestDefaultScheduler_Schedule_RoundRobin(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})
	s.AddNode(ctx, NodeState{NodeID: "n2", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})

	ids := make(map[string]int)
	for i := 0; i < 4; i++ {
		res, err := s.Schedule(ctx, TaskRequest{
			TaskID:        fmt.Sprintf("t%d", i),
			CPURequest:    1.0,
			MemoryRequest: 1 * GiB,
		})
		if err != nil {
			t.Fatalf("schedule %d: %v", i, err)
		}
		if !res.Success {
			t.Fatalf("schedule %d failed: %s", i, res.Message)
		}
		ids[res.NodeID]++
	}

	if len(ids) < 2 {
		t.Errorf("round-robin should distribute across nodes, got: %v", ids)
	}
}

func TestDefaultScheduler_Schedule_UnhealthyNodeSkipped(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "unhealthy", CPUCores: 4, MemoryBytes: 8 * GiB})
	s.AddNode(ctx, NodeState{NodeID: "n2", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})

	res, err := s.Schedule(ctx, TaskRequest{
		TaskID:        "t1",
		CPURequest:    1.0,
		MemoryRequest: 1 * GiB,
	})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !res.Success {
		t.Fatal("expected success")
	}
	if res.NodeID == "n1" {
		t.Error("should not schedule on unhealthy node")
	}
}

func TestDefaultScheduler_Schedule_InsufficientCPU(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    4,
		MemoryBytes: 8 * GiB,
		CPUUsage:    99.0,
	})

	res, _ := s.Schedule(ctx, TaskRequest{
		TaskID:     "t1",
		CPURequest: 1.0,
	})
	if res.Success {
		t.Fatal("expected failure due to insufficient CPU")
	}
}

func TestDefaultScheduler_Schedule_InsufficientMemory(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    4,
		MemoryBytes: 8 * GiB,
		MemoryUsage: 99.0,
	})

	res, _ := s.Schedule(ctx, TaskRequest{
		TaskID:        "t1",
		MemoryRequest: 4 * GiB,
	})
	if res.Success {
		t.Fatal("expected failure due to insufficient memory")
	}
}

func TestDefaultScheduler_ScheduleBatch(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 8, MemoryBytes: 32 * GiB})

	reqs := []TaskRequest{
		{TaskID: "t1", CPURequest: 1.0, MemoryRequest: 1 * GiB},
		{TaskID: "t2", CPURequest: 1.0, MemoryRequest: 1 * GiB},
		{TaskID: "t3", CPURequest: 1.0, MemoryRequest: 1 * GiB},
	}

	results, err := s.ScheduleBatch(ctx, reqs)
	if err != nil {
		t.Fatalf("schedule batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("expected success for task %s, got: %s", r.TaskID, r.Message)
		}
	}
}

func TestDefaultScheduler_Reschedule(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})

	res, err := s.Reschedule(ctx, "t1", RescheduleReasonNodeFailure)
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Message)
	}
}

func TestDefaultScheduler_CancelTask(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	if err := s.CancelTask(ctx, "nonexistent"); err != nil {
		t.Fatalf("cancel should not error: %v", err)
	}
}

func TestDefaultScheduler_UpdateNode(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})

	err := s.UpdateNode(ctx, NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    8,
		MemoryBytes: 16 * GiB,
	})
	if err != nil {
		t.Fatalf("update node: %v", err)
	}

	nodes, _ := s.ListNodes(ctx)
	if nodes[0].CPUCores != 8 {
		t.Errorf("expected 8 cores, got %d", nodes[0].CPUCores)
	}
	if nodes[0].MemoryBytes != 16*GiB {
		t.Errorf("expected 16 GiB, got %d", nodes[0].MemoryBytes)
	}
}

func TestDefaultScheduler_UpdateNode_NonExisting(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	err := s.UpdateNode(ctx, NodeState{NodeID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error updating non-existent node")
	}
}

func TestDefaultScheduler_GetStats(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 4, MemoryBytes: 8 * GiB})
	s.Schedule(ctx, TaskRequest{TaskID: "t1", CPURequest: 1.0, MemoryRequest: 1 * GiB})

	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.TotalScheduled != 1 {
		t.Errorf("expected 1 scheduled, got %d", stats.TotalScheduled)
	}
	if stats.NodeCount != 1 {
		t.Errorf("expected 1 node, got %d", stats.NodeCount)
	}
}

func TestDefaultScheduler_GetStats_FailedSchedules(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	res, _ := s.Schedule(ctx, TaskRequest{TaskID: "t1", CPURequest: 1.0, MemoryRequest: 1 * GiB})
	if res.Success {
		t.Fatal("expected failure with no nodes")
	}

	stats, _ := s.GetStats(ctx)
	if stats.TotalFailed != 1 {
		t.Errorf("expected 1 failed, got %d", stats.TotalFailed)
	}
}

func TestDefaultScheduler_ConcurrentOperations(t *testing.T) {
	s := NewDefaultScheduler()
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", Health: "healthy", CPUCores: 16, MemoryBytes: 64 * GiB})
	s.AddNode(ctx, NodeState{NodeID: "n2", Health: "healthy", CPUCores: 16, MemoryBytes: 64 * GiB})

	var successCount int64
	var failCount int64

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(i int) {
			res, err := s.Schedule(ctx, TaskRequest{
				TaskID:        fmt.Sprintf("concurrent-%d", i),
				CPURequest:    1.0,
				MemoryRequest: 512 * MiB,
			})
			if err != nil {
				t.Logf("goroutine %d error: %v", i, err)
				atomic.AddInt64(&failCount, 1)
			} else if res.Success {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	if successCount == 0 {
		t.Fatal("expected at least some successful schedules")
	}
	t.Logf("concurrent: %d success, %d fail", successCount, failCount)
}
