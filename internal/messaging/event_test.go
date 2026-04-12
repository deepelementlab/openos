package messaging

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	payload := map[string]string{"key": "value"}
	event, err := NewEvent("agent.created", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID == "" {
		t.Error("expected non-empty ID")
	}
	if event.Type != "agent.created" {
		t.Errorf("expected type agent.created, got %s", event.Type)
	}
	if event.SchemaVersion != "1.0" {
		t.Errorf("expected schema version 1.0, got %s", event.SchemaVersion)
	}
	if event.OccurredAt.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Metadata == nil {
		t.Error("expected non-nil metadata map")
	}
	if len(event.Payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestNewEvent_NilPayload(t *testing.T) {
	event, err := NewEvent("test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Payload == nil {
		t.Error("expected non-nil payload")
	}
}

func TestNewEvent_InvalidPayload(t *testing.T) {
	_, err := NewEvent("test", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshallable payload")
	}
}

func TestNewEventWithID(t *testing.T) {
	event, err := NewEventWithID("my-id", "test.type", map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "my-id" {
		t.Errorf("expected ID my-id, got %s", event.ID)
	}
	if event.Type != "test.type" {
		t.Errorf("expected type test.type, got %s", event.Type)
	}
}

func TestNewEventWithID_InvalidPayload(t *testing.T) {
	_, err := NewEventWithID("id", "test", make(chan int))
	if err == nil {
		t.Error("expected error")
	}
}

func TestEvent_SetTraceID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	result := event.SetTraceID("trace-123")
	if result != event {
		t.Error("expected fluent return (same pointer)")
	}
	if event.TraceID != "trace-123" {
		t.Errorf("expected trace-123, got %s", event.TraceID)
	}
}

func TestEvent_SetAgentID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	result := event.SetAgentID("agent-456")
	if result != event {
		t.Error("expected fluent return")
	}
	if event.AgentID != "agent-456" {
		t.Errorf("expected agent-456, got %s", event.AgentID)
	}
}

func TestEvent_SetTenantID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	result := event.SetTenantID("tenant-789")
	if result != event {
		t.Error("expected fluent return")
	}
	if event.TenantID != "tenant-789" {
		t.Errorf("expected tenant-789, got %s", event.TenantID)
	}
}

func TestEvent_SetMetadata(t *testing.T) {
	event, _ := NewEvent("test", nil)
	result := event.SetMetadata("key1", "value1")
	if result != event {
		t.Error("expected fluent return")
	}
	if event.Metadata["key1"] != "value1" {
		t.Errorf("expected value1, got %s", event.Metadata["key1"])
	}
}

func TestEvent_SetMetadata_NilMap(t *testing.T) {
	event := &Event{}
	event.SetMetadata("key", "value")
	if event.Metadata["key"] != "value" {
		t.Error("expected metadata to be created and set")
	}
}

func TestEvent_GetMetadata(t *testing.T) {
	event, _ := NewEvent("test", nil)
	event.SetMetadata("foo", "bar")
	if v := event.GetMetadata("foo"); v != "bar" {
		t.Errorf("expected bar, got %s", v)
	}
	if v := event.GetMetadata("nonexistent"); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestEvent_GetMetadata_NilMap(t *testing.T) {
	event := &Event{}
	if v := event.GetMetadata("key"); v != "" {
		t.Errorf("expected empty string for nil metadata, got %s", v)
	}
}

func TestEvent_PayloadObject(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}
	event, _ := NewEvent("test", sample{Name: "hello"})
	var s sample
	if err := event.PayloadObject(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "hello" {
		t.Errorf("expected hello, got %s", s.Name)
	}
}

func TestEvent_PayloadObject_Invalid(t *testing.T) {
	event, _ := NewEvent("test", "valid")
	var m map[string]string
	if err := event.PayloadObject(&m); err == nil {
		t.Error("expected error for invalid payload object type")
	}
}

func TestEvent_ToJSON(t *testing.T) {
	event, _ := NewEvent("test", map[string]string{"k": "v"})
	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["event_type"] != "test" {
		t.Errorf("expected event_type test, got %v", parsed["event_type"])
	}
}

func TestEvent_FromJSON(t *testing.T) {
	original, _ := NewEvent("test", map[string]string{"k": "v"})
	original.SetTraceID("trace-1")
	original.SetAgentID("agent-1")
	original.SetMetadata("meta", "data")

	data, _ := original.ToJSON()
	parsed := &Event{}
	if err := parsed.FromJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, parsed.ID)
	}
	if parsed.Type != original.Type {
		t.Errorf("expected type %s, got %s", original.Type, parsed.Type)
	}
	if parsed.TraceID != "trace-1" {
		t.Errorf("expected trace-1, got %s", parsed.TraceID)
	}
	if parsed.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", parsed.AgentID)
	}
	if parsed.GetMetadata("meta") != "data" {
		t.Error("expected metadata to be preserved")
	}
}

