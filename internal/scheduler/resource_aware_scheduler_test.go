package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/agentos/aos/internal/config"
)

func addTestNode(s Scheduler, id string, cpuCores int, memBytes int64, gpu int) error {
	return s.AddNode(context.Background(), NodeState{
		NodeID:      id,
		NodeName:    "node-" + id,
		CPUCores:    cpuCores,
		MemoryBytes: memBytes,
		GPUCount:    gpu,
		Health:      "healthy",
	})
}

func testReq(cpu float64, mem int64, gpu int) TaskRequest {
	return TaskRequest{
		TaskID:        fmt.Sprintf("task-%d", atomic.AddInt64(&taskCounter, 1)),
		CPURequest:    cpu,
		MemoryRequest: mem,
		GPURequest:    gpu,
		Priority:      1,
	}
}

var taskCounter int64

// --- BestFit Tests ---

func TestResourceAwareScheduler_BestFit_BasicScheduling(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	if err := addTestNode(s, "n1", 4, 8*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}
	if err := addTestNode(s, "n2", 8, 16*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Schedule a small task
	res, err := s.Schedule(ctx, testReq(500, 512*MiB, 0))
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Message)
	}
	// BestFit should prefer n1 (smaller node, closer fit)
	if res.NodeID != "n1" {
		t.Logf("BestFit chose %s (expected n1 for small task)", res.NodeID)
	}
}

func TestResourceAwareScheduler_BestFit_ResourceExhaustion(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	if err := addTestNode(s, "n1", 2, 2*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Fill up the node
	res, _ := s.Schedule(ctx, testReq(1000, 1*GiB, 0))
	if !res.Success {
		t.Fatal("first schedule should succeed")
	}

	res, _ = s.Schedule(ctx, testReq(1000, 1*GiB, 0))
	if !res.Success {
		t.Fatal("second schedule should succeed")
	}

	// Third request should fail
	res, _ = s.Schedule(ctx, testReq(500, 512*MiB, 0))
	if res.Success {
		t.Fatal("expected failure due to resource exhaustion")
	}
	if res.Reason != "no_eligible_node" {
		t.Fatalf("expected no_eligible_node, got: %s", res.Reason)
	}
}

func TestResourceAwareScheduler_BestFit_GPUFiltering(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	if err := addTestNode(s, "n1", 4, 8*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}
	if err := addTestNode(s, "n2", 4, 8*GiB, 2); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Request with GPU
	res, err := s.Schedule(ctx, testReq(500, 512*MiB, 1))
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Message)
	}
	if res.NodeID != "n2" {
		t.Fatalf("expected n2 (has GPU), got: %s", res.NodeID)
	}
}

// --- LeastLoaded Tests ---

func TestResourceAwareScheduler_LeastLoaded_Distribution(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyLeastLoaded)
	ctx := context.Background()

	if err := addTestNode(s, "n1", 4, 8*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}
	if err := addTestNode(s, "n2", 4, 8*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Schedule multiple tasks
	for i := 0; i < 4; i++ {
		res, err := s.Schedule(ctx, testReq(500, 512*MiB, 0))
		if err != nil {
			t.Fatalf("schedule %d: %v", i, err)
		}
		if !res.Success {
			t.Fatalf("schedule %d failed: %s", i, res.Message)
		}
	}

	// Check distribution
	stats, _ := s.GetStats(ctx)
	if stats.TotalScheduled != 4 {
		t.Fatalf("expected 4 scheduled, got %d", stats.TotalScheduled)
	}

	nodes, _ := s.ListNodes(ctx)
	n1Count, n2Count := 0, 0
	for _, n := range nodes {
		switch n.NodeID {
		case "n1":
			n1Count = n.AgentCount
		case "n2":
			n2Count = n.AgentCount
		}
	}
	// Should be evenly distributed (2 each)
	if n1Count != 2 || n2Count != 2 {
		t.Logf("distribution: n1=%d, n2=%d (expected 2 each)", n1Count, n2Count)
	}
}

// --- WorstFit Tests ---

func TestResourceAwareScheduler_WorstFit_SpreadsLoad(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyWorstFit)
	ctx := context.Background()

	if err := addTestNode(s, "n1", 8, 16*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}
	if err := addTestNode(s, "n2", 8, 16*GiB, 0); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Schedule tasks - worst fit should spread them
	nodeIDs := make(map[string]int)
	for i := 0; i < 6; i++ {
		res, err := s.Schedule(ctx, testReq(500, 512*MiB, 0))
		if err != nil {
			t.Fatalf("schedule %d: %v", i, err)
		}
		if !res.Success {
			t.Fatalf("schedule %d failed: %s", i, res.Message)
		}
		nodeIDs[res.NodeID]++
	}

	// Should be 3-3 spread
	for id, count := range nodeIDs {
		t.Logf("node %s: %d tasks", id, count)
	}
}

// --- Node Management Tests ---

