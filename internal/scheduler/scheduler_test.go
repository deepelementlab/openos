package scheduler

import (
	"testing"
	"time"
)

func TestTaskRequest_Fields(t *testing.T) {
	req := TaskRequest{
		TaskID:        "t1",
		TaskType:      "compute",
		TaskName:      "my-task",
		CPURequest:    2.0,
		MemoryRequest: 4 * GiB,
		DiskRequest:   10 * GiB,
		GPURequest:    1,
		Priority:      5,
		Affinity:      []string{"zone=a"},
		AntiAffinity:  []string{"gpu=false"},
	}

	if req.TaskID != "t1" {
		t.Errorf("expected TaskID=t1, got %s", req.TaskID)
	}
	if req.TaskType != "compute" {
		t.Errorf("expected TaskType=compute, got %s", req.TaskType)
	}
	if req.TaskName != "my-task" {
		t.Errorf("expected TaskName=my-task, got %s", req.TaskName)
	}
	if req.CPURequest != 2.0 {
		t.Errorf("expected CPURequest=2.0, got %f", req.CPURequest)
	}
	if req.MemoryRequest != 4*GiB {
		t.Errorf("expected MemoryRequest=%d, got %d", 4*GiB, req.MemoryRequest)
	}
	if req.DiskRequest != 10*GiB {
		t.Errorf("expected DiskRequest=%d, got %d", 10*GiB, req.DiskRequest)
	}
	if req.GPURequest != 1 {
		t.Errorf("expected GPURequest=1, got %d", req.GPURequest)
	}
	if req.Priority != 5 {
		t.Errorf("expected Priority=5, got %d", req.Priority)
	}
	if len(req.Affinity) != 1 || req.Affinity[0] != "zone=a" {
		t.Errorf("expected Affinity=[zone=a], got %v", req.Affinity)
	}
	if len(req.AntiAffinity) != 1 || req.AntiAffinity[0] != "gpu=false" {
		t.Errorf("expected AntiAffinity=[gpu=false], got %v", req.AntiAffinity)
	}
}

func TestScheduleResult_Fields(t *testing.T) {
	now := time.Now()
	res := ScheduleResult{
		Success:     true,
		NodeID:      "n1",
		TaskID:      "t1",
		Reason:      "scheduled",
		Message:     "scheduled on n1",
		Score:       0.95,
		ScheduledAt: now,
	}

	if !res.Success {
		t.Error("expected Success=true")
	}
	if res.NodeID != "n1" {
		t.Errorf("expected NodeID=n1, got %s", res.NodeID)
	}
	if res.TaskID != "t1" {
		t.Errorf("expected TaskID=t1, got %s", res.TaskID)
	}
	if res.Reason != "scheduled" {
		t.Errorf("expected Reason=scheduled, got %s", res.Reason)
	}
	if res.Score != 0.95 {
		t.Errorf("expected Score=0.95, got %f", res.Score)
	}
	if !res.ScheduledAt.Equal(now) {
		t.Errorf("expected ScheduledAt=%v, got %v", now, res.ScheduledAt)
	}
}

func TestNodeState_Fields(t *testing.T) {
	ns := NodeState{
		NodeID:        "node-1",
		NodeName:      "worker-1",
		Address:       "10.0.0.1:8080",
		CPUUsage:      45.5,
		MemoryUsage:   60.0,
		DiskUsage:     30.0,
		CPUCores:      8,
		MemoryBytes:   32 * GiB,
		DiskBytes:     500 * GiB,
		GPUCount:      2,
		AgentCount:    3,
		Health:        "healthy",
		LastUpdate:    time.Now(),
		CPUUsed:       4,
		MemoryUsed:    16 * GiB,
		DiskUsed:      100 * GiB,
		Labels:        map[string]string{"zone": "us-east-1a"},
		CostPerHour:   0.50,
		IsSpot:        true,
		MigrationsIn:  1,
		MigrationsOut: 0,
	}

	if ns.NodeID != "node-1" {
		t.Errorf("expected NodeID=node-1, got %s", ns.NodeID)
	}
	if ns.CPUCores != 8 {
		t.Errorf("expected CPUCores=8, got %d", ns.CPUCores)
	}
	if ns.GPUCount != 2 {
		t.Errorf("expected GPUCount=2, got %d", ns.GPUCount)
	}
	if ns.Health != "healthy" {
		t.Errorf("expected Health=healthy, got %s", ns.Health)
	}
	if ns.CostPerHour != 0.50 {
		t.Errorf("expected CostPerHour=0.50, got %f", ns.CostPerHour)
	}
	if !ns.IsSpot {
		t.Error("expected IsSpot=true")
	}
	if ns.Labels["zone"] != "us-east-1a" {
		t.Errorf("expected zone=us-east-1a, got %s", ns.Labels["zone"])
	}
}

func TestAgentInfo_Fields(t *testing.T) {
	ai := AgentInfo{ID: "agent-123"}
	if ai.ID != "agent-123" {
		t.Errorf("expected ID=agent-123, got %s", ai.ID)
	}
}

