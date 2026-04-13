package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalRegistry_PushPull(t *testing.T) {
	root := t.TempDir()
	reg := NewLocalRegistry(root)
	pkg := filepath.Join(t.TempDir(), "bundle")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "manifest.json"), []byte(`{"metadata":{"name":"x","version":"1"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "layers.json"), []byte(`{"layer_digests":["abc"]}`), 0o644))

	require.NoError(t, reg.Push(pkg, "x", "1", PushOptions{}))
	path, err := reg.Pull("x", "1", PullOptions{})
	require.NoError(t, err)
	ap, err := LoadAgentPackage(path)
	require.NoError(t, err)
	require.Equal(t, "x", ap.Name)
}
