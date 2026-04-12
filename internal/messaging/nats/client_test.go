package nats

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestClient_IsHealthy_ZeroValue(t *testing.T) {
	client := &Client{
		config:  &ClientConfig{URL: "nats://localhost:4222"},
		logger:  zap.NewNop(),
		healthy: false,
	}
	if client.IsHealthy() {
		t.Error("expected unhealthy for zero-value client")
	}
}

func TestClient_IsHealthy_WithHealthyFlag(t *testing.T) {
	client := &Client{
		config:  &ClientConfig{URL: "nats://localhost:4222"},
		logger:  zap.NewNop(),
		healthy: true,
	}
	if client.IsHealthy() {
		t.Error("expected unhealthy because conn is nil even with healthy=true")
	}
}

func TestClient_Stats_NilConn(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	stats := client.Stats()
	if stats["connected"] != false {
		t.Error("expected connected=false for nil conn")
	}
}

func TestClient_JetStream_NotEnabled(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	_, err := client.JetStream()
	if err == nil {
		t.Error("expected error when JetStream is not enabled")
	}
}

func TestClient_JetStream_EnabledButNil(t *testing.T) {
	client := &Client{
		config: &ClientConfig{
			URL:              "nats://localhost:4222",
			JetStreamEnabled: true,
		},
		logger: zap.NewNop(),
	}
	_, err := client.JetStream()
	if err != nil {
		t.Errorf("expected no error when JetStream enabled, got: %v", err)
	}
}

func TestClient_Close_NilConn(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	client.Close()
}

func TestClient_Close_Idempotent(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	client.Close()
	client.Close()
}

func TestClient_WaitForConnection_CancelledContext(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.WaitForConnection(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestClient_EnsureStream_JetStreamDisabled(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	err := client.EnsureStream(context.Background(), nil)
	if err == nil {
		t.Error("expected error when JetStream is not enabled")
	}
}

func TestClient_EnsureConsumer_JetStreamDisabled(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	err := client.EnsureConsumer(context.Background(), "stream", nil)
	if err == nil {
		t.Error("expected error when JetStream is not enabled")
	}
}

func TestClient_Conn_Nil(t *testing.T) {
	client := &Client{
		config: &ClientConfig{URL: "nats://localhost:4222"},
		logger: zap.NewNop(),
	}
	if client.Conn() != nil {
		t.Error("expected nil conn")
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	_, err := NewClient(&ClientConfig{URL: ""}, zap.NewNop())
	if err == nil {
		t.Error("expected error for empty URL")
	}
}
