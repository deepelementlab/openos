package nats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPublisher_PublishBatch_Empty(t *testing.T) {
	p := NewPublisher(&Client{}, zap.NewNop())
	err := p.PublishBatch(context.Background(), "t", nil)
	require.NoError(t, err)
}
