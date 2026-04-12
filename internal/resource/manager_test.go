package resource

import (
	"context"
	"testing"
)

func TestNodeResources_AvailableCalculations(t *testing.T) {
	n := NodeResources{
		NodeID:         "node-1",
		TotalCPU:       8.0,
		TotalMemory:    16 * 1024 * 1024 * 1024,
		TotalDisk:      100 * 1024 * 1024 * 1024,
		AllocatedCPU:   2.0,
		AllocatedMemory: 4 * 1024 * 1024 * 1024,
		AllocatedDisk:  20 * 1024 * 1024 * 1024,
	}

	if n.AvailableCPU() != 6.0 {
		t.Fatalf("expected 6.0 available CPU, got %f", n.AvailableCPU())
	}
	if n.AvailableMemory() != 12*1024*1024*1024 {
		t.Fatalf("expected 12GB available memory")
	}
	if n.AvailableDisk() != 80*1024*1024*1024 {
		t.Fatalf("expected 80GB available disk")
	}
}

func TestInMemoryResourceManager_RegisterNode(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	err := m.RegisterNode(ctx, NodeResources{
		NodeID:      "node-1",
		TotalCPU:    4.0,
		TotalMemory: 8 * 1024 * 1024 * 1024,
		TotalDisk:   100 * 1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("register node failed: %v", err)
	}

	// Verify node exists
	node, err := m.GetNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("get node failed: %v", err)
	}
	if node.TotalCPU != 4.0 {
		t.Fatalf("expected 4.0 CPU, got %f", node.TotalCPU)
	}

	// Empty node ID should fail
	err = m.RegisterNode(ctx, NodeResources{NodeID: ""})
	if err == nil {
		t.Fatal("expected error for empty node ID")
	}
}

func TestInMemoryResourceManager_AllocateAndRelease(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	// Register node
	m.RegisterNode(ctx, NodeResources{
		NodeID:      "node-1",
		TotalCPU:    4.0,
		TotalMemory: 8 * 1024 * 1024 * 1024,
		TotalDisk:   100 * 1024 * 1024 * 1024,
	})

	// Allocate resources
	err := m.Allocate(ctx, AllocationRequest{
		AgentID: "agent-1",
		NodeID:  "node-1",
		CPU:     1.0,
		Memory:  2 * 1024 * 1024 * 1024,
		Disk:    10 * 1024 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("allocate failed: %v", err)
	}

	// Verify allocation
	node, _ := m.GetNode(ctx, "node-1")
	if node.AllocatedCPU != 1.0 {
		t.Fatalf("expected 1.0 allocated CPU, got %f", node.AllocatedCPU)
	}
	if node.AvailableCPU() != 3.0 {
		t.Fatalf("expected 3.0 available CPU, got %f", node.AvailableCPU())
	}

	// Release resources
	err = m.Release(ctx, "agent-1")
	if err != nil {
		t.Fatalf("release failed: %v", err)
	}

	// Verify release
	node, _ = m.GetNode(ctx, "node-1")
	if node.AllocatedCPU != 0.0 {
		t.Fatalf("expected 0.0 allocated CPU after release, got %f", node.AllocatedCPU)
	}

	// Release non-existent agent
	err = m.Release(ctx, "agent-999")
	if err == nil {
		t.Fatal("expected error for releasing non-existent agent")
	}
}

func TestInMemoryResourceManager_InsufficientResources(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	m.RegisterNode(ctx, NodeResources{
		NodeID:      "node-1",
		TotalCPU:    2.0,
		TotalMemory: 4 * 1024 * 1024 * 1024,
		TotalDisk:   10 * 1024 * 1024 * 1024,
	})

	// Try to allocate more CPU than available
	err := m.Allocate(ctx, AllocationRequest{
		AgentID: "agent-1",
		NodeID:  "node-1",
		CPU:     4.0,
		Memory:  1 * 1024 * 1024 * 1024,
		Disk:    1 * 1024 * 1024 * 1024,
	})
	if err == nil {
		t.Fatal("expected error for insufficient CPU")
	}

	// Try to allocate more memory than available
	err = m.Allocate(ctx, AllocationRequest{
		AgentID: "agent-1",
		NodeID:  "node-1",
		CPU:     1.0,
		Memory:  8 * 1024 * 1024 * 1024,
		Disk:    1 * 1024 * 1024 * 1024,
	})
	if err == nil {
		t.Fatal("expected error for insufficient memory")
	}
}

func TestInMemoryResourceManager_CheckQuota(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	m.RegisterNode(ctx, NodeResources{
		NodeID:      "node-1",
		TotalCPU:    4.0,
		TotalMemory: 8 * 1024 * 1024 * 1024,
		TotalDisk:   100 * 1024 * 1024 * 1024,
	})

	// Check valid quota
	ok, msg := m.CheckQuota(ctx, AllocationRequest{
		NodeID: "node-1",
		CPU:    2.0,
		Memory: 4 * 1024 * 1024 * 1024,
	})
	if !ok {
		t.Fatalf("expected quota check to pass, got: %s", msg)
	}

	// Check over-quota
	ok, msg = m.CheckQuota(ctx, AllocationRequest{
		NodeID: "node-1",
		CPU:    8.0,
		Memory: 4 * 1024 * 1024 * 1024,
	})
	if ok {
		t.Fatal("expected quota check to fail")
	}

	// Check non-existent node
	ok, msg = m.CheckQuota(ctx, AllocationRequest{
		NodeID: "node-999",
		CPU:    1.0,
	})
	if ok {
		t.Fatal("expected quota check to fail for non-existent node")
	}
}

func TestInMemoryResourceManager_UnregisterNode(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	m.RegisterNode(ctx, NodeResources{NodeID: "node-1", TotalCPU: 4.0})

	err := m.UnregisterNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("unregister failed: %v", err)
	}

	_, err = m.GetNode(ctx, "node-1")
	if err == nil {
		t.Fatal("expected error for unregistered node")
	}

	// Unregister non-existent node
	err = m.UnregisterNode(ctx, "node-999")
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
}

func TestInMemoryResourceManager_ListNodes(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	m.RegisterNode(ctx, NodeResources{NodeID: "node-1", TotalCPU: 4.0})
	m.RegisterNode(ctx, NodeResources{NodeID: "node-2", TotalCPU: 8.0})

	nodes, err := m.ListNodes(ctx)
	if err != nil {
		t.Fatalf("list nodes failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestInMemoryResourceManager_AllocateNonExistentNode(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	err := m.Allocate(ctx, AllocationRequest{
		AgentID: "agent-1",
		NodeID:  "node-999",
		CPU:     1.0,
	})
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
}

func TestInMemoryResourceManager_GetNodeNotFound(t *testing.T) {
	m := NewInMemoryResourceManager()
	ctx := context.Background()

	_, err := m.GetNode(ctx, "node-999")
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
}
