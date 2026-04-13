package nats

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecompressPayloadIfNeeded(t *testing.T) {
	m := NewMessage("evt", bytes.Repeat([]byte("x"), 2048))
	b := &Batcher{minCompress: 1024}
	b.maybeCompress(m)
	require.Equal(t, "snappy", m.Headers[headerCompression])
	require.NoError(t, DecompressPayloadIfNeeded(m))
	require.GreaterOrEqual(t, len(m.Payload), 2048)
}
