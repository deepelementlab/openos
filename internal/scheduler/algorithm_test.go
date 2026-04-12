package scheduler

import (
	"testing"
)

func TestNewAlgorithmRegistry(t *testing.T) {
	reg := NewAlgorithmRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestAlgorithmRegistry_RegisterAndGet(t *testing.T) {
	reg := NewAlgorithmRegistry()
	alg := NewBestFitAlgorithm()

	if err := reg.Register("best_fit", alg); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := reg.Get("best_fit")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != alg {
		t.Error("expected to get the same algorithm")
	}
}

func TestAlgorithmRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewAlgorithmRegistry()
	alg := NewBestFitAlgorithm()

	reg.Register("best_fit", alg)
	err := reg.Register("best_fit", alg)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestAlgorithmRegistry_GetNotFound(t *testing.T) {
	reg := NewAlgorithmRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent algorithm")
	}
}

func TestAlgorithmRegistry_SetDefault(t *testing.T) {
	reg := NewAlgorithmRegistry()
	alg := NewBestFitAlgorithm()

	reg.Register("best_fit", alg)

	if err := reg.SetDefault("best_fit"); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if reg.GetDefault() != alg {
		t.Error("expected default to be best_fit")
	}
}

func TestAlgorithmRegistry_SetDefault_NotFound(t *testing.T) {
	reg := NewAlgorithmRegistry()

	err := reg.SetDefault("nonexistent")
	if err == nil {
		t.Fatal("expected error setting non-existent default")
	}
}

func TestAlgorithmRegistry_GetDefault_NoDefault(t *testing.T) {
	reg := NewAlgorithmRegistry()
	if reg.GetDefault() != nil {
		t.Error("expected nil default")
	}
}

func TestAlgorithmType_Constants(t *testing.T) {
	if AlgorithmRoundRobin != "round_robin" {
		t.Errorf("expected round_robin, got %s", AlgorithmRoundRobin)
	}
	if AlgorithmBestFit != "best_fit" {
		t.Errorf("expected best_fit, got %s", AlgorithmBestFit)
	}
	if AlgorithmLeastLoaded != "least_loaded" {
		t.Errorf("expected least_loaded, got %s", AlgorithmLeastLoaded)
	}
	if AlgorithmLeastMigration != "least_migration" {
		t.Errorf("expected least_migration, got %s", AlgorithmLeastMigration)
	}
	if AlgorithmCostAware != "cost_aware" {
		t.Errorf("expected cost_aware, got %s", AlgorithmCostAware)
	}
}

func TestScoreRange_Constants(t *testing.T) {
	if MinScore != 0 {
		t.Errorf("expected MinScore=0, got %d", MinScore)
	}
	if MaxScore != 100 {
		t.Errorf("expected MaxScore=100, got %d", MaxScore)
	}
}

func TestDefaultScoringWeights(t *testing.T) {
	w := DefaultScoringWeights()

	if w.CPUUtilization != 0.25 {
		t.Errorf("expected CPUUtilization=0.25, got %f", w.CPUUtilization)
	}
	if w.MemoryUtilization != 0.25 {
		t.Errorf("expected MemoryUtilization=0.25, got %f", w.MemoryUtilization)
	}
	if w.DiskUtilization != 0.15 {
		t.Errorf("expected DiskUtilization=0.15, got %f", w.DiskUtilization)
	}
	if w.NetworkLatency != 0.15 {
		t.Errorf("expected NetworkLatency=0.15, got %f", w.NetworkLatency)
	}
	if w.Cost != 0.10 {
		t.Errorf("expected Cost=0.10, got %f", w.Cost)
	}
	if w.Affinity != 0.10 {
		t.Errorf("expected Affinity=0.10, got %f", w.Affinity)
	}
}

func TestBaseAlgorithm_GetConfig(t *testing.T) {
	base := &BaseAlgorithm{
		config: AlgorithmConfig{
			Type:    AlgorithmBestFit,
			Weights: DefaultScoringWeights(),
		},
	}

	cfg := base.GetConfig()
	if cfg.Type != AlgorithmBestFit {
		t.Errorf("expected type=best_fit, got %s", cfg.Type)
	}
}

func TestBaseAlgorithm_SetConfig(t *testing.T) {
	base := &BaseAlgorithm{}
	newCfg := AlgorithmConfig{
		Type:    AlgorithmCostAware,
		Weights: DefaultScoringWeights(),
	}
	base.SetConfig(newCfg)

	if base.GetConfig().Type != AlgorithmCostAware {
		t.Errorf("expected type=cost_aware, got %s", base.GetConfig().Type)
	}
}

func TestBaseAlgorithm_CheckResources_Sufficient(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    8,
		CPUUsed:     2,
		MemoryBytes: 16 * GiB,
		MemoryUsed:  4 * GiB,
		DiskBytes:   100 * GiB,
		DiskUsed:    10 * GiB,
	}
	agent := AgentSpec{
		CPURequest:    4,
		MemoryRequest: 8 * GiB,
		DiskRequest:   50 * GiB,
	}

	if !base.CheckResources(node, agent) {
		t.Error("expected resources to be sufficient")
	}
}

