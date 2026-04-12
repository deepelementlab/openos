package messaging

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestEventBus() *InMemoryEventBus {
	return NewInMemoryEventBus(zap.NewNop())
}

func TestInMemoryEventBus_Publish(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	var received atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	_, err := bus.Subscribe(context.Background(), "agent.created", handler)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	event, _ := NewEvent("agent.created", map[string]string{"k": "v"})
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if received.Load() != 1 {
		t.Errorf("expected 1 received, got %d", received.Load())
	}
}

func TestInMemoryEventBus_PublishToWrongType(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	var received atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	_, err := bus.Subscribe(context.Background(), "agent.created", handler)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	event, _ := NewEvent("agent.deleted", nil)
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if received.Load() != 0 {
		t.Errorf("expected 0 received for non-matching event type, got %d", received.Load())
	}
}

func TestInMemoryEventBus_Publish_InvalidEvent(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	invalidEvent := &Event{Type: "test"}
	if err := bus.Publish(context.Background(), invalidEvent); err == nil {
		t.Error("expected error for invalid event")
	}
}

func TestInMemoryEventBus_Publish_Closed(t *testing.T) {
	bus := newTestEventBus()
	bus.Close()

	event, _ := NewEvent("test", nil)
	if err := bus.Publish(context.Background(), event); err == nil {
		t.Error("expected error when publishing to closed bus")
	}
}

func TestInMemoryEventBus_Subscribe(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	handler := func(ctx context.Context, event *Event) error { return nil }
	sub, err := bus.Subscribe(context.Background(), "test", handler)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if len(sub.EventTypes()) != 1 {
		t.Errorf("expected 1 event type, got %d", len(sub.EventTypes()))
	}
	if sub.EventTypes()[0] != "test" {
		t.Errorf("expected event type 'test', got %s", sub.EventTypes()[0])
	}
}

func TestInMemoryEventBus_Subscribe_Closed(t *testing.T) {
	bus := newTestEventBus()
	bus.Close()

	handler := func(ctx context.Context, event *Event) error { return nil }
	_, err := bus.Subscribe(context.Background(), "test", handler)
	if err == nil {
		t.Error("expected error when subscribing to closed bus")
	}
}

func TestInMemoryEventBus_SubscribeMultiple(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	var received atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	sub, err := bus.SubscribeMultiple(context.Background(), []string{"agent.created", "agent.updated"}, handler)
	if err != nil {
		t.Fatalf("subscribe multiple failed: %v", err)
	}
	if len(sub.EventTypes()) != 2 {
		t.Errorf("expected 2 event types, got %d", len(sub.EventTypes()))
	}

	e1, _ := NewEvent("agent.created", nil)
	e2, _ := NewEvent("agent.updated", nil)
	bus.Publish(context.Background(), e1)
	bus.Publish(context.Background(), e2)

	if received.Load() != 2 {
		t.Errorf("expected 2 received, got %d", received.Load())
	}
}

func TestInMemoryEventBus_SubscribeMultiple_Closed(t *testing.T) {
	bus := newTestEventBus()
	bus.Close()

	handler := func(ctx context.Context, event *Event) error { return nil }
	_, err := bus.SubscribeMultiple(context.Background(), []string{"a", "b"}, handler)
	if err == nil {
		t.Error("expected error when subscribing to closed bus")
	}
}

func TestInMemoryEventBus_SubscribeAll(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	handler := func(ctx context.Context, event *Event) error { return nil }
	_, err := bus.SubscribeAll(context.Background(), handler)
	if err == nil {
		t.Error("expected error for SubscribeAll on in-memory bus")
	}
}

func TestInMemoryEventBus_Unsubscribe(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	var received atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	sub, _ := bus.Subscribe(context.Background(), "test", handler)

	e1, _ := NewEvent("test", nil)
	bus.Publish(context.Background(), e1)
	time.Sleep(10 * time.Millisecond)

	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	e2, _ := NewEvent("test", nil)
	bus.Publish(context.Background(), e2)
	time.Sleep(10 * time.Millisecond)

	if received.Load() != 1 {
		t.Errorf("expected 1 received after unsubscribe, got %d", received.Load())
	}
}

func TestInMemoryEventBus_Unsubscribe_Unknown(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	handler := func(ctx context.Context, event *Event) error { return nil }
	_, err := bus.Subscribe(context.Background(), "test", handler)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	fakeSub := &subscription{
		id:         "nonexistent",
		eventTypes: []string{"test"},
		handler:    handler,
		unsubFn: func() error {
			return bus.unsubscribe("nonexistent")
		},
	}
	if err := fakeSub.Unsubscribe(); err == nil {
		t.Error("expected error for unknown subscription")
	}
}

func TestInMemoryEventBus_Close(t *testing.T) {
	bus := newTestEventBus()
	if err := bus.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestInMemoryEventBus_Close_Idempotent(t *testing.T) {
	bus := newTestEventBus()
	bus.Close()
	bus.Close()
}

func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	var count atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		count.Add(1)
		return nil
	}

	bus.Subscribe(context.Background(), "test", handler)
	bus.Subscribe(context.Background(), "test", func(ctx context.Context, event *Event) error {
		count.Add(1)
		return nil
	})

	event, _ := NewEvent("test", nil)
	bus.Publish(context.Background(), event)
	time.Sleep(10 * time.Millisecond)

	if count.Load() != 2 {
		t.Errorf("expected 2 handler calls, got %d", count.Load())
	}
}

func TestInMemoryEventBus_HandlerError(t *testing.T) {
	bus := newTestEventBus()
	defer bus.Close()

	handler := func(ctx context.Context, event *Event) error {
		return context.Canceled
	}

	bus.Subscribe(context.Background(), "test", handler)

	event, _ := NewEvent("test", nil)
	err := bus.Publish(context.Background(), event)
	if err == nil {
		t.Error("expected error from handler")
	}
}

func TestNATSEventBus_buildTopic(t *testing.T) {
	bus := &NATSEventBus{}
	topic := bus.buildTopic("agent.created")
	if topic != "aos.agent.created.v1" {
		t.Errorf("expected aos.agent.created.v1, got %s", topic)
	}
	topic2 := bus.buildTopic("workflow.step.started")
	if topic2 != "aos.workflow.step.started.v1" {
		t.Errorf("expected aos.workflow.step.started.v1, got %s", topic2)
	}
}
