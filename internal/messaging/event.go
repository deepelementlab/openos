package messaging

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event represents the standard event structure for Agent OS messaging.
// This aligns with the event contract defined in tech-architecture.md.
type Event struct {
	// Unique identifier for the event
	ID string `json:"event_id"`

	// Event type (e.g., "agent.created", "agent.started")
	Type string `json:"event_type"`

	// Schema version for backward compatibility
	SchemaVersion string `json:"schema_version"`

	// Timestamp when the event occurred
	OccurredAt time.Time `json:"occurred_at"`

	// Distributed tracing ID
	TraceID string `json:"trace_id,omitempty"`

	// Agent ID associated with this event
	AgentID string `json:"agent_id,omitempty"`

	// Tenant ID for multi-tenancy
	TenantID string `json:"tenant_id,omitempty"`

	// Event payload
	Payload json.RawMessage `json:"payload"`

	// Additional metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with the given type and payload.
func NewEvent(eventType string, payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Event{
		ID:            uuid.New().String(),
		Type:          eventType,
		SchemaVersion: "1.0",
		OccurredAt:    time.Now().UTC(),
		Payload:       data,
		Metadata:      make(map[string]string),
	}, nil
}

// NewEventWithID creates a new event with a specific ID (for idempotency).
func NewEventWithID(eventID, eventType string, payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Event{
		ID:            eventID,
		Type:          eventType,
		SchemaVersion: "1.0",
		OccurredAt:    time.Now().UTC(),
		Payload:       data,
		Metadata:      make(map[string]string),
	}, nil
}

// SetTraceID sets the trace ID for distributed tracing.
func (e *Event) SetTraceID(traceID string) *Event {
	e.TraceID = traceID
	return e
}

// SetAgentID sets the agent ID.
func (e *Event) SetAgentID(agentID string) *Event {
	e.AgentID = agentID
	return e
}

// SetTenantID sets the tenant ID.
func (e *Event) SetTenantID(tenantID string) *Event {
	e.TenantID = tenantID
	return e
}

// SetMetadata sets a metadata value.
func (e *Event) SetMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// GetMetadata gets a metadata value.
func (e *Event) GetMetadata(key string) string {
	if e.Metadata == nil {
		return ""
	}
	return e.Metadata[key]
}

// PayloadObject deserializes the payload into an object.
func (e *Event) PayloadObject(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// ToJSON serializes the event to JSON.
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes an event from JSON.
func (e *Event) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}

// Validate validates the event structure.
func (e *Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if e.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if e.SchemaVersion == "" {
		return fmt.Errorf("schema version is required")
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at timestamp is required")
	}
	return nil
}

// Copy creates a copy of the event with a new ID.
func (e *Event) Copy() *Event {
	metadata := make(map[string]string)
	for k, v := range e.Metadata {
		metadata[k] = v
	}

	payload := make([]byte, len(e.Payload))
	copy(payload, e.Payload)

	return &Event{
		ID:            uuid.New().String(),
		Type:          e.Type,
		SchemaVersion: e.SchemaVersion,
		OccurredAt:    time.Now().UTC(),
		TraceID:       e.TraceID,
		AgentID:       e.AgentID,
		TenantID:      e.TenantID,
		Payload:       payload,
		Metadata:      metadata,
	}
}

// IsOlderThan checks if the event is older than the given duration.
func (e *Event) IsOlderThan(duration time.Duration) bool {
	return time.Since(e.OccurredAt) > duration
}

// EventType constants for standard Agent OS events.
const (
	// Agent lifecycle events
	EventAgentCreated   = "agent.created"
	EventAgentUpdated   = "agent.updated"
	EventAgentDeleted   = "agent.deleted"
	EventAgentScheduled = "agent.scheduled"
	EventAgentStarting  = "agent.starting"
	EventAgentReady     = "agent.ready"
	EventAgentStopping  = "agent.stopping"
	EventAgentStopped   = "agent.stopped"
	EventAgentFailed    = "agent.failed"
	EventAgentRecovered = "agent.recovered"

	// Workflow events
	EventWorkflowStarted   = "workflow.started"
	EventWorkflowCompleted = "workflow.completed"
	EventWorkflowFailed    = "workflow.failed"
	EventWorkflowStepStarted   = "workflow.step.started"
	EventWorkflowStepCompleted = "workflow.step.completed"
	EventWorkflowStepFailed    = "workflow.step.failed"

	// Saga events
	EventSagaStarted     = "saga.started"
	EventSagaCompleted   = "saga.completed"
	EventSagaFailed      = "saga.failed"
	EventSagaCompensated = "saga.compensated"

	// Resource events
	EventResourceAllocated = "resource.allocated"
	EventResourceReleased  = "resource.released"
	EventResourceScaled    = "resource.scaled"

	// Task events
	EventTaskScheduled = "task.scheduled"
	EventTaskAssigned  = "task.assigned"
	EventTaskCompleted = "task.completed"
	EventTaskFailed    = "task.failed"

	// Security events
	EventSecurityAlert      = "security.alert"
	EventAccessDenied       = "security.access_denied"
	EventPolicyViolation    = "security.policy_violation"
	EventAuthenticationFailed = "security.auth_failed"

	// System events
	EventSystemHealthCheck = "system.health_check"
	EventSystemStartup     = "system.startup"
	EventSystemShutdown    = "system.shutdown"
)

// EventFilter provides filtering capabilities for events.
type EventFilter struct {
	EventTypes []string
	AgentID    string
	TenantID   string
	TraceID    string
	Since      time.Time
	Until      time.Time
}

// Match checks if an event matches the filter.
func (f *EventFilter) Match(event *Event) bool {
	if len(f.EventTypes) > 0 {
		matched := false
		for _, et := range f.EventTypes {
			if event.Type == et {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if f.AgentID != "" && event.AgentID != f.AgentID {
		return false
	}

	if f.TenantID != "" && event.TenantID != f.TenantID {
		return false
	}

	if f.TraceID != "" && event.TraceID != f.TraceID {
		return false
	}

	if !f.Since.IsZero() && event.OccurredAt.Before(f.Since) {
		return false
	}

	if !f.Until.IsZero() && event.OccurredAt.After(f.Until) {
		return false
	}

	return true
}