func TestEvent_Validate(t *testing.T) {
	event, _ := NewEvent("test", nil)
	if err := event.Validate(); err != nil {
		t.Errorf("valid event should pass: %v", err)
	}
}

func TestEvent_Validate_MissingID(t *testing.T) {
	event := &Event{Type: "test", SchemaVersion: "1.0", OccurredAt: time.Now()}
	if err := event.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestEvent_Validate_MissingType(t *testing.T) {
	event := &Event{ID: "id", SchemaVersion: "1.0", OccurredAt: time.Now()}
	if err := event.Validate(); err == nil {
		t.Error("expected error for missing type")
	}
}

func TestEvent_Validate_MissingSchemaVersion(t *testing.T) {
	event := &Event{ID: "id", Type: "test", OccurredAt: time.Now()}
	if err := event.Validate(); err == nil {
		t.Error("expected error for missing schema version")
	}
}

func TestEvent_Validate_ZeroTimestamp(t *testing.T) {
	event := &Event{ID: "id", Type: "test", SchemaVersion: "1.0"}
	if err := event.Validate(); err == nil {
		t.Error("expected error for zero timestamp")
	}
}

func TestEvent_Copy(t *testing.T) {
	original, _ := NewEvent("test", map[string]string{"k": "v"})
	original.SetTraceID("trace-1")
	original.SetMetadata("m", "v")

	copy := original.Copy()
	if copy.ID == original.ID {
		t.Error("expected different ID after copy")
	}
	if copy.Type != original.Type {
		t.Errorf("expected same type, got %s", copy.Type)
	}
	if copy.TraceID != original.TraceID {
		t.Errorf("expected same trace ID, got %s", copy.TraceID)
	}
	if copy.GetMetadata("m") != "v" {
		t.Error("expected metadata to be copied")
	}
	copy.SetMetadata("m", "changed")
	if original.GetMetadata("m") != "v" {
		t.Error("expected original metadata to be unchanged after copy modification")
	}
}

func TestEvent_IsOlderThan(t *testing.T) {
	event := &Event{
		ID:            "id",
		Type:          "test",
		SchemaVersion: "1.0",
		OccurredAt:    time.Now().Add(-2 * time.Hour),
	}
	if !event.IsOlderThan(1 * time.Hour) {
		t.Error("expected 2h old event to be older than 1h")
	}
	if event.IsOlderThan(3 * time.Hour) {
		t.Error("expected 2h old event to NOT be older than 3h")
	}
}

func TestEvent_FluentChaining(t *testing.T) {
	event, _ := NewEvent("test", nil)
	event.SetTraceID("t1").SetAgentID("a1").SetTenantID("tn1").SetMetadata("k", "v")
	if event.TraceID != "t1" {
		t.Errorf("expected t1, got %s", event.TraceID)
	}
	if event.AgentID != "a1" {
		t.Errorf("expected a1, got %s", event.AgentID)
	}
	if event.TenantID != "tn1" {
		t.Errorf("expected tn1, got %s", event.TenantID)
	}
	if event.Metadata["k"] != "v" {
		t.Error("expected metadata to be set")
	}
}

func TestEventTypeConstants(t *testing.T) {
	if EventAgentCreated != "agent.created" {
		t.Errorf("unexpected EventAgentCreated: %s", EventAgentCreated)
	}
	if EventAgentDeleted != "agent.deleted" {
		t.Errorf("unexpected EventAgentDeleted: %s", EventAgentDeleted)
	}
	if EventWorkflowStarted != "workflow.started" {
		t.Errorf("unexpected EventWorkflowStarted: %s", EventWorkflowStarted)
	}
	if EventSagaStarted != "saga.started" {
		t.Errorf("unexpected EventSagaStarted: %s", EventSagaStarted)
	}
	if EventTaskScheduled != "task.scheduled" {
		t.Errorf("unexpected EventTaskScheduled: %s", EventTaskScheduled)
	}
	if EventSecurityAlert != "security.alert" {
		t.Errorf("unexpected EventSecurityAlert: %s", EventSecurityAlert)
	}
	if EventSystemHealthCheck != "system.health_check" {
		t.Errorf("unexpected EventSystemHealthCheck: %s", EventSystemHealthCheck)
	}
}

func TestEventFilter_Match_EventType(t *testing.T) {
	event, _ := NewEvent("agent.created", nil)
	filter := &EventFilter{EventTypes: []string{"agent.created", "agent.updated"}}
	if !filter.Match(event) {
		t.Error("expected event to match filter")
	}
	filter2 := &EventFilter{EventTypes: []string{"agent.deleted"}}
	if filter2.Match(event) {
		t.Error("expected event to NOT match filter")
	}
}

func TestEventFilter_Match_AgentID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	event.SetAgentID("agent-1")

	filter := &EventFilter{AgentID: "agent-1"}
	if !filter.Match(event) {
		t.Error("expected match on agent ID")
	}
	filter2 := &EventFilter{AgentID: "agent-2"}
	if filter2.Match(event) {
		t.Error("expected no match on different agent ID")
	}
}

