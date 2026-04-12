package models

import (
	"time"
)

// AgentStatus represents the lifecycle status of an agent
type AgentStatus string

const (
	AgentStatusPending  AgentStatus = "pending"
	AgentStatusCreating AgentStatus = "creating"
	AgentStatusRunning  AgentStatus = "running"
	AgentStatusStopping AgentStatus = "stopping"
	AgentStatusStopped  AgentStatus = "stopped"
	AgentStatusError    AgentStatus = "error"
)

// Agent represents the API model for an agent
type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	Runtime     string                 `json:"runtime,omitempty"`
	Status      AgentStatus            `json:"status"`
	Resources   map[string]string      `json:"resources,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// CreateAgentRequest represents the request body for creating an agent
type CreateAgentRequest struct {
	Name        string            `json:"name" binding:"required,max=63"`
	Image       string            `json:"image" binding:"required"`
	Runtime     string            `json:"runtime,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// UpdateAgentRequest represents the request body for updating an agent
type UpdateAgentRequest struct {
	Name        string            `json:"name,omitempty" binding:"omitempty,max=63"`
	Image       string            `json:"image,omitempty"`
	Runtime     string            `json:"runtime,omitempty"`
	Status      AgentStatus       `json:"status,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// AgentListFilter represents filtering options for listing agents
type AgentListFilter struct {
	Status string `form:"status"`
	Name   string `form:"name"`
	Label  string `form:"label"`
}

// AgentListResponse represents the response for listing agents
type AgentListResponse struct {
	Agents []Agent `json:"agents"`
	Total  int     `json:"total"`
}

// Validate validates the create agent request
func (r *CreateAgentRequest) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(r.Name) > 63 {
		return &ValidationError{Field: "name", Message: "name must be 63 characters or less"}
	}
	if r.Image == "" {
		return &ValidationError{Field: "image", Message: "image is required"}
	}
	return nil
}

// Validate validates the update agent request
func (r *UpdateAgentRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 63 {
		return &ValidationError{Field: "name", Message: "name must be 63 characters or less"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// AgentActionRequest represents a request for agent actions (start, stop, restart)
type AgentActionRequest struct {
	Force bool `json:"force,omitempty"` // Force the action even if state transition is invalid
}

// AgentActionResponse represents the response for agent actions
type AgentActionResponse struct {
	AgentID    string      `json:"agentId"`
	Action     string      `json:"action"`
	OldStatus  AgentStatus `json:"oldStatus"`
	NewStatus  AgentStatus `json:"newStatus"`
	StartedAt  time.Time   `json:"startedAt"`
	Message    string      `json:"message"`
}

// AgentMetrics represents runtime metrics for an agent
type AgentMetrics struct {
	AgentID        string  `json:"agentId"`
	CPUUsage       float64 `json:"cpuUsage"`       // CPU usage percentage
	MemoryUsage    float64 `json:"memoryUsage"`    // Memory usage in bytes
	NetworkRxBytes int64   `json:"networkRxBytes"` // Network received bytes
	NetworkTxBytes int64   `json:"networkTxBytes"` // Network transmitted bytes
	DiskReadBytes  int64   `json:"diskReadBytes"`  // Disk read bytes
	DiskWriteBytes int64   `json:"diskWriteBytes"` // Disk write bytes
	Uptime         int64   `json:"uptime"`         // Uptime in seconds
	RestartCount   int     `json:"restartCount"`   // Number of restarts
}

// AgentEvent represents an event related to an agent
type AgentEvent struct {
	EventID   string      `json:"eventId"`
	AgentID   string      `json:"agentId"`
	EventType string      `json:"eventType"` // created, started, stopped, restarted, error
	Timestamp time.Time   `json:"timestamp"`
	Status    AgentStatus `json:"status"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
}

// ResourceRequirements represents resource requirements for an agent
type ResourceRequirements struct {
	CPU     string `json:"cpu"`     // CPU cores or millicores (e.g., "500m", "2")
	Memory  string `json:"memory"`  // Memory (e.g., "512Mi", "1Gi")
	Storage string `json:"storage"` // Storage (e.g., "10Gi")
	GPU     string `json:"gpu"`     // GPU count (e.g., "1", "2")
}

// NetworkPolicy represents network policy for an agent
type NetworkPolicy struct {
	AllowInbound  bool     `json:"allowInbound"`
	AllowOutbound bool     `json:"allowOutbound"`
	InboundPorts  []int    `json:"inboundPorts,omitempty"`
	OutboundHosts []string `json:"outboundHosts,omitempty"`
}

// SecurityContext represents security settings for an agent
type SecurityContext struct {
	RunAsUser         int    `json:"runAsUser,omitempty"`
	RunAsGroup        int    `json:"runAsGroup,omitempty"`
	ReadOnlyRootFS    bool   `json:"readOnlyRootFS"`
	AllowPrivilegeEsc bool   `json:"allowPrivilegeEscalation"`
	SandboxType       string `json:"sandboxType"` // "gvisor", "kata", "containerd"
	SeccompProfile    string `json:"seccompProfile,omitempty"`
}
