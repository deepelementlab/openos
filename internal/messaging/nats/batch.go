package nats

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// PublishBatch publishes multiple validated messages to the same topic in one call path,
// reducing lock round-trips and allowing future JetStream bulk APIs.
func (p *Publisher) PublishBatch(ctx context.Context, topic string, msgs []*Message) error {
	if p == nil {
		return fmt.Errorf("nats: nil publisher")
	}
	if len(msgs) == 0 {
		return nil
	}
	if !p.client.IsHealthy() {
		return fmt.Errorf("NATS connection is not healthy")
	}
	for i, m := range msgs {
		if err := p.Publish(ctx, topic, m); err != nil {
			if p.logger != nil {
				p.logger.Warn("batch publish failed", zap.Int("index", i), zap.Error(err))
			}
			return fmt.Errorf("batch item %d: %w", i, err)
		}
	}
	return nil
}
