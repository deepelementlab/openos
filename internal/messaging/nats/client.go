package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Client wraps a NATS connection with reconnection and JetStream support.
type Client struct {
	config     *ClientConfig
	logger     *zap.Logger
	conn       *nats.Conn
	jetStream  nats.JetStreamContext
	mu         sync.RWMutex
	healthy    bool
}

// NewClient creates a new NATS client with the given configuration.
func NewClient(cfg *ClientConfig, logger *zap.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid NATS configuration: %w", err)
	}

	client := &Client{
		config:  cfg,
		logger:  logger,
		healthy: false,
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// connect establishes the NATS connection with retry logic.
func (c *Client) connect() error {
	opts := []nats.Option{
		nats.Name("agent-os"),
		nats.ReconnectWait(c.config.ReconnectWait),
		nats.MaxReconnects(c.config.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			c.mu.Lock()
			c.healthy = false
			c.mu.Unlock()
			c.logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.mu.Lock()
			c.healthy = true
			c.mu.Unlock()
			c.logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			c.logger.Error("NATS error", zap.Error(err), zap.String("subject", sub.Subject))
		}),
	}

	// Add authentication options
	if c.config.Token != "" {
		opts = append(opts, nats.Token(c.config.Token))
	} else if c.config.CredentialsFile != "" {
		opts = append(opts, nats.UserCredentials(c.config.CredentialsFile))
	}

	// Add TLS if configured
	if c.config.TLS != nil {
		tlsConfig, err := c.config.TLS.BuildTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		if tlsConfig != nil {
			opts = append(opts, nats.Secure(tlsConfig))
		}
	}

	conn, err := nats.Connect(c.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	c.conn = conn
	c.healthy = true

	// Initialize JetStream if enabled
	if c.config.JetStreamEnabled {
		js, err := conn.JetStream(nats.Domain(c.config.JetStreamDomain))
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to initialize JetStream: %w", err)
		}
		c.jetStream = js
		c.logger.Info("JetStream initialized", zap.String("domain", c.config.JetStreamDomain))
	}

	c.logger.Info("NATS client connected", zap.String("url", c.config.URL), zap.Bool("jetstream", c.config.JetStreamEnabled))
	return nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.healthy = false
		c.logger.Info("NATS connection closed")
	}
}

// IsHealthy returns true if the connection is healthy.
func (c *Client) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy && c.conn != nil && c.conn.IsConnected()
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// JetStream returns the JetStream context if enabled.
func (c *Client) JetStream() (nats.JetStreamContext, error) {
	if !c.config.JetStreamEnabled {
		return nil, fmt.Errorf("JetStream is not enabled")
	}
	return c.jetStream, nil
}

// EnsureStream creates or updates a JetStream stream.
func (c *Client) EnsureStream(ctx context.Context, config *nats.StreamConfig) error {
	if !c.config.JetStreamEnabled {
		return fmt.Errorf("JetStream is not enabled")
	}

	_, err := c.jetStream.AddStream(config)
	if err != nil {
		// Try to update if stream already exists
		_, err = c.jetStream.UpdateStream(config)
		if err != nil {
			return fmt.Errorf("failed to create/update stream %s: %w", config.Name, err)
		}
	}

	return nil
}

// EnsureConsumer creates or updates a JetStream consumer.
func (c *Client) EnsureConsumer(ctx context.Context, stream string, config *nats.ConsumerConfig) error {
	if !c.config.JetStreamEnabled {
		return fmt.Errorf("JetStream is not enabled")
	}

	_, err := c.jetStream.AddConsumer(stream, config)
	if err != nil {
		return fmt.Errorf("failed to create consumer %s on stream %s: %w", config.Name, stream, err)
	}

	return nil
}

// Stats returns connection statistics.
func (c *Client) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return map[string]interface{}{
			"connected": false,
		}
	}

	return map[string]interface{}{
		"connected":         c.conn.IsConnected(),
		"healthy":           c.healthy,
		"reconnects":        c.conn.Reconnects,
		"stats":             c.conn.Stats(),
		"jetstream_enabled": c.config.JetStreamEnabled,
	}
}

// WaitForConnection blocks until connected or context timeout.
func (c *Client) WaitForConnection(ctx context.Context) error {
	for {
		if c.IsHealthy() {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for NATS connection: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}
