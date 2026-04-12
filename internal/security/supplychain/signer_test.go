package supplychain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPayload(t *testing.T) {
	ctx := context.Background()
	payload := []byte("hello")
	s, err := NewSigner("k").SignPayload(ctx, payload)
	require.NoError(t, err)
	require.NoError(t, VerifyPayload(ctx, payload, s))
	require.Error(t, VerifyPayload(ctx, payload, "bad"))
}
