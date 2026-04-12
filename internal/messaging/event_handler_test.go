package messaging

import (
	"context"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

func TestHandlerRegistry_Register(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }

	reg := &HandlerRegistration{
		ID:         "handler-1",
		EventTypes: []string{"agent.created"},
		Handler:    handler,
	}
	if err := registry.Register(reg); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if count := registry.GetHandlerCount("agent.created"); count != 1 {
		t.Errorf("expected 1 handler, got %d", count)
	}
}

func TestHandlerRegistry_Register_MissingID(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	reg := &HandlerRegistration{
		Handler:    func(ctx context.Context, event *Event) error { return nil },
		EventTypes: []string{"test"},
	}
	if err := registry.Register(reg); err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestHandlerRegistry_Register_NilHandler(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	reg := &HandlerRegistration{
		ID:         "handler-1",
		EventTypes: []string{"test"},
	}
	if err := registry.Register(reg); err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestHandlerRegistry_Register_NoEventTypes(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	reg := &HandlerRegistration{
		ID:      "handler-1",
		Handler: func(ctx context.Context, event *Event) error { return nil },
	}
	if err := registry.Register(reg); err == nil {
		t.Error("expected error for missing event types")
	}
}

func TestHandlerRegistry_Register_DuplicateID(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }
	reg1 := &HandlerRegistration{ID: "dup", Handler: handler, EventTypes: []string{"a"}}
	reg2 := &HandlerRegistration{ID: "dup", Handler: handler, EventTypes: []string{"b"}}

	if err := registry.Register(reg1); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := registry.Register(reg2); err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestHandlerRegistry_Unregister(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }
	reg := &HandlerRegistration{ID: "h1", Handler: handler, EventTypes: []string{"test"}}
	registry.Register(reg)

	if err := registry.Unregister("h1"); err != nil {
		t.Fatalf("unregister failed: %v", err)
	}
	if count := registry.GetHandlerCount("test"); count != 0 {
		t.Errorf("expected 0 handlers after unregister, got %d", count)
	}
}

func TestHandlerRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	if err := registry.Unregister("nonexistent"); err == nil {
		t.Error("expected error for unregistering nonexistent handler")
	}
}

func TestHandlerRegistry_GetHandlers(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	var called1, called2 atomic.Bool
	h1 := func(ctx context.Context, event *Event) error { called1.Store(true); return nil }
	h2 := func(ctx context.Context, event *Event) error { called2.Store(true); return nil }

	registry.Register(&HandlerRegistration{ID: "h1", Handler: h1, EventTypes: []string{"test"}})
	registry.Register(&HandlerRegistration{ID: "h2", Handler: h2, EventTypes: []string{"test"}})

	handlers := registry.GetHandlers("test")
	if len(handlers) != 2 {
		t.Fatalf("expected 2 handlers, got %d", len(handlers))
	}
	for _, h := range handlers {
		h(context.Background(), &Event{})
	}
	if !called1.Load() || !called2.Load() {
		t.Error("expected both handlers to be callable")
	}
}

func TestHandlerRegistry_GetHandlers_NoMatch(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handlers := registry.GetHandlers("nonexistent")
	if len(handlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(handlers))
	}
}

func TestHandlerRegistry_GetAllHandlers(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }
	registry.Register(&HandlerRegistration{ID: "h1", Handler: handler, EventTypes: []string{"a"}})
	registry.Register(&HandlerRegistration{ID: "h2", Handler: handler, EventTypes: []string{"b"}})

	all := registry.GetAllHandlers()
	if len(all) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(all))
	}
}

func TestHandlerRegistry_Priority(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	var order []string
	h1 := func(ctx context.Context, event *Event) error { order = append(order, "low"); return nil }
	h2 := func(ctx context.Context, event *Event) error { order = append(order, "high"); return nil }

	registry.Register(&HandlerRegistration{ID: "low", Handler: h1, EventTypes: []string{"test"}, Priority: 1})
	registry.Register(&HandlerRegistration{ID: "high", Handler: h2, EventTypes: []string{"test"}, Priority: 10})

	handlers := registry.GetHandlers("test")
	if len(handlers) != 2 {
		t.Fatalf("expected 2 handlers, got %d", len(handlers))
	}

	handlers[0](context.Background(), &Event{})
	handlers[1](context.Background(), &Event{})
	if order[0] != "high" {
		t.Errorf("expected high priority first, got %s", order[0])
	}
}

func TestHandlerRegistry_GetStats(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }
	registry.Register(&HandlerRegistration{ID: "h1", Handler: handler, EventTypes: []string{"a", "b"}})

	stats := registry.GetStats()
	if stats["total_handlers"] != 1 {
		t.Errorf("expected 1 handler, got %v", stats["total_handlers"])
	}
	if stats["total_event_types"] != 2 {
		t.Errorf("expected 2 event types, got %v", stats["total_event_types"])
	}
}