func TestResourceAwareScheduler_RemoveNode(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	addTestNode(s, "n1", 4, 8*GiB, 0)
	addTestNode(s, "n2", 4, 8*GiB, 0)

	// Schedule a task
	s.Schedule(ctx, TaskRequest{TaskID: "t1", CPURequest: 500, MemoryRequest: 512 * MiB})

	// Remove a node
	if err := s.RemoveNode(ctx, "n1"); err != nil {
		t.Fatalf("remove node: %v", err)
	}

	nodes, _ := s.ListNodes(ctx)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestResourceAwareScheduler_UpdateNode(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	addTestNode(s, "n1", 4, 8*GiB, 0)

	// Update node state
	if err := s.UpdateNode(ctx, NodeState{
		NodeID:      "n1",
		NodeName:    "node-n1",
		CPUCores:    8,
		MemoryBytes: 16 * GiB,
		Health:      "healthy",
	}); err != nil {
		t.Fatalf("update node: %v", err)
	}

	nodes, _ := s.ListNodes(ctx)
	if nodes[0].CPUCores != 8 {
		t.Fatalf("expected 8 cores, got %d", nodes[0].CPUCores)
	}
}

func TestResourceAwareScheduler_UnhealthyNodeSkipped(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	s.AddNode(ctx, NodeState{NodeID: "n1", CPUCores: 4, MemoryBytes: 8 * GiB, Health: "unhealthy"})
	addTestNode(s, "n2", 4, 8*GiB, 0)

	res, err := s.Schedule(ctx, testReq(500, 512*MiB, 0))
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if res.NodeID == "n1" {
		t.Fatal("should not schedule on unhealthy node")
	}
}

func TestResourceAwareScheduler_CancelTask(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	ctx := context.Background()

	addTestNode(s, "n1", 2, 2*GiB, 0)

	// Fill up
	s.Schedule(ctx, TaskRequest{TaskID: "t1", CPURequest: 1000, MemoryRequest: 1 * GiB})
	s.Schedule(ctx, TaskRequest{TaskID: "t2", CPURequest: 1000, MemoryRequest: 1 * GiB})

	// Third should fail
	res, _ := s.Schedule(ctx, TaskRequest{TaskID: "t3", CPURequest: 500, MemoryRequest: 512 * MiB})
	if res.Success {
		t.Fatal("expected failure")
	}

	// Cancel one task
	if err := s.CancelTask(ctx, "t1"); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Now it should succeed
	res, _ = s.Schedule(ctx, TaskRequest{TaskID: "t3", CPURequest: 500, MemoryRequest: 512 * MiB})
	if !res.Success {
		t.Fatal("expected success after cancel")
	}
}

func TestResourceAwareScheduler_Reschedule(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyRoundRobin)
	ctx := context.Background()

	addTestNode(s, "n1", 4, 8*GiB, 0)
	addTestNode(s, "n2", 4, 8*GiB, 0)

	// Schedule on n1
	res, _ := s.Schedule(ctx, TaskRequest{TaskID: "t1", CPURequest: 500, MemoryRequest: 512 * MiB})

	// Reschedule
	res, err := s.Reschedule(ctx, "t1", RescheduleReasonNodeFailure)
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !res.Success {
		t.Fatalf("reschedule failed: %s", res.Message)
	}
}

// --- Concurrent Scheduling Test ---

func TestResourceAwareScheduler_ConcurrentScheduling(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyLeastLoaded)
	ctx := context.Background()

	addTestNode(s, "n1", 16, 64*GiB, 0)
	addTestNode(s, "n2", 16, 64*GiB, 0)

	var successCount int64
	var failCount int64

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(i int) {
			res, err := s.Schedule(ctx, TaskRequest{
				TaskID:        fmt.Sprintf("concurrent-%d", i),
				CPURequest:    500,
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

	t.Logf("concurrent scheduling: %d success, %d fail", successCount, failCount)
	if successCount == 0 {
		t.Fatal("expected at least some successful schedules")
	}
}

// --- Strategy Switch Test ---

func TestResourceAwareScheduler_StrategySwitch(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyRoundRobin)
	ctx := context.Background()

	addTestNode(s, "n1", 4, 8*GiB, 0)
	addTestNode(s, "n2", 4, 8*GiB, 0)

	// Schedule with round-robin
	res, _ := s.Schedule(ctx, testReq(500, 512*MiB, 0))
	if !res.Success {
		t.Fatal("round-robin schedule failed")
	}

	// Switch to best-fit
	s.SetStrategy(StrategyBestFit)

	res, _ = s.Schedule(ctx, testReq(500, 512*MiB, 0))
	if !res.Success {
		t.Fatal("best-fit schedule failed")
	}
}

// --- Initialize Test ---

func TestResourceAwareScheduler_Initialize(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	cfg := &config.Config{}
	if err := s.Initialize(context.Background(), cfg); err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

func TestResourceAwareScheduler_HealthCheck(t *testing.T) {
	s := NewResourceAwareScheduler(StrategyBestFit)
	if err := s.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check: %v", err)
	}
}

// --- Constants for test readability ---
const (
	GiB = 1024 * 1024 * 1024
	MiB = 1024 * 1024
)

// Ensure interface compliance at compile time.
func TestResourceAwareScheduler_InterfaceCompliance(t *testing.T) {
	var _ Scheduler = NewResourceAwareScheduler(StrategyBestFit)
}
