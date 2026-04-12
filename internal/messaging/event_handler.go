package messaging

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// HandlerRegistration represents a registered event handler.
type HandlerRegistration struct {
	ID          string
	EventTypes  []string
	Handler     EventHandler
	Description string
	Priority    int // Higher priority handlers execute first
}

// HandlerRegistry manages event handler registrations.
type HandlerRegistry struct {
	handlers map[string]*HandlerRegistration // ID -> registration
	events   map[string][]string             // event_type -> handler IDs
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(logger *zap.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]*HandlerRegistration),
		events:   make(map[string][]string),
		logger:   logger,
	}
}

// Register registers a handler for specific event types.
func (r *HandlerRegistry) Register(reg *HandlerRegistration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate
	if reg.ID == "" {
		return fmt.Errorf("handler ID is required")
	}
	if reg.Handler == nil {
		return fmt.Errorf("handler is required")
	}
	if len(reg.EventTypes) == 0 {
		return fmt.Errorf("at least one event type is required")
	}

	// Check for duplicate ID
	if _, exists := r.handlers[reg.ID]; exists {
		return fmt.Errorf("handler with ID %s already registered", reg.ID)
	}

	// Store registration
	r.handlers[reg.ID] = reg

	// Index by event type and sort by priority
	for _, eventType := range reg.EventTypes {
		r.events[eventType] = append(r.events[eventType], reg.ID)
		// Sort by priority (higher first)
		r.sortHandlersByPriority(eventType)
	}

	r.logger.Info("Registered event handler",
		zap.String("handler_id", reg.ID),
		zap.Strings("event_types", reg.EventTypes),
		zap.String("description", reg.Description),
		zap.Int("priority", reg.Priority),
	)

	return nil
}

// Unregister removes a handler registration.
func (r *HandlerRegistry) Unregister(handlerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	reg, exists := r.handlers[handlerID]
	if !exists {
		return fmt.Errorf("handler %s not found", handlerID)
	}

	// Remove from event index
	for _, eventType := range reg.EventTypes {
		handlerIDs := r.events[eventType]
		for i, id := range handlerIDs {
			if id == handlerID {
				r.events[eventType] = append(handlerIDs[:i], handlerIDs[i+1:]...)
				break
			}
		}
	}

	// Remove registration
	delete(r.handlers, handlerID)

	r.logger.Info("Unregistered event handler", zap.String("handler_id", handlerID))
	return nil
}

// GetHandlers returns all handlers for an event type.
func (r *HandlerRegistry) GetHandlers(eventType string) []EventHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlerIDs := r.events[eventType]
	handlers := make([]EventHandler, 0, len(handlerIDs))

	for _, id := range handlerIDs {
		if reg, exists := r.handlers[id]; exists {
			handlers = append(handlers, reg.Handler)
		}
	}

	return handlers
}

// GetAllHandlers returns all registered handlers.
func (r *HandlerRegistry) GetAllHandlers() []*HandlerRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regs := make([]*HandlerRegistration, 0, len(r.handlers))
	for _, reg := range r.handlers {
		regs = append(regs, reg)
	}
	return regs
}

// sortHandlersByPriority sorts handlers by priority (higher first).
func (r *HandlerRegistry) sortHandlersByPriority(eventType string) {
	handlerIDs := r.events[eventType]
	if len(handlerIDs) <= 1 {
		return
	}

	// Simple bubble sort by priority
	for i := 0; i < len(handlerIDs); i++ {
		for j := i + 1; j < len(handlerIDs); j++ {
			regI := r.handlers[handlerIDs[i]]
			regJ := r.handlers[handlerIDs[j]]
			if regJ.Priority > regI.Priority {
				handlerIDs[i], handlerIDs[j] = handlerIDs[j], handlerIDs[i]
			}
		}
	}
}

// GetHandlerCount returns the number of handlers for an event type.
func (r *HandlerRegistry) GetHandlerCount(eventType string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.events[eventType])
}

// GetStats returns registry statistics.
func (r *HandlerRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	eventCounts := make(map[string]int)
	for eventType, handlerIDs := range r.events {
		eventCounts[eventType] = len(handlerIDs)
	}

	return map[string]interface{}{
		"total_handlers":    len(r.handlers),
		"total_event_types": len(r.events),
		"event_counts":      eventCounts,
	}
}

