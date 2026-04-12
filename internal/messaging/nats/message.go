package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message represents a message to be published or consumed from NATS.
type Message struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Payload     []byte            `json:"payload"`
	Headers     map[string]string `json:"headers"`
	Timestamp   time.Time         `json:"timestamp"`
	ContentType string            `json:"content_type"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType string, payload []byte) *Message {
	return &Message{
		ID:          uuid.New().String(),
		Type:        msgType,
		Payload:     payload,
		Headers:     make(map[string]string),
		Timestamp:   time.Now().UTC(),
		ContentType: "application/json",
	}
}

// NewMessageFromObject creates a message from an object (serialized to JSON).
func NewMessageFromObject(msgType string, obj interface{}) (*Message, error) {
	payload, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object: %w", err)
	}

	return NewMessage(msgType, payload), nil
}

// SetHeader sets a header value.
func (m *Message) SetHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
}

// GetHeader gets a header value.
func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// SetTraceID sets the trace ID for distributed tracing.
func (m *Message) SetTraceID(traceID string) {
	m.SetHeader("trace_id", traceID)
}

// GetTraceID gets the trace ID.
func (m *Message) GetTraceID() string {
	return m.GetHeader("trace_id")
}

// SetAgentID sets the agent ID associated with this message.
func (m *Message) SetAgentID(agentID string) {
	m.SetHeader("agent_id", agentID)
}

// GetAgentID gets the agent ID.
func (m *Message) GetAgentID() string {
	return m.GetHeader("agent_id")
}

// SetTenantID sets the tenant ID for multi-tenancy.
func (m *Message) SetTenantID(tenantID string) {
	m.SetHeader("tenant_id", tenantID)
}

// GetTenantID gets the tenant ID.
func (m *Message) GetTenantID() string {
	return m.GetHeader("tenant_id")
}

// ToJSON serializes the message to JSON.
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes a message from JSON.
func (m *Message) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// PayloadObject deserializes the payload into an object.
func (m *Message) PayloadObject(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// Validate validates the message structure.
func (m *Message) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message ID is required")
	}
	if m.Type == "" {
		return fmt.Errorf("message type is required")
	}
	return nil
}

// Copy creates a copy of the message.
func (m *Message) Copy() *Message {
	headers := make(map[string]string)
	for k, v := range m.Headers {
		headers[k] = v
	}

	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)

	return &Message{
		ID:          uuid.New().String(), // New ID for the copy
		Type:        m.Type,
		Payload:     payload,
		Headers:     headers,
		Timestamp:   time.Now().UTC(),
		ContentType: m.ContentType,
	}
}

// TopicBuilder helps construct NATS topics following naming conventions.
type TopicBuilder struct {
	domain  string
	entity  string
	action  string
	version string
}

// NewTopicBuilder creates a new topic builder.
func NewTopicBuilder() *TopicBuilder {
	return &TopicBuilder{
		version: "v1",
	}
}

// Domain sets the domain (e.g., "agent", "scheduler", "workflow").
func (tb *TopicBuilder) Domain(domain string) *TopicBuilder {
	tb.domain = domain
	return tb
}

// Entity sets the entity type (e.g., "lifecycle", "task", "resource").
func (tb *TopicBuilder) Entity(entity string) *TopicBuilder {
	tb.entity = entity
	return tb
}

// Action sets the action (e.g., "created", "updated", "deleted").
func (tb *TopicBuilder) Action(action string) *TopicBuilder {
	tb.action = action
	return tb
}

// Version sets the version (default: v1).
func (tb *TopicBuilder) Version(version string) *TopicBuilder {
	tb.version = version
	return tb
}

// Build constructs the topic string: aos.{domain}.{entity}.{action}.{version}
func (tb *TopicBuilder) Build() string {
	return fmt.Sprintf("aos.%s.%s.%s.%s", tb.domain, tb.entity, tb.action, tb.version)
}

// ParseTopic parses a topic string into components.
func ParseTopic(topic string) (domain, entity, action, version string, err error) {
	// Expected format: aos.{domain}.{entity}.{action}.{version}
	_, err = fmt.Sscanf(topic, "aos.%s.%s.%s.%s", &domain, &entity, &action, &version)
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return domain, entity, action, version, nil
}

// Standard topic patterns.
const (
	TopicAgentCreated   = "aos.agent.lifecycle.created.v1"
	TopicAgentUpdated   = "aos.agent.lifecycle.updated.v1"
	TopicAgentDeleted   = "aos.agent.lifecycle.deleted.v1"
	TopicAgentStarted   = "aos.agent.lifecycle.started.v1"
	TopicAgentStopped   = "aos.agent.lifecycle.stopped.v1"
	TopicTaskScheduled  = "aos.scheduler.task.scheduled.v1"
	TopicTaskCompleted  = "aos.scheduler.task.completed.v1"
	TopicTaskFailed     = "aos.scheduler.task.failed.v1"
	TopicResourceAllocated = "aos.resource.allocated.v1"
	TopicResourceReleased  = "aos.resource.released.v1"
)
