package nats

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage("agent.created", []byte(`{"key":"value"}`))
	if msg.ID == "" {
		t.Error("expected non-empty ID")
	}
	if msg.Type != "agent.created" {
		t.Errorf("expected agent.created, got %s", msg.Type)
	}
	if string(msg.Payload) != `{"key":"value"}` {
		t.Errorf("unexpected payload: %s", string(msg.Payload))
	}
	if msg.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", msg.ContentType)
	}
	if msg.Headers == nil {
		t.Error("expected non-nil headers")
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewMessageFromObject(t *testing.T) {
	obj := map[string]string{"name": "test"}
	msg, err := NewMessageFromObject("test", obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "test" {
		t.Errorf("expected test, got %s", msg.Type)
	}
	var result map[string]string
	if err := json.Unmarshal(msg.Payload, &result); err != nil {
		t.Fatalf("invalid payload JSON: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected test, got %s", result["name"])
	}
}

func TestNewMessageFromObject_Invalid(t *testing.T) {
	_, err := NewMessageFromObject("test", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshallable object")
	}
}

func TestMessage_SetHeader(t *testing.T) {
	msg := NewMessage("test", nil)
	msg.SetHeader("key1", "value1")
	if msg.Headers["key1"] != "value1" {
		t.Errorf("expected value1, got %s", msg.Headers["key1"])
	}
}

func TestMessage_SetHeader_NilMap(t *testing.T) {
	msg := &Message{}
	msg.SetHeader("key", "value")
	if msg.Headers["key"] != "value" {
		t.Error("expected header to be set after nil map init")
	}
}

func TestMessage_GetHeader(t *testing.T) {
	msg := NewMessage("test", nil)
	msg.SetHeader("key1", "value1")
	if v := msg.GetHeader("key1"); v != "value1" {
		t.Errorf("expected value1, got %s", v)
	}
	if v := msg.GetHeader("nonexistent"); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestMessage_GetHeader_NilMap(t *testing.T) {
	msg := &Message{}
	if v := msg.GetHeader("key"); v != "" {
		t.Errorf("expected empty for nil headers, got %s", v)
	}
}

func TestMessage_SetTraceID(t *testing.T) {
	msg := NewMessage("test", nil)
	msg.SetTraceID("trace-123")
	if msg.GetTraceID() != "trace-123" {
		t.Errorf("expected trace-123, got %s", msg.GetTraceID())
	}
}

func TestMessage_SetAgentID(t *testing.T) {
	msg := NewMessage("test", nil)
	msg.SetAgentID("agent-456")
	if msg.GetAgentID() != "agent-456" {
		t.Errorf("expected agent-456, got %s", msg.GetAgentID())
	}
}

func TestMessage_SetTenantID(t *testing.T) {
	msg := NewMessage("test", nil)
	msg.SetTenantID("tenant-789")
	if msg.GetTenantID() != "tenant-789" {
		t.Errorf("expected tenant-789, got %s", msg.GetTenantID())
	}
}

func TestMessage_ToJSON(t *testing.T) {
	msg := NewMessage("test", []byte(`{"data":1}`))
	msg.SetHeader("h1", "v1")
	data, err := msg.ToJSON()
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
	if parsed["type"] != "test" {
		t.Errorf("expected type test, got %v", parsed["type"])
	}
}

func TestMessage_FromJSON(t *testing.T) {
	original := NewMessage("test", []byte(`{"data":1}`))
	original.SetTraceID("trace-1")
	original.SetAgentID("agent-1")

	data, _ := original.ToJSON()
	parsed := &Message{}
	if err := parsed.FromJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, parsed.ID)
	}
	if parsed.Type != original.Type {
		t.Errorf("expected type %s, got %s", original.Type, parsed.Type)
	}
	if parsed.GetTraceID() != "trace-1" {
		t.Errorf("expected trace-1, got %s", parsed.GetTraceID())
	}
	if parsed.GetAgentID() != "agent-1" {
		t.Errorf("expected agent-1, got %s", parsed.GetAgentID())
	}
}

func TestMessage_PayloadObject(t *testing.T) {
	msg := NewMessage("test", []byte(`{"name":"hello"}`))
	var obj map[string]string
	if err := msg.PayloadObject(&obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj["name"] != "hello" {
		t.Errorf("expected hello, got %s", obj["name"])
	}
}

func TestMessage_Validate(t *testing.T) {
	msg := NewMessage("test", nil)
	if err := msg.Validate(); err != nil {
		t.Errorf("valid message should pass: %v", err)
	}
}

func TestMessage_Validate_NoID(t *testing.T) {
	msg := &Message{Type: "test"}
	if err := msg.Validate(); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestMessage_Validate_NoType(t *testing.T) {
	msg := &Message{ID: "id"}
	if err := msg.Validate(); err == nil {
		t.Error("expected error for missing type")
	}
}

func TestMessage_Copy(t *testing.T) {
	msg := NewMessage("test", []byte(`{"data":1}`))
	msg.SetHeader("key", "value")
	msg.SetTraceID("trace-1")

	copy := msg.Copy()
	if copy.ID == msg.ID {
		t.Error("expected different ID after copy")
	}
	if copy.Type != msg.Type {
		t.Errorf("expected same type, got %s", copy.Type)
	}
	if copy.GetTraceID() != "trace-1" {
		t.Error("expected trace ID to be copied")
	}
	if copy.GetHeader("key") != "value" {
		t.Error("expected header to be copied")
	}
	copy.SetHeader("key", "changed")
	if msg.GetHeader("key") != "value" {
		t.Error("expected original header to be unchanged")
	}
}

func TestTopicBuilder(t *testing.T) {
	topic := NewTopicBuilder().
		Domain("agent").
		Entity("lifecycle").
		Action("created").
		Build()
	if topic != "aos.agent.lifecycle.created.v1" {
		t.Errorf("expected aos.agent.lifecycle.created.v1, got %s", topic)
	}
}

func TestTopicBuilder_CustomVersion(t *testing.T) {
	topic := NewTopicBuilder().
		Domain("scheduler").
		Entity("task").
		Action("completed").
		Version("v2").
		Build()
	if topic != "aos.scheduler.task.completed.v2" {
		t.Errorf("expected aos.scheduler.task.completed.v2, got %s", topic)
	}
}

func TestTopicConstants(t *testing.T) {
	if TopicAgentCreated != "aos.agent.lifecycle.created.v1" {
		t.Errorf("unexpected TopicAgentCreated: %s", TopicAgentCreated)
	}
	if TopicTaskScheduled != "aos.scheduler.task.scheduled.v1" {
		t.Errorf("unexpected TopicTaskScheduled: %s", TopicTaskScheduled)
	}
	if TopicResourceAllocated != "aos.resource.allocated.v1" {
		t.Errorf("unexpected TopicResourceAllocated: %s", TopicResourceAllocated)
	}
}