func TestAgentSpec_Fields(t *testing.T) {
	spec := AgentSpec{
		ID:            "agent-1",
		Name:          "test-agent",
		CPURequest:    2,
		MemoryRequest: 4 * GiB,
		DiskRequest:   10 * GiB,
		Labels:        map[string]string{"tier": "frontend"},
	}

	if spec.ID != "agent-1" {
		t.Errorf("expected ID=agent-1, got %s", spec.ID)
	}
	if spec.Name != "test-agent" {
		t.Errorf("expected Name=test-agent, got %s", spec.Name)
	}
	if spec.CPURequest != 2 {
		t.Errorf("expected CPURequest=2, got %d", spec.CPURequest)
	}
	if spec.MemoryRequest != 4*GiB {
		t.Errorf("expected MemoryRequest=%d, got %d", 4*GiB, spec.MemoryRequest)
	}
}

func TestNodeScore_Fields(t *testing.T) {
	ns := NodeScore{
		NodeID: "n1",
		Score:  85.5,
		Details: map[string]float64{
			"cpu": 80.0,
			"mem": 91.0,
		},
	}

	if ns.NodeID != "n1" {
		t.Errorf("expected NodeID=n1, got %s", ns.NodeID)
	}
	if ns.Score != 85.5 {
		t.Errorf("expected Score=85.5, got %f", ns.Score)
	}
	if ns.Details["cpu"] != 80.0 {
		t.Errorf("expected cpu=80.0, got %f", ns.Details["cpu"])
	}
}

func TestSchedulerMetrics_Fields(t *testing.T) {
	m := SchedulerMetrics{
		TotalScheduled: 100,
		TotalFailed:    5,
		NodeCount:      3,
		QueueSize:      10,
		AlgorithmStats: map[string]int64{"strategy": 12},
	}

	if m.TotalScheduled != 100 {
		t.Errorf("expected TotalScheduled=100, got %d", m.TotalScheduled)
	}
	if m.TotalFailed != 5 {
		t.Errorf("expected TotalFailed=5, got %d", m.TotalFailed)
	}
	if m.NodeCount != 3 {
		t.Errorf("expected NodeCount=3, got %d", m.NodeCount)
	}
	if m.QueueSize != 10 {
		t.Errorf("expected QueueSize=10, got %d", m.QueueSize)
	}
}

func TestRescheduleReason_Constants(t *testing.T) {
	if RescheduleReasonNodeFailure != "node_failure" {
		t.Errorf("expected node_failure, got %s", RescheduleReasonNodeFailure)
	}
	if RescheduleReasonResourceShort != "resource_shortage" {
		t.Errorf("expected resource_shortage, got %s", RescheduleReasonResourceShort)
	}
	if RescheduleReasonOptimization != "optimization" {
		t.Errorf("expected optimization, got %s", RescheduleReasonOptimization)
	}
	if RescheduleReasonManual != "manual" {
		t.Errorf("expected manual, got %s", RescheduleReasonManual)
	}
}

func TestErrNoAvailableNodes(t *testing.T) {
	if ErrNoAvailableNodes == nil {
		t.Fatal("ErrNoAvailableNodes should not be nil")
	}
	if ErrNoAvailableNodes.Error() != "no available nodes for scheduling" {
		t.Errorf("unexpected error message: %s", ErrNoAvailableNodes.Error())
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Algorithm != "resource_aware" {
		t.Errorf("expected algorithm=resource_aware, got %s", cfg.Algorithm)
	}
	if cfg.MaxConcurrentAgents != 100 {
		t.Errorf("expected MaxConcurrentAgents=100, got %d", cfg.MaxConcurrentAgents)
	}
	if cfg.MaxAgentsPerHost != 10 {
		t.Errorf("expected MaxAgentsPerHost=10, got %d", cfg.MaxAgentsPerHost)
	}
	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("expected CheckInterval=30s, got %v", cfg.CheckInterval)
	}
	if !cfg.AutoScale {
		t.Error("expected AutoScale=true")
	}
	if cfg.ScaleUpThreshold != 80 {
		t.Errorf("expected ScaleUpThreshold=80, got %d", cfg.ScaleUpThreshold)
	}
	if cfg.ScaleDownThreshold != 20 {
		t.Errorf("expected ScaleDownThreshold=20, got %d", cfg.ScaleDownThreshold)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestConfig_Validate_EmptyAlgorithm(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Algorithm = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty algorithm")
	}
}

func TestConfig_Validate_NegativeMaxConcurrentAgents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrentAgents = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative max_concurrent_agents")
	}
}

func TestConfig_Validate_ZeroMaxConcurrentAgents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrentAgents = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for zero max_concurrent_agents")
	}
}

func TestConfig_Validate_NegativeMaxAgentsPerHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAgentsPerHost = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative max_agents_per_host")
	}
}

func TestConfig_Validate_ZeroCheckInterval(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CheckInterval = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for zero check_interval")
	}
}

func TestConfig_Validate_ScaleDownGreaterOrEqualScaleUp(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScaleDownThreshold = 80
	cfg.ScaleUpThreshold = 80
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when scale_down >= scale_up")
	}

	cfg.ScaleDownThreshold = 90
	cfg.ScaleUpThreshold = 80
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when scale_down > scale_up")
	}
}

func TestDefaultScheduler_InterfaceCompliance(t *testing.T) {
	var _ Scheduler = NewDefaultScheduler()
}
