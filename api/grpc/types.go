package grpc

import "time"

type Empty struct{}

type AgentInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Runtime     string            `json:"runtime"`
	Status      string            `json:"status"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CreateAgentRequest struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Runtime     string            `json:"runtime"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	TenantID    string            `json:"tenant_id,omitempty"`
}

type GetAgentRequest struct {
	AgentID string `json:"agent_id"`
}

type ListAgentsRequest struct {
	Page     int               `json:"page,omitempty"`
	PageSize int               `json:"page_size,omitempty"`
	Status   string            `json:"status,omitempty"`
	TenantID string            `json:"tenant_id,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type ListAgentsResponse struct {
	Agents []*AgentInfo `json:"agents"`
	Total  int          `json:"total"`
	Page   int          `json:"page"`
	Pages  int          `json:"pages"`
}

type UpdateAgentRequest struct {
	AgentID     string            `json:"agent_id"`
	Name        string            `json:"name,omitempty"`
	Image       string            `json:"image,omitempty"`
	Runtime     string            `json:"runtime,omitempty"`
	Status      string            `json:"status,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type DeleteAgentRequest struct {
	AgentID string `json:"agent_id"`
}

type ScheduleRequest struct {
	AgentID   string            `json:"agent_id"`
	Resources map[string]string `json:"resources,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Priority  int32             `json:"priority,omitempty"`
}

type ScheduleResponse struct {
	NodeID        string    `json:"node_id"`
	NodeName      string    `json:"node_name"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	EstimatedTime int32     `json:"estimated_start_time_seconds,omitempty"`
}

type NodeInfo struct {
	NodeID string            `json:"node_id"`
	Name   string            `json:"name"`
	Status string            `json:"status"`
	CPU    string            `json:"cpu,omitempty"`
	Memory string            `json:"memory,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

type MetricsRequest struct {
	AgentID string `json:"agent_id"`
}

type MetricsResponse struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	RequestsTotal      int64   `json:"requests_total"`
	ErrorRate          float64 `json:"error_rate"`
	ResponseTimeP95    float64 `json:"response_time_p95"`
	ResponseTimeP99    float64 `json:"response_time_p99"`
	Uptime             int64   `json:"uptime"`
	RestartCount       int     `json:"restart_count"`
}

type HealthCheckRequest struct {
	Service string `json:"service,omitempty"`
}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type OperationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
