package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/config"
)

// Scheduler is the core scheduler interface.
type Scheduler interface {
	Initialize(ctx context.Context, cfg *config.Config) error
	Schedule(ctx context.Context, req TaskRequest) (*ScheduleResult, error)
	ScheduleBatch(ctx context.Context, reqs []TaskRequest) ([]ScheduleResult, error)
	Reschedule(ctx context.Context, taskID string, reason RescheduleReason) (*ScheduleResult, error)
	CancelTask(ctx context.Context, taskID string) error
	AddNode(ctx context.Context, node NodeState) error
	RemoveNode(ctx context.Context, nodeID string) error
	UpdateNode(ctx context.Context, node NodeState) error
	ListNodes(ctx context.Context) ([]NodeState, error)
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context) (*SchedulerMetrics, error)
}

// NodeState describes a scheduling node's resource state.
type NodeState struct {
	NodeID      string    `json:"node_id"`
	NodeName    string    `json:"node_name"`
	Address     string    `json:"address"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	CPUCores    int       `json:"cpu_cores"`
	MemoryBytes int64     `json:"memory_bytes"`
	DiskBytes   int64     `json:"disk_bytes"`
	AgentCount  int       `json:"agent_count"`
	Health      string    `json:"health"`
	LastUpdate  time.Time `json:"last_update"`
}

// TaskRequest describes what needs to be scheduled.
type TaskRequest struct {
	TaskID        string   `json:"task_id"`
	TaskType      string   `json:"task_type"`
	TaskName      string   `json:"task_name"`
	CPURequest    float64  `json:"cpu_request"`
	MemoryRequest int64    `json:"memory_request"`
	DiskRequest   int64    `json:"disk_request"`
	Priority      int      `json:"priority"`
	Affinity      []string `json:"affinity,omitempty"`
	AntiAffinity  []string `json:"anti_affinity,omitempty"`
}

// ScheduleResult holds the outcome of a scheduling decision.
type ScheduleResult struct {
	Success     bool      `json:"success"`
	NodeID      string    `json:"node_id,omitempty"`
	TaskID      string    `json:"task_id,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
	Score       float64   `json:"score,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// RescheduleReason enumerates reasons for rescheduling.
type RescheduleReason string

const (
	RescheduleReasonNodeFailure   RescheduleReason = "node_failure"
	RescheduleReasonResourceShort RescheduleReason = "resource_shortage"
	RescheduleReasonOptimization  RescheduleReason = "optimization"
	RescheduleReasonManual        RescheduleReason = "manual"
)

// Algorithm abstracts a pluggable scoring/filtering algorithm.
type Algorithm interface {
	Name() string
	FilterNodes(ctx context.Context, nodes []NodeState, req TaskRequest) ([]NodeState, error)
	ScoreNodes(ctx context.Context, nodes []NodeState, req TaskRequest) ([]NodeScore, error)
	SelectNode(ctx context.Context, scores []NodeScore) (*NodeState, error)
}

// NodeScore pairs a node ID with its computed score.
type NodeScore struct {
	NodeID  string             `json:"node_id"`
	Score   float64            `json:"score"`
	Details map[string]float64 `json:"details,omitempty"`
}

// SchedulerMetrics exposes operational counters.
type SchedulerMetrics struct {
	TotalScheduled int64            `json:"total_scheduled"`
	TotalFailed    int64            `json:"total_failed"`
	NodeCount      int              `json:"node_count"`
	QueueSize      int              `json:"queue_size"`
	AlgorithmStats map[string]int64 `json:"algorithm_stats,omitempty"`
}

// Config holds scheduler-specific tunables.
type Config struct {
	Algorithm           string        `json:"algorithm" mapstructure:"algorithm"`
	MaxConcurrentAgents int           `json:"max_concurrent_agents" mapstructure:"max_concurrent_agents"`
	MaxAgentsPerHost    int           `json:"max_agents_per_host" mapstructure:"max_agents_per_host"`
	CheckInterval       time.Duration `json:"check_interval" mapstructure:"check_interval"`
	AutoScale           bool          `json:"auto_scale" mapstructure:"auto_scale"`
	ScaleUpThreshold    int           `json:"scale_up_threshold" mapstructure:"scale_up_threshold"`
	ScaleDownThreshold  int           `json:"scale_down_threshold" mapstructure:"scale_down_threshold"`
}

func DefaultConfig() *Config {
	return &Config{
		Algorithm:           "resource_aware",
		MaxConcurrentAgents: 100,
		MaxAgentsPerHost:    10,
		CheckInterval:       30 * time.Second,
		AutoScale:           true,
		ScaleUpThreshold:    80,
		ScaleDownThreshold:  20,
	}
}

func (c *Config) Validate() error {
	if c.Algorithm == "" {
		return fmt.Errorf("algorithm is required")
	}
	if c.MaxConcurrentAgents <= 0 {
		return fmt.Errorf("max_concurrent_agents must be positive")
	}
	if c.MaxAgentsPerHost <= 0 {
		return fmt.Errorf("max_agents_per_host must be positive")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("check_interval must be positive")
	}
	if c.ScaleDownThreshold >= c.ScaleUpThreshold {
		return fmt.Errorf("scale_down_threshold must be less than scale_up_threshold")
	}
	return nil
}
