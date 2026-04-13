package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInMemoryManager_MapCheckpoint(t *testing.T) {
	ctx := context.Background()
	m := NewInMemoryManager()
	r, err := m.MapRegion(ctx, "a1", MemoryRegionSpec{
		Size:        4096,
		Protection:  ProtRead | ProtWrite,
		BackingType: BackingVolume,
		BackingRef:  "vol-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r.RegionID)

	cp, err := m.Checkpoint(ctx, "a1")
	require.NoError(t, err)
	id, err := m.Restore(ctx, cp)
	require.NoError(t, err)
	require.Equal(t, "a1", id)

	require.NoError(t, m.UnmapRegion(ctx, r.RegionID))
}
