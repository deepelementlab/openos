package ipc

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInMemoryManager_MQSemShm(t *testing.T) {
	ctx := context.Background()
	m := NewInMemoryManager()

	q, err := m.CreateMessageQueue(ctx, "q1", MQConfig{})
	require.NoError(t, err)
	var n int32
	sub, err := q.Subscribe(ctx, func(ctx context.Context, msg Message) error {
		atomic.AddInt32(&n, 1)
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, q.Publish(ctx, Message{Topic: "t", Payload: []byte("x")}))
	require.Equal(t, int32(1), atomic.LoadInt32(&n))
	require.NoError(t, sub.Unsubscribe())

	sem, err := m.CreateSemaphore(ctx, "s1", 1)
	require.NoError(t, err)
	require.NoError(t, sem.Acquire(ctx, true))
	require.NoError(t, sem.Release())

	shm, err := m.CreateSharedMemory(ctx, "shm1", 16)
	require.NoError(t, err)
	_, err = shm.WriteAt([]byte("hi"), 0)
	require.NoError(t, err)
	_, err = m.AttachShm(ctx, "a1", "shm1")
	require.NoError(t, err)
}
