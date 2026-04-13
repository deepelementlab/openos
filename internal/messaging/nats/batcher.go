package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/klauspost/compress/snappy"
	"go.uber.org/zap"
)

const (
	headerCompression = "aos-compression"
	compressSnappy    = "snappy"
)

// Batcher aggregates small NATS publishes and optionally compresses payloads (Snappy).
type Batcher struct {
	publisher   *Publisher
	topic       string
	maxBatch    int
	maxWait     time.Duration
	minCompress int // compress when serialized JSON >= this many bytes
	logger      *zap.Logger

	mu    sync.Mutex
	buf   []*Message
	timer *time.Timer
	flush chan struct{}
}

// NewBatcher creates a batcher. maxBatch>=2 recommended; maxWait bounds latency.
func NewBatcher(p *Publisher, topic string, maxBatch int, maxWait time.Duration, minCompress int, logger *zap.Logger) *Batcher {
	if maxBatch < 1 {
		maxBatch = 10
	}
	if maxWait <= 0 {
		maxWait = 5 * time.Millisecond
	}
	if minCompress <= 0 {
		minCompress = 1024
	}
	return &Batcher{
		publisher:   p,
		topic:       topic,
		maxBatch:    maxBatch,
		maxWait:     maxWait,
		minCompress: minCompress,
		logger:      logger,
		flush:       make(chan struct{}, 1),
	}
}

// Enqueue adds a message; flushes when batch full. Caller must not mutate msg after Enqueue.
func (b *Batcher) Enqueue(ctx context.Context, msg *Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}
	b.mu.Lock()
	b.buf = append(b.buf, msg)
	n := len(b.buf)
	needTimer := n == 1
	if needTimer {
		if b.timer != nil {
			b.timer.Stop()
		}
		b.timer = time.AfterFunc(b.maxWait, func() { b.signalFlush() })
	}
	if n >= b.maxBatch {
		msgs := b.drainLocked()
		b.mu.Unlock()
		return b.publishCompressedBatch(ctx, msgs)
	}
	b.mu.Unlock()
	return nil
}

func (b *Batcher) signalFlush() {
	select {
	case b.flush <- struct{}{}:
	default:
	}
}

// Flush sends any pending messages immediately.
func (b *Batcher) Flush(ctx context.Context) error {
	b.mu.Lock()
	msgs := b.drainLocked()
	b.mu.Unlock()
	if len(msgs) == 0 {
		return nil
	}
	return b.publishCompressedBatch(ctx, msgs)
}

func (b *Batcher) drainLocked() []*Message {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	out := b.buf
	b.buf = nil
	return out
}

// Run processes delayed flush signals until ctx done.
func (b *Batcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = b.Flush(context.Background())
			return
		case <-b.flush:
			_ = b.Flush(ctx)
		}
	}
}

func (b *Batcher) publishCompressedBatch(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}
	for _, m := range msgs {
		b.maybeCompress(m)
	}
	return b.publisher.PublishBatch(ctx, b.topic, msgs)
}

func (b *Batcher) maybeCompress(m *Message) {
	if m == nil || len(m.Payload) < b.minCompress {
		return
	}
	enc := snappy.Encode(nil, m.Payload)
	if len(enc) >= len(m.Payload) {
		return
	}
	m.Payload = enc
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[headerCompression] = compressSnappy
}

// DecompressPayloadIfNeeded restores payload after consume (subscriber side).
func DecompressPayloadIfNeeded(m *Message) error {
	if m == nil || m.Headers == nil {
		return nil
	}
	if m.Headers[headerCompression] != compressSnappy {
		return nil
	}
	dec, err := snappy.Decode(nil, m.Payload)
	if err != nil {
		return fmt.Errorf("snappy decode: %w", err)
	}
	m.Payload = dec
	delete(m.Headers, headerCompression)
	return nil
}
