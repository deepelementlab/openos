package nats

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func newTestSubscriber() *Subscriber {
	return NewSubscriber(&Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}, zap.NewNop())
}

func TestNewSubscriber(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()
	if s == nil {
		t.Error("expected non-nil subscriber")
	}
}

func TestSubscriber_IsSubscribed(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()
	if s.IsSubscribed("test") {
		t.Error("expected not subscribed initially")
	}
}

func TestSubscriber_Subscribe_UnhealthyClient(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()

	handler := func(ctx context.Context, msg *Message) error { return nil }
	err := s.Subscribe(context.Background(), "aos.agent.created.v1", handler)
	if err == nil {
		t.Error("expected error when client is not healthy")
	}
}

func TestSubscriber_SubscribeQueue_UnhealthyClient(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()

	handler := func(ctx context.Context, msg *Message) error { return nil }
	err := s.SubscribeQueue(context.Background(), "aos.agent.created.v1", "queue-group", handler)
	if err == nil {
		t.Error("expected error when client is not healthy")
	}
}

func TestSubscriber_SubscribeJetStream_UnhealthyClient(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()

	handler := func(ctx context.Context, msg *Message) error { return nil }
	err := s.SubscribeJetStream(context.Background(), "aos.agent.created.v1", "durable-name", handler)
	if err == nil {
		t.Error("expected error when client is not healthy")
	}
}

func TestSubscriber_SubscribeJetStream_DisabledJetStream(t *testing.T) {
	client := &Client{
		config:  &ClientConfig{URL: "nats://localhost:4222", JetStreamEnabled: false},
		logger:  zap.NewNop(),
		healthy: true,
	}
	s := NewSubscriber(client, zap.NewNop())
	defer s.Close()

	handler := func(ctx context.Context, msg *Message) error { return nil }
	err := s.SubscribeJetStream(context.Background(), "test", "durable", handler)
	if err == nil {
		t.Error("expected error when JetStream is not enabled")
	}
}

func TestSubscriber_Unsubscribe_NotSubscribed(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()

	err := s.Unsubscribe("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent subscription")
	}
}

func TestSubscriber_Stats(t *testing.T) {
	s := newTestSubscriber()
	defer s.Close()

	stats := s.Stats()
	if stats["active_subscriptions"] != 0 {
		t.Errorf("expected 0 subscriptions, got %v", stats["active_subscriptions"])
	}
	topics, ok := stats["topics"].([]string)
	if !ok || len(topics) != 0 {
		t.Errorf("expected empty topics, got %v", stats["topics"])
	}
}

func TestSubscriber_Close(t *testing.T) {
	s := newTestSubscriber()
	s.Close()
}

func TestSubscriber_Close_Idempotent(t *testing.T) {
	s := newTestSubscriber()
	s.Close()
	s.Close()
}

func TestSubscriber_HandlerFunction(t *testing.T) {
	var received *Message
	handler := func(ctx context.Context, msg *Message) error {
		received = msg
		return nil
	}

	msg := NewMessage("test", []byte(`{"data":1}`))
	if err := handler(context.Background(), msg); err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if received == nil || received.Type != "test" {
		t.Error("handler did not capture message")
	}
}

func TestSubscriber_HandlerFunction_Error(t *testing.T) {
	handler := func(ctx context.Context, msg *Message) error {
		return context.Canceled
	}
	msg := NewMessage("test", nil)
	if err := handler(context.Background(), msg); err == nil {
		t.Error("expected error from handler")
	}
}
