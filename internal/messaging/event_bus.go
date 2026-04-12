package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/messaging/nats"
	"go.uber.org/zap"
)

// EventBus provides publish/subscribe messaging capabilities.
type EventBus interface {
	// Publish publishes an event to the bus.
	Publish(ctx context.Context, event *Event) error

	// Subscribe creates a subscription for a specific event type.
	Subscribe(ctx context.Context, eventType string, handler EventHandler) (Subscription, error)

	// SubscribeMultiple creates a subscription for multiple event types.
	SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) (Subscription, error)

	// SubscribeAll creates a subscription for all events.
	SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error)

	// Close shuts down the event bus.
	Close() error
}

// EventHandler is a function that handles events.
type EventHandler func(ctx context.Context, event *Event) error

// Subscription represents an event subscription.
type Subscription interface {
	// Unsubscribe removes the subscription.
	Unsubscribe() error

	// EventTypes returns the subscribed event types.
	EventTypes() []string
}

// subscription implements Subscription.
type subscription struct {
	id         string
	eventTypes []string
	handler    EventHandler
	unsubFn    func() error
}

// Unsubscribe removes the subscription.
func (s *subscription) Unsubscribe() error {
	return s.unsubFn()
}

// EventTypes returns the subscribed event types.
func (s *subscription) EventTypes() []string {
	return s.eventTypes
}

// NATSEventBus implements EventBus using NATS.
type NATSEventBus struct {
	client     *nats.Client
	publisher  *nats.Publisher
	subscriber *nats.Subscriber
	router     *EventRouter
	config     *config.MessagingConfig
	logger     *zap.Logger
	mu         sync.RWMutex
	subs       map[string]*subscription
	closed     bool
}

// NewNATSEventBus creates a new NATS-based event bus.
func NewNATSEventBus(cfg *config.Config, logger *zap.Logger) (*NATSEventBus, error) {
	if !cfg.Messaging.Enabled {
		return nil, fmt.Errorf("messaging is not enabled")
	}

	// Create NATS client
	clientCfg := nats.NewClientConfig(cfg)
	client, err := nats.NewClient(clientCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	// Create publisher and subscriber
	publisher := nats.NewPublisher(client, logger)
	subscriber := nats.NewSubscriber(client, logger)

	// Create event router
	router := NewEventRouter(logger)

	bus := &NATSEventBus{
		client:     client,
		publisher:  publisher,
		subscriber: subscriber,
		router:     router,
		config:     &cfg.Messaging,
		logger:     logger,
		subs:       make(map[string]*subscription),
	}

	return bus, nil
}

// Publish publishes an event to the bus.
func (b *NATSEventBus) Publish(ctx context.Context, event *Event) error {
	if b.closed {
		return fmt.Errorf("event bus is closed")
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Convert to NATS message
	msg := &nats.Message{
		ID:          event.ID,
		Type:        event.Type,
		Payload:     event.Payload,
		Timestamp:   event.OccurredAt,
		ContentType: "application/json",
	}

	// Set headers for tracing and multi-tenancy
	if event.TraceID != "" {
		msg.SetTraceID(event.TraceID)
	}
	if event.AgentID != "" {
		msg.SetAgentID(event.AgentID)
	}
	if event.TenantID != "" {
		msg.SetTenantID(event.TenantID)
	}

	// Set additional metadata
	msg.SetHeader("schema_version", event.SchemaVersion)
	for k, v := range event.Metadata {
		msg.SetHeader(k, v)
	}

	// Build topic from event type
	topic := b.buildTopic(event.Type)

	// Publish to NATS
	if err := b.publisher.Publish(ctx, topic, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	b.logger.Debug("Event published",
		zap.String("event_type", event.Type),
		zap.String("event_id", event.ID),
		zap.String("topic", topic),
	)

	return nil
}

// Subscribe creates a subscription for a specific event type.
func (b *NATSEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) (Subscription, error) {
	return b.SubscribeMultiple(ctx, []string{eventType}, handler)
}

// SubscribeMultiple creates a subscription for multiple event types.
func (b *NATSEventBus) SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) (Subscription, error) {
	if b.closed {
		return nil, fmt.Errorf("event bus is closed")
	}

	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("at least one event type is required")
	}

	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	// Create subscription
	sub := &subscription{
		id:         subID,
		eventTypes: eventTypes,
		handler:    handler,
		unsubFn: func() error {
			return b.unsubscribe(subID)
		},
	}

	// Register with router
	for _, eventType := range eventTypes {
		topic := b.buildTopic(eventType)
		if err := b.router.RegisterHandler(eventType, handler); err != nil {
			return nil, err
		}

		// Subscribe to NATS topic if not already subscribed
		if !b.subscriber.IsSubscribed(topic) {
			natsHandler := b.createNATSHandler()
			if err := b.subscriber.SubscribeQueue(ctx, topic, "agentos-eventbus", natsHandler); err != nil {
				return nil, fmt.Errorf("failed to subscribe to %s: %w", topic, err)
			}
		}
	}

	b.mu.Lock()
	b.subs[subID] = sub
	b.mu.Unlock()

	b.logger.Info("Subscribed to events",
		zap.String("subscription_id", subID),
		zap.Strings("event_types", eventTypes),
	)

	return sub, nil
}

// SubscribeAll creates a subscription for all events.
func (b *NATSEventBus) SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error) {
	// Subscribe to all events using wildcard topic
	if b.closed {
		return nil, fmt.Errorf("event bus is closed")
	}

	subID := fmt.Sprintf("sub-all-%d", time.Now().UnixNano())

	// Create subscription
	sub := &subscription{
		id:         subID,
		eventTypes: []string{"*"},
		handler:    handler,
		unsubFn: func() error {
			return b.unsubscribe(subID)
		},
	}

	// Subscribe to wildcard topic
	wildcardTopic := "aos.*.*.*.*"
	natsHandler := b.createNATSHandler()
	if err := b.subscriber.SubscribeQueue(ctx, wildcardTopic, "agentos-eventbus-all", natsHandler); err != nil {
		return nil, fmt.Errorf("failed to subscribe to all events: %w", err)
	}

	b.mu.Lock()
	b.subs[subID] = sub
	b.mu.Unlock()

	b.logger.Info("Subscribed to all events",
		zap.String("subscription_id", subID),
	)

	return sub, nil
}

