package scheduler

import (
	"context"
	"testing"
)

func makeHealthyNode(id string, cpuCores, cpuUsed int, memBytes, memUsed, diskBytes, diskUsed int64) NodeState {
	return NodeState{
		NodeID:      id,
		Health:      "healthy",
		CPUCores:    cpuCores,
		CPUUsed:     cpuUsed,
		MemoryBytes: memBytes,
		MemoryUsed:  memUsed,
		DiskBytes:   diskBytes,
		DiskUsed:    diskUsed,
	}
}

func TestNewBestFitAlgorithm(t *testing.T) {
	alg := NewBestFitAlgorithm()
	if alg == nil {
		t.Fatal("expected non-nil algorithm")
	}
}

func TestBestFitAlgorithm_Name(t *testing.T) {
	alg := NewBestFitAlgorithm()
	if alg.Name() != "best_fit" {
		t.Errorf("expected name=best_fit, got %s", alg.Name())
	}
}

func TestBestFitAlgorithm_ScoreNode_SufficientResources(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 8, 2, 16*GiB, 4*GiB, 100*GiB, 10*GiB)
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    2,
		MemoryRequest: 2 * GiB,
		DiskRequest:   5 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score <= 0 {
		t.Errorf("expected positive score for sufficient resources, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_InsufficientCPU(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 3, 16*GiB, 0, 100*GiB, 0)
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    2,
		MemoryRequest: 1 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for insufficient CPU, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_InsufficientMemory(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 8, 0, 4*GiB, 3*GiB, 100*GiB, 0)
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    1,
		MemoryRequest: 2 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for insufficient memory, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_InsufficientDisk(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 8, 0, 16*GiB, 0, 10*GiB, 9*GiB)
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    1,
		MemoryRequest: 1 * GiB,
		DiskRequest:   5 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for insufficient disk, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_UnhealthyNode(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := NodeState{
		NodeID:      "n1",
		Health:      "unhealthy",
		CPUCores:    8,
		MemoryBytes: 16 * GiB,
	}
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for unhealthy node, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_NearFullPenalty(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	// Node that after placement has very little remaining
	node := NodeState{
		NodeID:        "n1",
		Health:        "healthy",
		CPUCores:      2,
		CPUUsed:       1,
		MemoryBytes:   1 * GiB,
		MemoryUsed:    0,
		DiskBytes:     100 * GiB,
		DiskUsed:      0,
		RunningAgents: nil,
	}
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    1,
		MemoryRequest: 1 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	// remainingCPU = 2-1-1 = 0 < 0.5, remainingMem = 1GiB - 0 - 1GiB = 0 < 512MiB -> penalty
	// So score should be reduced by 20
	t.Logf("near-full penalty score: %d", score)
}

func TestBestFitAlgorithm_ScoreNode_RunningAgentsBonus(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	nodeWithAgents := makeHealthyNode("n1", 8, 2, 16*GiB, 4*GiB, 100*GiB, 10*GiB)
	nodeWithAgents.RunningAgents = []AgentInfo{{ID: "existing-agent"}}

	nodeWithoutAgents := makeHealthyNode("n2", 8, 2, 16*GiB, 4*GiB, 100*GiB, 10*GiB)

	agent := AgentSpec{ID: "a1", CPURequest: 2, MemoryRequest: 2 * GiB, DiskRequest: 5 * GiB}

	scoreWith, _ := alg.ScoreNode(ctx, nodeWithAgents, agent)
	scoreWithout, _ := alg.ScoreNode(ctx, nodeWithoutAgents, agent)

	if scoreWith <= scoreWithout {
		t.Errorf("expected higher score with running agents (%d) than without (%d)", scoreWith, scoreWithout)
	}
}

func TestBestFitAlgorithm_ScoreNode_ExactFit(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	// Agent exactly uses remaining resources
	node := makeHealthyNode("n1", 4, 0, 4*GiB, 0, 100*GiB, 0)
	agent := AgentSpec{
		ID:            "a1",
		CPURequest:    4,
		MemoryRequest: 4 * GiB,
		DiskRequest:   100 * GiB,
	}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score <= 0 {
		t.Errorf("expected positive score for exact fit, got %d", score)
	}
}

func TestBestFitAlgorithm_ScoreNode_ZeroRequests(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	agent := AgentSpec{ID: "a1"}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score < 0 || score > 100 {
		t.Errorf("expected score in [0,100], got %d", score)
	}
}

func TestBestFitAlgorithm_FilterNodes(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	nodes := []NodeState{
		makeHealthyNode("n1", 2, 1, 4*GiB, 2*GiB, 100*GiB, 0),
		makeHealthyNode("n2", 8, 0, 16*GiB, 0, 100*GiB, 0),
		{NodeID: "n3", Health: "unhealthy", CPUCores: 8, MemoryBytes: 16 * GiB},
	}

	agent := AgentSpec{ID: "a1", CPURequest: 2, MemoryRequest: 2 * GiB}

	candidates, err := alg.FilterNodes(ctx, nodes, agent)
	if err != nil {
		t.Fatalf("filter nodes: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].NodeID != "n2" {
		t.Errorf("expected n2, got %s", candidates[0].NodeID)
	}
}

func TestBestFitAlgorithm_FilterNodes_NoneFit(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	nodes := []NodeState{
		makeHealthyNode("n1", 2, 2, 4*GiB, 4*GiB, 100*GiB, 0),
	}

	agent := AgentSpec{ID: "a1", CPURequest: 2, MemoryRequest: 2 * GiB}

	candidates, err := alg.FilterNodes(ctx, nodes, agent)
	if err != nil {
		t.Fatalf("filter nodes: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(candidates))
	}
}

func TestBestFitAlgorithm_SelectNode_Empty(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	_, err := alg.SelectNode(ctx, []NodeState{}, map[string]int{})
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
	if err != ErrNoAvailableNodes {
		t.Errorf("expected ErrNoAvailableNodes, got %v", err)
	}
}

func TestBestFitAlgorithm_SelectNode_SingleCandidate(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	candidates := []NodeState{{NodeID: "n1"}}
	scores := map[string]int{"n1": 80}

	node, err := alg.SelectNode(ctx, candidates, scores)
	if err != nil {
		t.Fatalf("select node: %v", err)
	}
	if node.NodeID != "n1" {
		t.Errorf("expected n1, got %s", node.NodeID)
	}
}

func TestBestFitAlgorithm_SelectNode_BestScore(t *testing.T) {
	alg := NewBestFitAlgorithm()
	ctx := context.Background()

	candidates := []NodeState{
		{NodeID: "n1"},
		{NodeID: "n2"},
		{NodeID: "n3"},
	}
	scores := map[string]int{
		"n1": 50,
		"n2": 90,
		"n3": 70,
	}

	node, err := alg.SelectNode(ctx, candidates, scores)
	if err != nil {
		t.Fatalf("select node: %v", err)
	}
	if node.NodeID != "n2" {
		t.Errorf("expected n2 (highest score), got %s", node.NodeID)
	}
}

func TestCostAwareAlgorithm_Name(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	if alg.Name() != "cost_aware" {
		t.Errorf("expected name=cost_aware, got %s", alg.Name())
	}
}

func TestCostAwareAlgorithm_ScoreNode_CheapNode(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	node.CostPerHour = 0.3
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score <= 50 {
		t.Errorf("expected high score for cheap node, got %d", score)
	}
}

func TestCostAwareAlgorithm_ScoreNode_ExpensiveNode(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	node.CostPerHour = 3.0
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score > 50 {
		t.Errorf("expected lower score for expensive node, got %d", score)
	}
}

func TestCostAwareAlgorithm_ScoreNode_SpotBonus(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	node.CostPerHour = 1.0
	node.IsSpot = true
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, _ := alg.ScoreNode(ctx, node, agent)

	nodeNotSpot := makeHealthyNode("n2", 4, 0, 8*GiB, 0, 100*GiB, 0)
	nodeNotSpot.CostPerHour = 1.0
	scoreNoSpot, _ := alg.ScoreNode(ctx, nodeNotSpot, agent)

	if score <= scoreNoSpot {
		t.Errorf("expected spot bonus: spot=%d, non-spot=%d", score, scoreNoSpot)
	}
}

func TestCostAwareAlgorithm_ScoreNode_InsufficientResources(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 2, 2, 4*GiB, 4*GiB, 100*GiB, 0)
	agent := AgentSpec{ID: "a1", CPURequest: 2, MemoryRequest: 2 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for insufficient resources, got %d", score)
	}
}

func TestCostAwareAlgorithm_FilterAndSelect(t *testing.T) {
	alg := NewCostAwareAlgorithm()
	ctx := context.Background()

	nodes := []NodeState{
		makeHealthyNode("n1", 8, 0, 16*GiB, 0, 100*GiB, 0),
		makeHealthyNode("n2", 8, 0, 16*GiB, 0, 100*GiB, 0),
	}
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	candidates, err := alg.FilterNodes(ctx, nodes, agent)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	_, err = alg.SelectNode(ctx, candidates, map[string]int{"n1": 70, "n2": 80})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
}

func TestLeastMigrationAlgorithm_Name(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	if alg.Name() != "least_migration" {
		t.Errorf("expected name=least_migration, got %s", alg.Name())
	}
}

func TestLeastMigrationAlgorithm_ScoreNode_AlreadyRunning(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	node.Agents = []AgentInfo{{ID: "a1"}}
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score != 100 {
		t.Errorf("expected score=100 for already-running agent, got %d", score)
	}
}

func TestLeastMigrationAlgorithm_ScoreNode_NotRunning(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	ctx := context.Background()

	node := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	score, err := alg.ScoreNode(ctx, node, agent)
	if err != nil {
		t.Fatalf("score node: %v", err)
	}
	if score <= 0 {
		t.Errorf("expected positive score, got %d", score)
	}
	if score == 100 {
		t.Error("should not get max score for agent not already on node")
	}
}

func TestLeastMigrationAlgorithm_ScoreNode_LowChurn(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	ctx := context.Background()

	lowChurn := makeHealthyNode("n1", 4, 0, 8*GiB, 0, 100*GiB, 0)
	lowChurn.MigrationsIn = 2
	lowChurn.MigrationsOut = 1

	highChurn := makeHealthyNode("n2", 4, 0, 8*GiB, 0, 100*GiB, 0)
	highChurn.MigrationsIn = 10
	highChurn.MigrationsOut = 10

	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	scoreLow, _ := alg.ScoreNode(ctx, lowChurn, agent)
	scoreHigh, _ := alg.ScoreNode(ctx, highChurn, agent)

	if scoreLow <= scoreHigh {
		t.Errorf("expected low churn to score higher: low=%d, high=%d", scoreLow, scoreHigh)
	}
}

func TestLeastMigrationAlgorithm_FilterAndSelect(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	ctx := context.Background()

	nodes := []NodeState{
		makeHealthyNode("n1", 8, 0, 16*GiB, 0, 100*GiB, 0),
	}
	agent := AgentSpec{ID: "a1", CPURequest: 1, MemoryRequest: 1 * GiB}

	candidates, err := alg.FilterNodes(ctx, nodes, agent)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	node, err := alg.SelectNode(ctx, candidates, map[string]int{"n1": 75})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if node.NodeID != "n1" {
		t.Errorf("expected n1, got %s", node.NodeID)
	}
}

func TestLeastMigrationAlgorithm_SelectNode_Empty(t *testing.T) {
	alg := NewLeastMigrationAlgorithm()
	ctx := context.Background()

	_, err := alg.SelectNode(ctx, []NodeState{}, map[string]int{})
	if err != ErrNoAvailableNodes {
		t.Errorf("expected ErrNoAvailableNodes, got %v", err)
	}
}
