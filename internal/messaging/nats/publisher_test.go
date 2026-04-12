package nats

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestPublisher() *Publisher {
	return NewPublisher(&Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}, zap.NewNop())
}

func TestNewPublisher(t *testing.T) {
	p := newTestPublisher()
	if p == nil {
		t.Error("expected non-nil publisher")
	}
}

func TestPublisher_RegisterTopic(t *testing.T) {
	p := newTestPublisher()
	defer p.Close()

	p.RegisterTopic("aos.agent.created.v1", 3, time.Second, 5*time.Second)

	stats := p.Stats()
	if stats["registered_topics"] != 1 {
		t.Errorf("expected 1 registered topic, got %v", stats["registered_topics"])
	}
}

func TestPublisher_RegisterMultipleTopics(t *testing.T) {
	p := newTestPublisher()
	defer p.Close()

	p.RegisterTopic("aos.agent.created.v1", 3, time.Second, 5*time.Second)
	p.RegisterTopic("aos.agent.deleted.v1", 5, 2*time.Second, 10*time.Second)

	stats := p.Stats()
	if stats["registered_topics"] != 2 {
		t.Errorf("expected 2 registered topics, got %v", stats["registered_topics"])
	}
}

func TestPublisher_Publish_UnhealthyClient(t *testing.T) {
	p := newTestPublisher()
	defer p.Close()

	msg := NewMessage("agent.created", []byte(`{}`))
	err := p.Publish(context.Background(), "aos.agent.created.v1", msg)
	if err == nil {
		t.Error("expected error when client is not healthy")
	}
}

func TestPublisher_Publish_InvalidMessage(t *testing.T) {
	client := &Client{
		config:  &ClientConfig{URL: "nats://localhost:4222"},
		logger:  zap.NewNop(),
		healthy: true,
	}
	p := NewPublisher(client, zap.NewNop())
	defer p.Close()

	msg := &Message{}
	err := p.Publish(context.Background(), "test", msg)
	if err == nil {
		t.Error("expected error for invalid message")
	}
}

func TestPublisher_PublishAsync_UnhealthyClient(t *testing.T) {
	p := newTestPublisher()
	defer p.Close()

	msg := NewMessage("test", []byte(`{}`))
	errCh, err := p.PublishAsync(context.Background(), "test", msg)
	if err != nil {
		t.Fatalf("PublishAsync failed: %v", err)
	}
	if errCh == nil {
		t.Error("expected error channel")
	}

	select {
	case asyncErr := <-errCh:
		if asyncErr == nil {
			t.Error("expected error from async publish with unhealthy client")
		}
	case <-time.After(5 * time.Second):
		t.Error("timed out waiting for async publish")
	}
}

func TestPublisher_Stats(t *testing.T) {
	p := newTestPublisher()
	stats := p.Stats()
	if stats["registered_topics"] != 0 {
		t.Errorf("expected 0 topics initially, got %v", stats["registered_topics"])
	}
}

func TestPublisher_Close(t *testing.T) {
	p := newTestPublisher()
	p.RegisterTopic("test", 3, time.Second, 5*time.Second)
	p.Close()

	stats := p.Stats()
	if stats["registered_topics"] != 0 {
		t.Errorf("expected 0 topics after close, got %v", stats["registered_topics"])
	}
}

func TestPublisher_Close_Idempotent(t *testing.T) {
	p := newTestPublisher()
	p.Close()
	p.Close()
}
