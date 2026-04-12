package scheduler

import (
	"context"
	"fmt"
)

// SchedulingAlgorithm defines the interface for scheduling algorithms.
type SchedulingAlgorithm interface {
	// Name returns the algorithm name.
	Name() string

	// ScoreNode scores a node for scheduling an agent.
	ScoreNode(ctx context.Context, node NodeState, agent AgentSpec) (int, error)

	// FilterNodes filters nodes that can run the agent.
	FilterNodes(ctx context.Context, nodes []NodeState, agent AgentSpec) ([]NodeState, error)

	// SelectNode selects the best node from the candidates.
	SelectNode(ctx context.Context, candidates []NodeState, scores map[string]int) (*NodeState, error)
}

// AlgorithmRegistry manages scheduling algorithms.
type AlgorithmRegistry struct {
	algorithms map[string]SchedulingAlgorithm
	defaultAlg SchedulingAlgorithm
}

// NewAlgorithmRegistry creates a new algorithm registry.
func NewAlgorithmRegistry() *AlgorithmRegistry {
	return &AlgorithmRegistry{
		algorithms: make(map[string]SchedulingAlgorithm),
	}
}

// Register registers a scheduling algorithm.
func (r *AlgorithmRegistry) Register(name string, alg SchedulingAlgorithm) error {
	if _, exists := r.algorithms[name]; exists {
		return fmt.Errorf("algorithm %s already registered", name)
	}
	r.algorithms[name] = alg
	return nil
}

// Get retrieves an algorithm by name.
func (r *AlgorithmRegistry) Get(name string) (SchedulingAlgorithm, error) {
	alg, exists := r.algorithms[name]
	if !exists {
		return nil, fmt.Errorf("algorithm %s not found", name)
	}
	return alg, nil
}

// GetDefault returns the default algorithm.
func (r *AlgorithmRegistry) GetDefault() SchedulingAlgorithm {
	return r.defaultAlg
}

// SetDefault sets the default algorithm.
func (r *AlgorithmRegistry) SetDefault(name string) error {
	alg, err := r.Get(name)
	if err != nil {
		return err
	}
	r.defaultAlg = alg
	return nil
}

// ScoreRange defines the scoring range.
const (
	MinScore = 0
	MaxScore = 100
)

// AlgorithmType defines algorithm types.
type AlgorithmType string

const (
	AlgorithmRoundRobin      AlgorithmType = "round_robin"
	AlgorithmBestFit         AlgorithmType = "best_fit"
	AlgorithmLeastLoaded     AlgorithmType = "least_loaded"
	AlgorithmLeastMigration  AlgorithmType = "least_migration"
	AlgorithmCostAware       AlgorithmType = "cost_aware"
)

// AlgorithmConfig provides configuration for scheduling algorithms.
type AlgorithmConfig struct {
	Type            AlgorithmType   `json:"type"`
	Weights         ScoringWeights  `json:"weights"`
	NodePreferences NodePreferences `json:"node_preferences"`
}

// ScoringWeights defines weights for different scoring factors.
type ScoringWeights struct {
	CPUUtilization    float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
	DiskUtilization   float64 `json:"disk_utilization"`
	NetworkLatency  float64 `json:"network_latency"`
	Cost            float64 `json:"cost"`
	Affinity          float64 `json:"affinity"`
}

// DefaultScoringWeights returns default scoring weights.
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		CPUUtilization:    0.25,
		MemoryUtilization: 0.25,
		DiskUtilization:   0.15,
		NetworkLatency:    0.15,
		Cost:              0.10,
		Affinity:          0.10,
	}
}

// NodePreferences defines node selection preferences.
type NodePreferences struct {
	PreferLocal       bool     `json:"prefer_local"`
	PreferLowCost     bool     `json:"prefer_low_cost"`
	PreferHighResource bool    `json:"prefer_high_resource"`
	AvoidOvercommit   bool     `json:"avoid_overcommit"`
	RequiredLabels    []string `json:"required_labels,omitempty"`
	PreferredZones    []string `json:"preferred_zones,omitempty"`
}

// BaseAlgorithm provides common functionality for algorithms.
type BaseAlgorithm struct {
	config AlgorithmConfig
}

// GetConfig returns the algorithm configuration.
func (b *BaseAlgorithm) GetConfig() AlgorithmConfig {
	return b.config
}

// SetConfig sets the algorithm configuration.
func (b *BaseAlgorithm) SetConfig(config AlgorithmConfig) {
	b.config = config
}

// CheckResources checks if a node has sufficient resources.
func (b *BaseAlgorithm) CheckResources(node NodeState, agent AgentSpec) bool {
	// Check CPU
	if agent.CPURequest > 0 {
		availableCPU := node.CPUCores - node.CPUUsed
		if agent.CPURequest > availableCPU {
			return false
		}
	}

	// Check memory
	if agent.MemoryRequest > 0 {
		availableMem := node.MemoryBytes - node.MemoryUsed
		if agent.MemoryRequest > availableMem {
			return false
		}
	}

	// Check disk
	if agent.DiskRequest > 0 {
		availableDisk := node.DiskBytes - node.DiskUsed
		if agent.DiskRequest > availableDisk {
			return false
		}
	}

	// Check if node is healthy
	if node.Health != "healthy" {
		return false
	}

	return true
}

// CalculateUtilization calculates resource utilization score (lower is better).
func (b *BaseAlgorithm) CalculateUtilization(node NodeState) float64 {
	cpuUtil := float64(node.CPUUsed) / float64(node.CPUCores)
	memUtil := float64(node.MemoryUsed) / float64(node.MemoryBytes)

	weights := b.config.Weights
	weightedUtil := cpuUtil*weights.CPUUtilization + memUtil*weights.MemoryUtilization

	return weightedUtil
}