func TestEventFilter_Match_TenantID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	event.SetTenantID("tenant-1")

	filter := &EventFilter{TenantID: "tenant-1"}
	if !filter.Match(event) {
		t.Error("expected match on tenant ID")
	}
	filter2 := &EventFilter{TenantID: "tenant-2"}
	if filter2.Match(event) {
		t.Error("expected no match on different tenant ID")
	}
}

func TestEventFilter_Match_TraceID(t *testing.T) {
	event, _ := NewEvent("test", nil)
	event.SetTraceID("trace-1")

	filter := &EventFilter{TraceID: "trace-1"}
	if !filter.Match(event) {
		t.Error("expected match on trace ID")
	}
}

func TestEventFilter_Match_TimeRange(t *testing.T) {
	event := &Event{
		ID:            "id",
		Type:          "test",
		SchemaVersion: "1.0",
		OccurredAt:    time.Now().Add(-1 * time.Hour),
	}

	since := time.Now().Add(-2 * time.Hour)
	until := time.Now().Add(-30 * time.Minute)
	filter := &EventFilter{Since: since, Until: until}
	if !filter.Match(event) {
		t.Error("expected match within time range")
	}

	filterEarly := &EventFilter{Since: time.Now().Add(-30 * time.Minute)}
	if filterEarly.Match(event) {
		t.Error("expected no match - event before 'since'")
	}

	filterLate := &EventFilter{Until: time.Now().Add(-2 * time.Hour)}
	if filterLate.Match(event) {
		t.Error("expected no match - event after 'until'")
	}
}

func TestEventFilter_Match_EmptyFilter(t *testing.T) {
	event, _ := NewEvent("test", nil)
	filter := &EventFilter{}
	if !filter.Match(event) {
		t.Error("empty filter should match all events")
	}
}