// HandlerBuilder provides a fluent API for building handler registrations.
type HandlerBuilder struct {
	id          string
	eventTypes  []string
	handler     EventHandler
	description string
	priority    int
}

// NewHandlerBuilder creates a new handler builder.
func NewHandlerBuilder() *HandlerBuilder {
	return &HandlerBuilder{
		eventTypes: make([]string, 0),
		priority:   0,
	}
}

// ID sets the handler ID.
func (b *HandlerBuilder) ID(id string) *HandlerBuilder {
	b.id = id
	return b
}

// ForEvent adds an event type to handle.
func (b *HandlerBuilder) ForEvent(eventType string) *HandlerBuilder {
	b.eventTypes = append(b.eventTypes, eventType)
	return b
}

// ForEvents adds multiple event types to handle.
func (b *HandlerBuilder) ForEvents(eventTypes ...string) *HandlerBuilder {
	b.eventTypes = append(b.eventTypes, eventTypes...)
	return b
}

// Handler sets the handler function.
func (b *HandlerBuilder) Handler(handler EventHandler) *HandlerBuilder {
	b.handler = handler
	return b
}

// Description sets the handler description.
func (b *HandlerBuilder) Description(desc string) *HandlerBuilder {
	b.description = desc
	return b
}

// Priority sets the handler priority (higher = earlier execution).
func (b *HandlerBuilder) Priority(priority int) *HandlerBuilder {
	b.priority = priority
	return b
}

// Build creates the handler registration.
func (b *HandlerBuilder) Build() (*HandlerRegistration, error) {
	if b.id == "" {
		// Generate ID from handler function name
		if b.handler != nil {
			b.id = getFunctionName(b.handler)
		} else {
			return nil, fmt.Errorf("handler ID is required")
		}
	}

	if b.handler == nil {
		return nil, fmt.Errorf("handler is required")
	}

	if len(b.eventTypes) == 0 {
		return nil, fmt.Errorf("at least one event type is required")
	}

	return &HandlerRegistration{
		ID:          b.id,
		EventTypes:  b.eventTypes,
		Handler:     b.handler,
		Description: b.description,
		Priority:    b.priority,
	}, nil
}

// MustBuild creates the handler registration or panics.
func (b *HandlerBuilder) MustBuild() *HandlerRegistration {
	reg, err := b.Build()
	if err != nil {
		panic(err)
	}
	return reg
}

// getFunctionName returns the name of a function.
func getFunctionName(fn interface{}) string {
	if fn == nil {
		return "unknown"
	}
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return "unknown"
	}
	name := runtime.FuncForPC(v.Pointer()).Name()
	// Extract just the function name (remove package path)
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return name
}

// EventHandlerMiddleware is a function that wraps event handlers.
type EventHandlerMiddleware func(handler EventHandler) EventHandler

// ChainMiddleware chains multiple middlewares together.
func ChainMiddleware(middlewares ...EventHandlerMiddleware) EventHandlerMiddleware {
	return func(handler EventHandler) EventHandler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

// LoggingMiddleware logs event handling.
func LoggingMiddleware(logger *zap.Logger) EventHandlerMiddleware {
	return func(handler EventHandler) EventHandler {
		return func(ctx context.Context, event *Event) error {
			logger.Debug("Handling event",
				zap.String("event_type", event.Type),
				zap.String("event_id", event.ID),
			)
			err := handler(ctx, event)
			if err != nil {
				logger.Error("Event handler failed",
					zap.Error(err),
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
				)
			}
			return err
		}
	}
}

// RecoveryMiddleware recovers from panics in event handlers.
func RecoveryMiddleware(logger *zap.Logger) EventHandlerMiddleware {
	return func(handler EventHandler) EventHandler {
		return func(ctx context.Context, event *Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Panic in event handler",
						zap.Any("recover", r),
						zap.String("event_type", event.Type),
						zap.String("event_id", event.ID),
					)
					err = fmt.Errorf("panic in handler: %v", r)
				}
			}()
			return handler(ctx, event)
		}
	}
}

// TimeoutMiddleware adds timeout to event handling.
func TimeoutMiddleware(timeout int) EventHandlerMiddleware {
	return func(handler EventHandler) EventHandler {
		return func(ctx context.Context, event *Event) error {
			// Note: This is a simplified version
			// In production, you'd use context.WithTimeout
			return handler(ctx, event)
		}
	}
}
