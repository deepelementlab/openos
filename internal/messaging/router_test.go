package messaging

import (
	"context"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

func newTestRouter() *EventRouter {
	return NewEventRouter(zap.NewNop())
}

func TestEventRouter_RegisterHandler(t *testing.T) {
	router := newTestRouter()
	handler := func(ctx context.Context, event *Event) error { return nil }

	if err := router.RegisterHandler("agent.created", handler); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if router.GetHandlerCount("agent.created") != 1 {
		t.Errorf("expected 1 handler, got %d", router.GetHandlerCount("agent.created"))
	}
}

func TestEventRouter_RegisterHandler_Duplicate(t *testing.T) {
	router := newTestRouter()
	handler := func(ctx context.Context, event *Event) error { return nil }

	router.RegisterHandler("test", handler)
	if err := router.RegisterHandler("test", handler); err == nil {
		t.Error("expected error for duplicate handler")
	}
}

func TestEventRouter_RegisterHandler_DifferentHandlers(t *testing.T) {
	router := newTestRouter()
	h1 := func(ctx context.Context, event *Event) error { return nil }
	h2 := func(ctx context.Context, event *Event) error { return nil }

	if err := router.RegisterHandler("test", h1); err != nil {
		t.Fatalf("register h1 failed: %v", err)
	}
	if err := router.RegisterHandler("test", h2); err != nil {
		t.Fatalf("register h2 failed: %v", err)
	}
	if router.GetHandlerCount("test") != 2 {
		t.Errorf("expected 2 handlers, got %d", router.GetHandlerCount("test"))
	}
}

func TestEventRouter_UnregisterHandler(t *testing.T) {
	router := newTestRouter()
	handler := func(ctx context.Context, event *Event) error { return nil }

	router.RegisterHandler("test", handler)
	router.UnregisterHandler("test", handler)

	if router.GetHandlerCount("test") != 0 {
		t.Errorf("expected 0 handlers after unregister, got %d", router.GetHandlerCount("test"))
	}
}

func TestEventRouter_UnregisterHandler_NotFound(t *testing.T) {
	router := newTestRouter()
	handler := func(ctx context.Context, event *Event) error { return nil }
	router.UnregisterHandler("test", handler)
}

func TestEventRouter_Route(t *testing.T) {
	router := newTestRouter()
	var received atomic.Int32
	handler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	router.RegisterHandler("agent.created", handler)
	event, _ := NewEvent("agent.created", nil)

	if err := router.Route(context.Background(), event); err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if received.Load() != 1 {
		t.Errorf("expected 1 handler call, got %d", received.Load())
	}
}

func TestEventRouter_Route_NoHandlers(t *testing.T) {
	router := newTestRouter()
	event, _ := NewEvent("nonexistent.type", nil)

	if err := router.Route(context.Background(), event); err != nil {
		t.Fatalf("route with no handlers should not error: %v", err)
	}
}

func TestEventRouter_Route_Wildcard(t *testing.T) {
	router := newTestRouter()
	var received atomic.Int32
	globalHandler := func(ctx context.Context, event *Event) error {
		received.Add(1)
		return nil
	}

	router.RegisterHandler("*", globalHandler)

	event, _ := NewEvent("any.type", nil)
	if err := router.Route(context.Background(), event); err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if received.Load() != 1 {
		t.Errorf("expected wildcard handler to be called, got %d", received.Load())
	}
}

func TestEventRouter_Route_SpecificAndWildcard(t *testing.T) {
	router := newTestRouter()
	var specificCalled, globalCalled atomic.Bool

	specificHandler := func(ctx context.Context, event *Event) error {
		specificCalled.Store(true)
		return nil
	}
	globalHandler := func(ctx context.Context, event *Event) error {
		globalCalled.Store(true)
		return nil
	}

	router.RegisterHandler("agent.created", specificHandler)
	router.RegisterHandler("*", globalHandler)

	event, _ := NewEvent("agent.created", nil)
	router.Route(context.Background(), event)

	if !specificCalled.Load() {
		t.Error("expected specific handler to be called")
	}
	if !globalCalled.Load() {
		t.Error("expected wildcard handler to be called")
	}
}

func TestEventRouter_Route_HandlerError(t *testing.T) {
	router := newTestRouter()
	failHandler := func(ctx context.Context, event *Event) error {
		return context.Canceled
	}

	router.RegisterHandler("test", failHandler)
	event, _ := NewEvent("test", nil)

	if err := router.Route(context.Background(), event); err == nil {
		t.Error("expected error from failing handler")
	}
}

func TestEventRouter_Route_MultipleHandlers(t *testing.T) {
	router := newTestRouter()
	var count atomic.Int32
	h1 := func(ctx context.Context, event *Event) error { count.Add(1); return nil }
	h2 := func(ctx context.Context, event *Event) error { count.Add(1); return nil }
	h3 := func(ctx context.Context, event *Event) error { count.Add(1); return nil }

	router.RegisterHandler("test", h1)
	router.RegisterHandler("test", h2)
	router.RegisterHandler("test", h3)

	event, _ := NewEvent("test", nil)
	router.Route(context.Background(), event)

	if count.Load() != 3 {
		t.Errorf("expected 3 handler calls, got %d", count.Load())
	}
}

func TestEventRouter_GetAllHandlerCounts(t *testing.T) {
	router := newTestRouter()
	handler := func(ctx context.Context, event *Event) error { return nil }

	router.RegisterHandler("a", handler)
	router.RegisterHandler("b", handler)
	router.RegisterHandler("b", func(ctx context.Context, event *Event) error { return nil })

	counts := router.GetAllHandlerCounts()
	if counts["a"] != 1 {
		t.Errorf("expected 1 handler for a, got %d", counts["a"])
	}
	if counts["b"] != 2 {
		t.Errorf("expected 2 handlers for b, got %d", counts["b"])
	}
}
