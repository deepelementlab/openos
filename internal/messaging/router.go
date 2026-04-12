package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventRouter routes events to registered handlers.
type EventRouter struct {
	handlers map[string][]EventHandler
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewEventRouter creates a new event router.
func NewEventRouter(logger *zap.Logger) *EventRouter {
	return &EventRouter{
		handlers: make(map[string][]EventHandler),
		logger:   logger,
	}
}

// RegisterHandler registers a handler for a specific event type.
func (r *EventRouter) RegisterHandler(eventType string, handler EventHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler already registered for this event type
	for _, h := range r.handlers[eventType] {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			return fmt.Errorf("handler already registered for event type: %s", eventType)
		}
	}

	r.handlers[eventType] = append(r.handlers[eventType], handler)
	r.logger.Debug("Registered event handler", zap.String("event_type", eventType))
	return nil
}

// UnregisterHandler unregisters a handler for a specific event type.
func (r *EventRouter) UnregisterHandler(eventType string, handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	handlers := r.handlers[eventType]
	targetPtr := fmt.Sprintf("%p", handler)

	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == targetPtr {
			// Remove handler
			r.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			r.logger.Debug("Unregistered event handler", zap.String("event_type", eventType))
			break
		}
	}
}

// Route routes an event to all registered handlers.
func (r *EventRouter) Route(ctx context.Context, event *Event) error {
	r.mu.RLock()
	handlers := r.handlers[event.Type]
	allHandlers := r.handlers["*"] // Global handlers
	r.mu.RUnlock()

	if len(handlers) == 0 && len(allHandlers) == 0 {
		// No handlers registered for this event type
		r.logger.Debug("No handlers for event type", zap.String("event_type", event.Type))
		return nil
	}

	// Combine specific and global handlers
	all := append(handlers, allHandlers...)

	var errs []error
	for _, handler := range all {
		if err := r.callHandler(ctx, handler, event); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d handlers failed: %v", len(errs), errs)
	}

	return nil
}

// callHandler calls a handler with timeout and error handling.
func (r *EventRouter) callHandler(ctx context.Context, handler EventHandler, event *Event) error {
	// Create timeout context
	handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call handler
	err := handler(handlerCtx, event)
	if err != nil {
		r.logger.Error("Event handler failed",
			zap.Error(err),
			zap.String("event_type", event.Type),
			zap.String("event_id", event.ID),
		)
		return err
	}

	return nil
}

// GetHandlerCount returns the number of handlers for an event type.
func (r *EventRouter) GetHandlerCount(eventType string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers[eventType])
}

// GetAllHandlerCounts returns a map of event types to handler counts.
func (r *EventRouter) GetAllHandlerCounts() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	counts := make(map[string]int)
	for eventType, handlers := range r.handlers {
		counts[eventType] = len(handlers)
	}
	return counts
}
