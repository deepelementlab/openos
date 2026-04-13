package vfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInMemoryManager_MountOpenWrite(t *testing.T) {
	ctx := context.Background()
	v := NewInMemoryManager()
	_, err := v.Mount(ctx, MountSpec{Source: "/data", Target: "/mnt", FSType: FSTypeMemory})
	require.NoError(t, err)
	fd, err := v.Open(ctx, "/mnt/x", OpenRead|OpenWrite)
	require.NoError(t, err)
	n, err := v.Write(ctx, fd, []byte("hello"), 0)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	buf := make([]byte, 10)
	rn, err := v.Read(ctx, fd, buf, 0)
	require.NoError(t, err)
	require.Equal(t, 5, rn)
	require.NoError(t, v.Close(ctx, fd))
}
