package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage_PutAndGet(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	err := s.Put(ctx, "test-key", []byte("hello world"), "text/plain")
	require.NoError(t, err)

	data, err := s.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), data)
}

func TestInMemoryStorage_GetNotFound(t *testing.T) {
	s := NewInMemoryStorage()
	_, err := s.Get(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryStorage_PutEmptyKey(t *testing.T) {
	s := NewInMemoryStorage()
	err := s.Put(context.Background(), "", []byte("data"), "text/plain")
	assert.Error(t, err)
}

func TestInMemoryStorage_Delete(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	_ = s.Put(ctx, "key1", []byte("data1"), "text/plain")
	err := s.Delete(ctx, "key1")
	require.NoError(t, err)

	_, err = s.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestInMemoryStorage_DeleteNotFound(t *testing.T) {
	s := NewInMemoryStorage()
	err := s.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryStorage_List(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	_ = s.Put(ctx, "agents/config1", []byte("c1"), "application/json")
	_ = s.Put(ctx, "agents/config2", []byte("c2"), "application/json")
	_ = s.Put(ctx, "logs/log1.txt", []byte("l1"), "text/plain")

	items, err := s.List(ctx, ListOptions{Prefix: "agents/"})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestInMemoryStorage_ListWithLimit(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = s.Put(ctx, "key-"+string(rune('0'+i)), []byte("data"), "text/plain")
	}

	items, err := s.List(ctx, ListOptions{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestInMemoryStorage_ListWithOffset(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	_ = s.Put(ctx, "a", []byte("1"), "text/plain")
	_ = s.Put(ctx, "b", []byte("2"), "text/plain")
	_ = s.Put(ctx, "c", []byte("3"), "text/plain")

	items, err := s.List(ctx, ListOptions{Offset: 1})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestInMemoryStorage_ListOffsetBeyondRange(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()
	_ = s.Put(ctx, "a", []byte("1"), "text/plain")

	items, err := s.List(ctx, ListOptions{Offset: 100})
	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestInMemoryStorage_ListEmpty(t *testing.T) {
	s := NewInMemoryStorage()
	items, err := s.List(context.Background(), ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestInMemoryStorage_Stat(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	_ = s.Put(ctx, "test-key", []byte("hello"), "text/plain")

	info, err := s.Stat(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-key", info.Key)
	assert.Equal(t, int64(5), info.Size)
	assert.Equal(t, "text/plain", info.ContentType)
	assert.False(t, info.CreatedAt.IsZero())
}

func TestInMemoryStorage_StatNotFound(t *testing.T) {
	s := NewInMemoryStorage()
	_, err := s.Stat(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryStorage_PutOverwrite(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()

	_ = s.Put(ctx, "key", []byte("old"), "text/plain")
	_ = s.Put(ctx, "key", []byte("new"), "text/plain")

	data, err := s.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), data)
}

func TestInMemoryStorage_GetReturnsCopy(t *testing.T) {
	s := NewInMemoryStorage()
	ctx := context.Background()
	_ = s.Put(ctx, "key", []byte("original"), "text/plain")

	data, _ := s.Get(ctx, "key")
	data[0] = 'X' // modify returned copy

	original, _ := s.Get(ctx, "key")
	assert.Equal(t, []byte("original"), original)
}
