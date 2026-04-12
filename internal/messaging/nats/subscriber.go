package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Handler is a function that handles incoming messages.
type Handler func(ctx context.Context, msg *Message) error

// Subscription represents a NATS subscription.
type Subscription struct {
	topic        string
	queueGroup   string
	handler      Handler
	sub          *nats.Subscription
	jsSub        *nats.Subscription // JetStream subscription
	maxInFlight  int
	ackWait      time.Duration
}

// Subscriber provides message subscription capabilities over NATS.
type Subscriber struct {
	client        *Client
	logger        *zap.Logger
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewSubscriber creates a new NATS subscriber.
func NewSubscriber(client *Client, logger *zap.Logger) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscriber{
		client:        client,
		logger:        logger,
		subscriptions: make(map[string]*Subscription),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Subscribe creates a subscription to a topic.
func (s *Subscriber) Subscribe(ctx context.Context, topic string, handler Handler) error {
	return s.subscribe(ctx, topic, "", handler, 1, 0)
}

// SubscribeQueue creates a queue subscription for load balancing.
func (s *Subscriber) SubscribeQueue(ctx context.Context, topic string, queueGroup string, handler Handler) error {
	return s.subscribe(ctx, topic, queueGroup, handler, 1, 0)
}

// SubscribeJetStream creates a JetStream subscription with durable consumer.
func (s *Subscriber) SubscribeJetStream(ctx context.Context, topic, durable string, handler Handler) error {
	return s.subscribeJetStream(ctx, topic, durable, handler, 10, 30*time.Second)
}

// subscribe is the internal subscription implementation.
func (s *Subscriber) subscribe(ctx context.Context, topic, queueGroup string, handler Handler, maxInFlight int, ackWait time.Duration) error {
	if !s.client.IsHealthy() {
		return fmt.Errorf("NATS connection is not healthy")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already subscribed
	if _, exists := s.subscriptions[topic]; exists {
		return fmt.Errorf("already subscribed to topic: %s", topic)
	}

	sub := &Subscription{
		topic:       topic,
		queueGroup:  queueGroup,
		handler:     handler,
		maxInFlight: maxInFlight,
		ackWait:     ackWait,
	}

	// Create NATS handler
	natsHandler := func(msg *nats.Msg) {
		s.handleMessage(msg, handler)
	}

	var err error
	if queueGroup != "" {
		sub.sub, err = s.client.Conn().QueueSubscribe(topic, queueGroup, natsHandler)
	} else {
		sub.sub, err = s.client.Conn().Subscribe(topic, natsHandler)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
	}

	s.subscriptions[topic] = sub
	s.logger.Info("Subscribed to NATS topic",
		zap.String("topic", topic),
		zap.String("queue_group", queueGroup),
	)

	return nil
}

// subscribeJetStream creates a JetStream subscription.
func (s *Subscriber) subscribeJetStream(ctx context.Context, topic, durable string, handler Handler, maxInFlight int, ackWait time.Duration) error {
	if !s.client.IsHealthy() {
		return fmt.Errorf("NATS connection is not healthy")
	}

	if !s.client.config.JetStreamEnabled {
		return fmt.Errorf("JetStream is not enabled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already subscribed
	subKey := topic + ":" + durable
	if _, exists := s.subscriptions[subKey]; exists {
		return fmt.Errorf("already subscribed to JetStream topic: %s with durable: %s", topic, durable)
	}

	js, err := s.client.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	sub := &Subscription{
		topic:       topic,
		handler:     handler,
		maxInFlight: maxInFlight,
		ackWait:     ackWait,
	}

	// Create NATS handler
	natsHandler := func(msg *nats.Msg) {
		s.handleJetStreamMessage(msg, handler)
	}

	// Subscribe with durable consumer
	sub.jsSub, err = js.Subscribe(topic, natsHandler,
		nats.Durable(durable),
		nats.MaxAckPending(maxInFlight),
		nats.AckWait(ackWait),
	)
	if err != nil {
		return fmt.Errorf("failed to create JetStream subscription: %w", err)
	}

	s.subscriptions[subKey] = sub
	s.logger.Info("Subscribed to JetStream topic",
		zap.String("topic", topic),
		zap.String("durable", durable),
	)

	return nil
}

// handleMessage processes a core NATS message.
func (s *Subscriber) handleMessage(natsMsg *nats.Msg, handler Handler) {
	s.wg.Add(1)
	defer s.wg.Done()

	// Parse message
	msg := &Message{}
	if err := msg.FromJSON(natsMsg.Data); err != nil {
		s.logger.Error("Failed to parse message",
			zap.Error(err),
			zap.String("subject", natsMsg.Subject),
		)
		return
	}

	// Extract headers
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	for k, v := range natsMsg.Header {
		if len(v) > 0 {
			msg.Headers[k] = v[0]
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Call handler
	if err := handler(ctx, msg); err != nil {
		s.logger.Error("Message handler failed",
			zap.Error(err),
			zap.String("topic", natsMsg.Subject),
			zap.String("message_id", msg.ID),
		)
	}
}

// handleJetStreamMessage processes a JetStream message.
func (s *Subscriber) handleJetStreamMessage(natsMsg *nats.Msg, handler Handler) {
	s.wg.Add(1)
	defer s.wg.Done()

	// Parse message
	msg := &Message{}
	if err := msg.FromJSON(natsMsg.Data); err != nil {
		s.logger.Error("Failed to parse JetStream message",
			zap.Error(err),
			zap.String("subject", natsMsg.Subject),
		)
		// Negative acknowledge
		natsMsg.Nak()
		return
	}

	// Extract headers
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	for k, v := range natsMsg.Header {
		if len(v) > 0 {
			msg.Headers[k] = v[0]
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Call handler
	if err := handler(ctx, msg); err != nil {
		s.logger.Error("JetStream message handler failed",
			zap.Error(err),
			zap.String("topic", natsMsg.Subject),
			zap.String("message_id", msg.ID),
		)
		// Negative acknowledge for retry
		natsMsg.Nak()
		return
	}

	// Acknowledge successful processing
	if err := natsMsg.Ack(); err != nil {
		s.logger.Error("Failed to acknowledge message",
			zap.Error(err),
			zap.String("message_id", msg.ID),
		)
	}
}

// Unsubscribe removes a subscription.
func (s *Subscriber) Unsubscribe(topic string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[topic]
	if !exists {
		return fmt.Errorf("no subscription found for topic: %s", topic)
	}

	var err error
	if sub.jsSub != nil {
		err = sub.jsSub.Unsubscribe()
	} else if sub.sub != nil {
		err = sub.sub.Unsubscribe()
	}

	delete(s.subscriptions, topic)
	s.logger.Info("Unsubscribed from NATS topic", zap.String("topic", topic))

	return err
}

// Close shuts down the subscriber and all subscriptions.
func (s *Subscriber) Close() {
	s.cancel()

	s.mu.Lock()
	subs := make([]*Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	// Unsubscribe all
	for _, sub := range subs {
		if sub.jsSub != nil {
			sub.jsSub.Unsubscribe()
		} else if sub.sub != nil {
			sub.sub.Unsubscribe()
		}
	}

	// Wait for all handlers to complete
	s.wg.Wait()

	s.logger.Info("NATS subscriber closed")
}

// Stats returns subscriber statistics.
func (s *Subscriber) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_subscriptions": len(s.subscriptions),
		"topics":               s.getTopics(),
	}
}

// getTopics returns the list of subscribed topics.
func (s *Subscriber) getTopics() []string {
	topics := make([]string, 0, len(s.subscriptions))
	for topic := range s.subscriptions {
		topics = append(topics, topic)
	}
	return topics
}

// IsSubscribed checks if already subscribed to a topic.
func (s *Subscriber) IsSubscribed(topic string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.subscriptions[topic]
	return exists
}