func TestHandlerRegistry_MultipleEventTypes(t *testing.T) {
	registry := NewHandlerRegistry(zap.NewNop())
	handler := func(ctx context.Context, event *Event) error { return nil }
	registry.Register(&HandlerRegistration{ID: "h1", Handler: handler, EventTypes: []string{"a", "b", "c"}})

	if registry.GetHandlerCount("a") != 1 {
		t.Error("expected 1 handler for type a")
	}
	if registry.GetHandlerCount("b") != 1 {
		t.Error("expected 1 handler for type b")
	}
	if registry.GetHandlerCount("c") != 1 {
		t.Error("expected 1 handler for type c")
	}
}

func TestHandlerBuilder_Build(t *testing.T) {
	handler := func(ctx context.Context, event *Event) error { return nil }
	reg, err := NewHandlerBuilder().
		ID("my-handler").
		ForEvent("agent.created").
		Handler(handler).
		Description("test handler").
		Priority(5).
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if reg.ID != "my-handler" {
		t.Errorf("expected ID my-handler, got %s", reg.ID)
	}
	if len(reg.EventTypes) != 1 || reg.EventTypes[0] != "agent.created" {
		t.Errorf("unexpected event types: %v", reg.EventTypes)
	}
	if reg.Description != "test handler" {
		t.Errorf("expected description, got %s", reg.Description)
	}
	if reg.Priority != 5 {
		t.Errorf("expected priority 5, got %d", reg.Priority)
	}
}

func TestHandlerBuilder_ForEvents(t *testing.T) {
	handler := func(ctx context.Context, event *Event) error { return nil }
	reg, err := NewHandlerBuilder().
		ID("h").
		ForEvents("a", "b", "c").
		Handler(handler).
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(reg.EventTypes) != 3 {
		t.Errorf("expected 3 event types, got %d", len(reg.EventTypes))
	}
}

func TestHandlerBuilder_Build_NoHandler(t *testing.T) {
	_, err := NewHandlerBuilder().ID("h").ForEvent("test").Build()
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

func TestHandlerBuilder_Build_NoID(t *testing.T) {
	_, err := NewHandlerBuilder().Handler(func(ctx context.Context, event *Event) error { return nil }).ForEvent("test").Build()
	if err != nil {
		t.Fatalf("expected auto-generated ID to succeed: %v", err)
	}
}

func TestHandlerBuilder_Build_NoIDNoHandler(t *testing.T) {
	_, err := NewHandlerBuilder().ForEvent("test").Build()
	if err == nil {
		t.Error("expected error for no ID and no handler")
	}
}

func TestHandlerBuilder_Build_NoEventTypes(t *testing.T) {
	_, err := NewHandlerBuilder().ID("h").Handler(func(ctx context.Context, event *Event) error { return nil }).Build()
	if err == nil {
		t.Error("expected error for missing event types")
	}
}

func TestHandlerBuilder_MustBuild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustBuild")
		}
	}()
	NewHandlerBuilder().MustBuild()
}

func TestChainMiddleware(t *testing.T) {
	var order []string
	mw1 := func(h EventHandler) EventHandler {
		return func(ctx context.Context, event *Event) error {
			order = append(order, "mw1-before")
			err := h(ctx, event)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(h EventHandler) EventHandler {
		return func(ctx context.Context, event *Event) error {
			order = append(order, "mw2-before")
			err := h(ctx, event)
			order = append(order, "mw2-after")
			return err
		}
	}

	inner := func(ctx context.Context, event *Event) error {
		order = append(order, "inner")
		return nil
	}

	chained := ChainMiddleware(mw1, mw2)(inner)
	chained(context.Background(), &Event{})

	expected := []string{"mw1-before", "mw2-before", "inner", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("at position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := zap.NewNop()
	panicHandler := func(ctx context.Context, event *Event) error {
		panic("something went wrong")
	}

	wrapped := RecoveryMiddleware(logger)(panicHandler)
	err := wrapped(context.Background(), &Event{ID: "test", Type: "test"})
	if err == nil {
		t.Error("expected error from recovered panic")
	}
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	logger := zap.NewNop()
	normalHandler := func(ctx context.Context, event *Event) error {
		return nil
	}

	wrapped := RecoveryMiddleware(logger)(normalHandler)
	err := wrapped(context.Background(), &Event{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	handler := func(ctx context.Context, event *Event) error { return nil }
	wrapped := TimeoutMiddleware(5)(handler)
	err := wrapped(context.Background(), &Event{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
