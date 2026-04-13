package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLayerCache_PutGetHas_ShortDigest(t *testing.T) {
	root := t.TempDir()
	c, err := OpenLayerCache(root)
	require.NoError(t, err)
	require.False(t, c.Has(""))
	require.False(t, c.Has("ab"))
	require.NoError(t, c.Put("ab", LayerMeta{PackageHint: "x"}))
	require.True(t, c.Has("ab"))
	b, err := c.Get("ab")
	require.NoError(t, err)
	require.Contains(t, string(b), "ab")
}

func TestLayerCache_ShardedDigest(t *testing.T) {
	root := t.TempDir()
	c, err := OpenLayerCache(root)
	require.NoError(t, err)
	d := "deadbeefcafe"
	require.NoError(t, c.Put(d, LayerMeta{PackageHint: "p"}))
	require.True(t, c.Has(d))
	_, err = os.Stat(filepath.Join(c.root, "de", "adbeefcafe"))
	require.NoError(t, err)
}

func TestLayerCache_PruneOlderThan(t *testing.T) {
	root := t.TempDir()
	c, err := OpenLayerCache(root)
	require.NoError(t, err)
	oldFile := filepath.Join(c.root, "oldlayer")
	require.NoError(t, os.WriteFile(oldFile, []byte("{}"), 0o644))
	past := time.Now().UTC().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(oldFile, past, past))
	n, err := c.PruneOlderThan(time.Now().UTC().Add(-24 * time.Hour))
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
}

func TestLayerCache_NilErrors(t *testing.T) {
	var c *LayerCache
	_, err := c.Get("x")
	require.Error(t, err)
	require.Error(t, c.Put("x", LayerMeta{}))
	_, err = c.PruneOlderThan(time.Time{})
	require.Error(t, err)
}
