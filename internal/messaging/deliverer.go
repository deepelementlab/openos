package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/messaging/nats"
	"go.uber.org/zap"
)

// Deliverer delivers events from Outbox to NATS.
// It implements the DeliverFunc interface for the Outbox Relay.
type Deliverer struct {
	client    *nats.Client
	publisher *nats.Publisher
	logger    *zap.Logger
}

// NewDeliverer creates a new deliverer.
func NewDeliverer(client *nats.Client, logger *zap.Logger) *Deliverer {
	return &Deliverer{
		client:    client,
		publisher: nats.NewPublisher(client, logger),
		logger:    logger,
	}
}

// Deliver delivers an event to NATS.
// This method implements the DeliverFunc interface used by Outbox Relay.
func (d *Deliverer) Deliver(ctx context.Context, eventID string, eventType string, payload []byte, metadata map[string]string) error {
	// Check if NATS is healthy
	if !d.client.IsHealthy() {
		return fmt.Errorf("NATS connection is not healthy, cannot deliver event %s", eventID)
	}

	// Create NATS message
	msg := &nats.Message{
		ID:          eventID,
		Type:        eventType,
		Payload:     payload,
		Headers:     metadata,
		Timestamp:   time.Now().UTC(),
		ContentType: "application/json",
	}

	// Extract trace context from metadata if present
	if traceID, ok := metadata["trace_id"]; ok {
		msg.SetTraceID(traceID)
	}
	if agentID, ok := metadata["agent_id"]; ok {
		msg.SetAgentID(agentID)
	}
	if tenantID, ok := metadata["tenant_id"]; ok {
		msg.SetTenantID(tenantID)
	}

	// Build topic from event type
	topic := d.buildTopic(eventType)

	// Register topic with default settings
	d.publisher.RegisterTopic(topic, 3, time.Second, 5*time.Second)

	// Publish to NATS
	if err := d.publisher.Publish(ctx, topic, msg); err != nil {
		d.logger.Error("Failed to deliver event to NATS",
			zap.Error(err),
			zap.String("event_id", eventID),
			zap.String("event_type", eventType),
			zap.String("topic", topic),
		)
		return fmt.Errorf("failed to deliver event %s: %w", eventID, err)
	}

	d.logger.Debug("Event delivered to NATS",
		zap.String("event_id", eventID),
		zap.String("event_type", eventType),
		zap.String("topic", topic),
	)

	return nil
}

// buildTopic builds a NATS topic from an event type.
func (d *Deliverer) buildTopic(eventType string) string {
	// Event type format: domain.entity.action
	// Topic format: aos.{domain}.{entity}.{action}.v1
	return fmt.Sprintf("aos.%s.v1", eventType)
}

// Close closes the deliverer.
func (d *Deliverer) Close() {
	if d.publisher != nil {
		d.publisher.Close()
	}
}

// IsHealthy returns true if the deliverer can deliver events.
func (d *Deliverer) IsHealthy() bool {
	return d.client.IsHealthy()
}

// Stats returns deliverer statistics.
func (d *Deliverer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"healthy":          d.IsHealthy(),
		"publisher_stats":  d.publisher.Stats(),
		"nats_client_stats": d.client.Stats(),
	}
}

// DeliverFunc is the function signature for delivering events.
// This matches the interface expected by Outbox Relay.
type DeliverFunc func(ctx context.Context, eventID string, eventType string, payload []byte, metadata map[string]string) error

// ToDeliverFunc converts the Deliverer to a DeliverFunc.
func (d *Deliverer) ToDeliverFunc() DeliverFunc {
	return d.Deliver
}
