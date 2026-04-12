package scheduler

import (
	"context"
)

// BestFitAlgorithm implements best-fit scheduling.
// It selects the node that most tightly fits the agent's requirements.
type BestFitAlgorithm struct {
	BaseAlgorithm
}

// NewBestFitAlgorithm creates a new best-fit algorithm.
func NewBestFitAlgorithm() *BestFitAlgorithm {
	return &BestFitAlgorithm{
		BaseAlgorithm: BaseAlgorithm{
			config: AlgorithmConfig{
				Type:    AlgorithmBestFit,
				Weights: DefaultScoringWeights(),
			},
		},
	}
}

// Name returns the algorithm name.
func (b *BestFitAlgorithm) Name() string {
	return "best_fit"
}

// ScoreNode scores a node based on how well it fits the agent.
// Higher score means better fit (less wasted resources).
func (b *BestFitAlgorithm) ScoreNode(ctx context.Context, node NodeState, agent AgentSpec) (int, error) {
	if !b.CheckResources(node, agent) {
		return 0, nil // Node cannot fit the agent
	}

	// Calculate remaining resources after placement
	remainingCPU := node.CPUCores - node.CPUUsed - agent.CPURequest
	remainingMem := node.MemoryBytes - node.MemoryUsed - agent.MemoryRequest
	_ = node.DiskBytes - node.DiskUsed - agent.DiskRequest // For future use

	// Calculate utilization ratios (0-1, higher means more utilized)
	cpuRatio := float64(node.CPUUsed+agent.CPURequest) / float64(node.CPUCores)
	memRatio := float64(node.MemoryUsed+agent.MemoryRequest) / float64(node.MemoryBytes)
	diskRatio := float64(node.DiskUsed+agent.DiskRequest) / float64(node.DiskBytes)

	// Score based on how tightly the node is packed
	// We want nodes that will be well-utilized after placement
	avgUtilization := (cpuRatio + memRatio + diskRatio) / 3.0

	// Convert to 0-100 score, preferring higher utilization
	score := int(avgUtilization * 100)

	// Penalty for nodes that would have very little remaining resources
	// (to avoid over-packing)
	if float64(remainingCPU) < 0.5 && remainingMem < 512*1024*1024 {
		score -= 20 // Penalty for near-full nodes
	}

	// Bonus for nodes that are already running similar agents (data locality)
	if len(node.RunningAgents) > 0 {
		score += 10
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, nil
}

// FilterNodes filters nodes that can run the agent.
func (b *BestFitAlgorithm) FilterNodes(ctx context.Context, nodes []NodeState, agent AgentSpec) ([]NodeState, error) {
	var candidates []NodeState

	for _, node := range nodes {
		if b.CheckResources(node, agent) {
			candidates = append(candidates, node)
		}
	}

	return candidates, nil
}

// SelectNode selects the best node from the candidates.
func (b *BestFitAlgorithm) SelectNode(ctx context.Context, candidates []NodeState, scores map[string]int) (*NodeState, error) {
	if len(candidates) == 0 {
		return nil, ErrNoAvailableNodes
	}

	var bestNode *NodeState
	bestScore := -1

	for i := range candidates {
		score := scores[candidates[i].NodeID]
		if score > bestScore {
			bestScore = score
			bestNode = &candidates[i]
		}
	}

	return bestNode, nil
}

// CostAwareAlgorithm implements cost-aware scheduling.
// It selects the most cost-effective node.
type CostAwareAlgorithm struct {
	BaseAlgorithm
}

// NewCostAwareAlgorithm creates a new cost-aware algorithm.
func NewCostAwareAlgorithm() *CostAwareAlgorithm {
	return &CostAwareAlgorithm{
		BaseAlgorithm: BaseAlgorithm{
			config: AlgorithmConfig{
				Type:    AlgorithmCostAware,
				Weights: DefaultScoringWeights(),
			},
		},
	}
}

// Name returns the algorithm name.
func (c *CostAwareAlgorithm) Name() string {
	return "cost_aware"
}

// ScoreNode scores a node based on cost efficiency.
func (c *CostAwareAlgorithm) ScoreNode(ctx context.Context, node NodeState, agent AgentSpec) (int, error) {
	if !c.CheckResources(node, agent) {
		return 0, nil
	}

	// Base score
	score := 50

	// Get node cost (default to medium cost if not set)
	nodeCost := node.CostPerHour
	if nodeCost == 0 {
		nodeCost = 1.0 // Default cost
	}

	// Score inversely proportional to cost
	// Lower cost = higher score
	if nodeCost <= 0.5 {
		score += 40 // Very cheap
	} else if nodeCost <= 1.0 {
		score += 25 // Cheap
	} else if nodeCost <= 2.0 {
		score += 10 // Moderate
	} else {
		score -= 20 // Expensive
	}

	// Bonus for spot/preemptible instances
	if node.IsSpot {
		score += 15
	}

	// Consider resource efficiency
	// Prefer nodes that will have good utilization
	projectedCPUUtil := float64(node.CPUUsed+agent.CPURequest) / float64(node.CPUCores)
	if projectedCPUUtil > 0.6 && projectedCPUUtil < 0.9 {
		score += 10 // Good utilization range
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, nil
}

// FilterNodes filters nodes that can run the agent.
func (c *CostAwareAlgorithm) FilterNodes(ctx context.Context, nodes []NodeState, agent AgentSpec) ([]NodeState, error) {
	var candidates []NodeState

	for _, node := range nodes {
		if c.CheckResources(node, agent) {
			candidates = append(candidates, node)
		}
	}

	return candidates, nil
}

// SelectNode selects the best node from the candidates.
func (c *CostAwareAlgorithm) SelectNode(ctx context.Context, candidates []NodeState, scores map[string]int) (*NodeState, error) {
	if len(candidates) == 0 {
		return nil, ErrNoAvailableNodes
	}

	var bestNode *NodeState
	bestScore := -1

	for i := range candidates {
		score := scores[candidates[i].NodeID]
		if score > bestScore {
			bestScore = score
			bestNode = &candidates[i]
		}
	}

	return bestNode, nil
}

// LeastMigrationAlgorithm minimizes agent migration.
type LeastMigrationAlgorithm struct {
	BaseAlgorithm
}

// NewLeastMigrationAlgorithm creates a new least-migration algorithm.
func NewLeastMigrationAlgorithm() *LeastMigrationAlgorithm {
	return &LeastMigrationAlgorithm{
		BaseAlgorithm: BaseAlgorithm{
			config: AlgorithmConfig{
				Type:    AlgorithmLeastMigration,
				Weights: DefaultScoringWeights(),
			},
		},
	}
}

// Name returns the algorithm name.
func (l *LeastMigrationAlgorithm) Name() string {
	return "least_migration"
}

// ScoreNode scores a node, strongly favoring nodes already running the agent.
func (l *LeastMigrationAlgorithm) ScoreNode(ctx context.Context, node NodeState, agent AgentSpec) (int, error) {
	if !l.CheckResources(node, agent) {
		return 0, nil
	}

	score := 50 // Base score

	// Strong bonus if agent is already running on this node
	for _, runningAgent := range node.Agents {
		if runningAgent.ID == agent.ID {
			score = 100 // Maximum score for current node
			return score, nil
		}
	}

	// Prefer nodes with low churn (stable nodes)
	if node.MigrationsIn < 5 && node.MigrationsOut < 5 {
		score += 15
	}

	// Prefer nodes with lower load (room for growth)
	loadScore := 100 - int(l.CalculateUtilization(node)*100)
	score += loadScore / 4

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, nil
}

// FilterNodes filters nodes that can run the agent.
func (l *LeastMigrationAlgorithm) FilterNodes(ctx context.Context, nodes []NodeState, agent AgentSpec) ([]NodeState, error) {
	var candidates []NodeState

	for _, node := range nodes {
		if l.CheckResources(node, agent) {
			candidates = append(candidates, node)
		}
	}

	return candidates, nil
}

// SelectNode selects the best node from the candidates.
func (l *LeastMigrationAlgorithm) SelectNode(ctx context.Context, candidates []NodeState, scores map[string]int) (*NodeState, error) {
	if len(candidates) == 0 {
		return nil, ErrNoAvailableNodes
	}

	var bestNode *NodeState
	bestScore := -1

	for i := range candidates {
		score := scores[candidates[i].NodeID]
		if score > bestScore {
			bestScore = score
			bestNode = &candidates[i]
		}
	}

	return bestNode, nil
}

// NodeState extensions for migration tracking.
type MigrationTracking struct {
	MigrationsIn  int `json:"migrations_in"`
	MigrationsOut int `json:"migrations_out"`
}