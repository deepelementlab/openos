package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Publisher provides message publishing capabilities over NATS.
type Publisher struct {
	client     *Client
	logger     *zap.Logger
	mu         sync.RWMutex
	publishers map[string]*streamPublisher // topic -> publisher
}

// streamPublisher manages publishing to a specific stream/topic.
type streamPublisher struct {
	topic          string
	maxRetries     int
	retryBackoff   time.Duration
	publishTimeout time.Duration
}

// NewPublisher creates a new NATS publisher.
func NewPublisher(client *Client, logger *zap.Logger) *Publisher {
	return &Publisher{
		client:     client,
		logger:     logger,
		publishers: make(map[string]*streamPublisher),
	}
}

// RegisterTopic registers a topic for publishing with specific settings.
func (p *Publisher) RegisterTopic(topic string, maxRetries int, retryBackoff, timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.publishers[topic] = &streamPublisher{
		topic:          topic,
		maxRetries:     maxRetries,
		retryBackoff:   retryBackoff,
		publishTimeout: timeout,
	}

	p.logger.Info("Registered NATS publisher topic", zap.String("topic", topic))
}

// Publish publishes a message to the specified topic.
func (p *Publisher) Publish(ctx context.Context, topic string, msg *Message) error {
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	if !p.client.IsHealthy() {
		return fmt.Errorf("NATS connection is not healthy")
	}

	// Get or create publisher settings for this topic
	p.mu.RLock()
	sp, exists := p.publishers[topic]
	p.mu.RUnlock()

	if !exists {
		// Use default settings
		sp = &streamPublisher{
			topic:          topic,
			maxRetries:     3,
			retryBackoff:   time.Second,
			publishTimeout: 5 * time.Second,
		}
	}

	// Serialize message
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Prepare NATS headers
	natsHeaders := nats.Header{}
	for k, v := range msg.Headers {
		natsHeaders.Add(k, v)
	}
	natsHeaders.Set("Content-Type", msg.ContentType)
	natsHeaders.Set("Message-ID", msg.ID)
	natsHeaders.Set("Message-Type", msg.Type)

	// Publish with retry
	return p.publishWithRetry(ctx, sp, data, natsHeaders)
}

// publishWithRetry attempts to publish with retries.
func (p *Publisher) publishWithRetry(ctx context.Context, sp *streamPublisher, data []byte, headers nats.Header) error {
	var lastErr error

	for attempt := 0; attempt <= sp.maxRetries; attempt++ {
		if attempt > 0 {
			p.logger.Warn("Retrying NATS publish",
				zap.String("topic", sp.topic),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
			time.Sleep(sp.retryBackoff * time.Duration(attempt))
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("publish context cancelled: %w", ctx.Err())
		default:
		}

		// Create timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, sp.publishTimeout)
		defer cancel()

		// Attempt publish
		err := p.doPublish(attemptCtx, sp.topic, data, headers)
		if err == nil {
			p.logger.Debug("Published message to NATS",
				zap.String("topic", sp.topic),
				zap.Int("size", len(data)),
			)
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed to publish after %d retries: %w", sp.maxRetries, lastErr)
}

// doPublish performs the actual NATS publish.
func (p *Publisher) doPublish(ctx context.Context, topic string, data []byte, headers nats.Header) error {
	msg := &nats.Msg{
		Subject: topic,
		Data:    data,
		Header:  headers,
	}

	// Try JetStream first if enabled
	if p.client.config.JetStreamEnabled {
		js, err := p.client.JetStream()
		if err == nil {
			_, err = js.PublishMsg(msg, nats.Context(ctx))
			if err != nil {
				return fmt.Errorf("jetstream publish failed: %w", err)
			}
			return nil
		}
	}

	// Fall back to core NATS
	err := p.client.Conn().PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("nats publish failed: %w", err)
	}

	return nil
}

// PublishAsync publishes a message asynchronously.
func (p *Publisher) PublishAsync(ctx context.Context, topic string, msg *Message) (<-chan error, error) {
	errCh := make(chan error, 1)

	go func() {
		errCh <- p.Publish(ctx, topic, msg)
	}()

	return errCh, nil
}

// RequestReply publishes a message and waits for a reply.
func (p *Publisher) RequestReply(ctx context.Context, topic string, msg *Message, timeout time.Duration) (*Message, error) {
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}

	if !p.client.IsHealthy() {
		return nil, fmt.Errorf("NATS connection is not healthy")
	}

	// Serialize message
	data, err := msg.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Prepare NATS headers
	natsHeaders := nats.Header{}
	for k, v := range msg.Headers {
		natsHeaders.Add(k, v)
	}
	natsHeaders.Set("Content-Type", msg.ContentType)
	natsHeaders.Set("Message-ID", msg.ID)
	natsHeaders.Set("Message-Type", msg.Type)

	// Create NATS message
	natsMsg := &nats.Msg{
		Subject: topic,
		Data:    data,
		Header:  natsHeaders,
	}

	// Send request
	reply, err := p.client.Conn().RequestMsg(natsMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Parse reply
	replyMsg := &Message{}
	if err := replyMsg.FromJSON(reply.Data); err != nil {
		return nil, fmt.Errorf("failed to parse reply: %w", err)
	}

	return replyMsg, nil
}

// Close cleans up the publisher.
func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.publishers = make(map[string]*streamPublisher)
	p.logger.Info("NATS publisher closed")
}

// Stats returns publisher statistics.
func (p *Publisher) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"registered_topics": len(p.publishers),
	}
}