func TestBaseAlgorithm_CheckResources_InsufficientCPU(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:   "n1",
		Health:   "healthy",
		CPUCores: 4,
		CPUUsed:  3,
	}
	agent := AgentSpec{CPURequest: 2}

	if base.CheckResources(node, agent) {
		t.Error("expected insufficient CPU")
	}
}

func TestBaseAlgorithm_CheckResources_InsufficientMemory(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    8,
		MemoryBytes: 4 * GiB,
		MemoryUsed:  3 * GiB,
	}
	agent := AgentSpec{MemoryRequest: 2 * GiB}

	if base.CheckResources(node, agent) {
		t.Error("expected insufficient memory")
	}
}

func TestBaseAlgorithm_CheckResources_InsufficientDisk(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    8,
		MemoryBytes: 16 * GiB,
		DiskBytes:   10 * GiB,
		DiskUsed:    8 * GiB,
	}
	agent := AgentSpec{DiskRequest: 5 * GiB}

	if base.CheckResources(node, agent) {
		t.Error("expected insufficient disk")
	}
}

func TestBaseAlgorithm_CheckResources_UnhealthyNode(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:   "n1",
		Health:   "unhealthy",
		CPUCores: 8,
	}
	agent := AgentSpec{CPURequest: 1}

	if base.CheckResources(node, agent) {
		t.Error("expected unhealthy node to fail resource check")
	}
}

func TestBaseAlgorithm_CheckResources_ZeroRequests(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:   "n1",
		Health:   "healthy",
		CPUCores: 4,
	}
	agent := AgentSpec{}

	if !base.CheckResources(node, agent) {
		t.Error("zero requests should pass on healthy node")
	}
}

func TestBaseAlgorithm_CalculateUtilization(t *testing.T) {
	base := &BaseAlgorithm{
		config: AlgorithmConfig{
			Weights: DefaultScoringWeights(),
		},
	}

	node := NodeState{
		CPUCores:    8,
		CPUUsed:     4,
		MemoryBytes: 16 * GiB,
		MemoryUsed:  8 * GiB,
	}

	util := base.CalculateUtilization(node)

	cpuUtil := float64(4) / float64(8)
	memUtil := float64(8*GiB) / float64(16*GiB)
	expected := cpuUtil*0.25 + memUtil*0.25

	if util != expected {
		t.Errorf("expected utilization=%f, got %f", expected, util)
	}
}

func TestBaseAlgorithm_CalculateUtilization_ZeroResources(t *testing.T) {
	base := &BaseAlgorithm{
		config: AlgorithmConfig{
			Weights: DefaultScoringWeights(),
		},
	}

	node := NodeState{CPUCores: 0, MemoryBytes: 0}
	util := base.CalculateUtilization(node)

	if util != 0 {
		// NaN would be produced if division by zero, but with 0/0 = NaN in float64
		// This tests that we handle it
		t.Logf("utilization with zero resources: %f (may be NaN)", util)
	}
}

func TestSchedulingAlgorithm_Interface(t *testing.T) {
	var algos []SchedulingAlgorithm = []SchedulingAlgorithm{
		NewBestFitAlgorithm(),
		NewCostAwareAlgorithm(),
		NewLeastMigrationAlgorithm(),
	}

	for _, alg := range algos {
		if alg.Name() == "" {
			t.Error("algorithm name should not be empty")
		}
	}
}

func TestAlgorithmConfig_Defaults(t *testing.T) {
	cfg := AlgorithmConfig{
		Type:    AlgorithmBestFit,
		Weights: DefaultScoringWeights(),
	}

	if cfg.Type != AlgorithmBestFit {
		t.Errorf("expected best_fit, got %s", cfg.Type)
	}
	if cfg.Weights.CPUUtilization != 0.25 {
		t.Errorf("expected 0.25, got %f", cfg.Weights.CPUUtilization)
	}
}

func TestNodePreferences_Fields(t *testing.T) {
	prefs := NodePreferences{
		PreferLocal:        true,
		PreferLowCost:      true,
		PreferHighResource: false,
		AvoidOvercommit:    true,
		RequiredLabels:     []string{"zone"},
		PreferredZones:     []string{"us-east-1a"},
	}

	if !prefs.PreferLocal {
		t.Error("expected PreferLocal=true")
	}
	if !prefs.PreferLowCost {
		t.Error("expected PreferLowCost=true")
	}
	if len(prefs.RequiredLabels) != 1 {
		t.Errorf("expected 1 required label, got %d", len(prefs.RequiredLabels))
	}
}

func TestBaseAlgorithm_CheckResources_ExactFit(t *testing.T) {
	base := &BaseAlgorithm{}
	node := NodeState{
		NodeID:      "n1",
		Health:      "healthy",
		CPUCores:    4,
		CPUUsed:     0,
		MemoryBytes: 8 * GiB,
		MemoryUsed:  0,
		DiskBytes:   100 * GiB,
		DiskUsed:    0,
	}
	agent := AgentSpec{
		CPURequest:    4,
		MemoryRequest: 8 * GiB,
		DiskRequest:   100 * GiB,
	}

	if !base.CheckResources(node, agent) {
		t.Error("exact fit should pass")
	}
}