// createNATSHandler creates a NATS message handler that routes to EventBus handlers.
func (b *NATSEventBus) createNATSHandler() nats.Handler {
	return func(ctx context.Context, msg *nats.Message) error {
		// Convert NATS message to Event
		event := &Event{
			ID:            msg.ID,
			Type:          msg.Type,
			SchemaVersion: msg.GetHeader("schema_version"),
			OccurredAt:    msg.Timestamp,
			TraceID:       msg.GetTraceID(),
			AgentID:       msg.GetAgentID(),
			TenantID:      msg.GetTenantID(),
			Payload:       msg.Payload,
			Metadata:      msg.Headers,
		}

		// Route to handlers
		return b.router.Route(ctx, event)
	}
}

// unsubscribe removes a subscription.
func (b *NATSEventBus) unsubscribe(subID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subs[subID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subID)
	}

	// Unregister handlers from router
	for _, eventType := range sub.eventTypes {
		b.router.UnregisterHandler(eventType, sub.handler)
	}

	delete(b.subs, subID)

	b.logger.Info("Unsubscribed from events",
		zap.String("subscription_id", subID),
	)

	return nil
}

// buildTopic builds a NATS topic from an event type.
func (b *NATSEventBus) buildTopic(eventType string) string {
	// Event type format: domain.entity.action
	// Topic format: aos.{domain}.{entity}.{action}.v1
	return fmt.Sprintf("aos.%s.v1", eventType)
}

// Close shuts down the event bus.
func (b *NATSEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	// Close subscriber
	b.subscriber.Close()

	// Close publisher
	b.publisher.Close()

	// Close NATS client
	b.client.Close()

	b.logger.Info("Event bus closed")
	return nil
}

// IsHealthy returns true if the event bus is healthy.
func (b *NATSEventBus) IsHealthy() bool {
	return b.client.IsHealthy()
}

// Stats returns event bus statistics.
func (b *NATSEventBus) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"healthy":       b.IsHealthy(),
		"subscriptions": len(b.subs),
		"client_stats":  b.client.Stats(),
	}
}

// InMemoryEventBus implements EventBus for testing without NATS.
type InMemoryEventBus struct {
	router  *EventRouter
	logger  *zap.Logger
	mu      sync.RWMutex
	subs    map[string]*subscription
	closed  bool
}

// NewInMemoryEventBus creates an in-memory event bus for testing.
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		router: NewEventRouter(logger),
		logger: logger,
		subs:   make(map[string]*subscription),
	}
}

// Publish publishes an event to the in-memory bus.
func (b *InMemoryEventBus) Publish(ctx context.Context, event *Event) error {
	if b.closed {
		return fmt.Errorf("event bus is closed")
	}

	if err := event.Validate(); err != nil {
		return err
	}

	return b.router.Route(ctx, event)
}

// Subscribe creates a subscription.
func (b *InMemoryEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) (Subscription, error) {
	return b.SubscribeMultiple(ctx, []string{eventType}, handler)
}

// SubscribeMultiple creates a subscription for multiple event types.
func (b *InMemoryEventBus) SubscribeMultiple(ctx context.Context, eventTypes []string, handler EventHandler) (Subscription, error) {
	if b.closed {
		return nil, fmt.Errorf("event bus is closed")
	}

	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	sub := &subscription{
		id:         subID,
		eventTypes: eventTypes,
		handler:    handler,
		unsubFn: func() error {
			return b.unsubscribe(subID)
		},
	}

	for _, eventType := range eventTypes {
		if err := b.router.RegisterHandler(eventType, handler); err != nil {
			return nil, err
		}
	}

	b.mu.Lock()
	b.subs[subID] = sub
	b.mu.Unlock()

	return sub, nil
}

// SubscribeAll creates a subscription for all events.
func (b *InMemoryEventBus) SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error) {
	// Not implemented for in-memory bus
	return nil, fmt.Errorf("SubscribeAll not supported in in-memory bus")
}

// unsubscribe removes a subscription.
func (b *InMemoryEventBus) unsubscribe(subID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subs[subID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subID)
	}

	for _, eventType := range sub.eventTypes {
		b.router.UnregisterHandler(eventType, sub.handler)
	}

	delete(b.subs, subID)
	return nil
}

// Close shuts down the in-memory bus.
func (b *InMemoryEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	return nil
}
