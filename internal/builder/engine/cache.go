package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LayerCache stores content-addressable layer metadata under a root directory (e.g. ~/.cache/aos/cache/layers).
type LayerCache struct {
	root string
}

// DefaultCacheRoot returns the OS user cache subdirectory for AOS.
func DefaultCacheRoot() (string, error) {
	d, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache: user cache dir: %w", err)
	}
	return filepath.Join(d, "aos", "cache"), nil
}

// OpenLayerCache opens or creates the layer cache. If root is empty, uses DefaultCacheRoot.
func OpenLayerCache(root string) (*LayerCache, error) {
	if root == "" {
		var err error
		root, err = DefaultCacheRoot()
		if err != nil {
			return nil, err
		}
	}
	layersRoot := filepath.Join(root, "layers")
	if err := os.MkdirAll(layersRoot, 0o755); err != nil {
		return nil, fmt.Errorf("cache: mkdir: %w", err)
	}
	return &LayerCache{root: layersRoot}, nil
}

func (c *LayerCache) shardPath(digest string) string {
	if len(digest) < 4 {
		return filepath.Join(c.root, digest)
	}
	return filepath.Join(c.root, digest[:2], digest[2:])
}

// Has reports whether a digest exists in the cache.
func (c *LayerCache) Has(digest string) bool {
	if c == nil || digest == "" {
		return false
	}
	st, err := os.Stat(c.shardPath(digest))
	return err == nil && !st.IsDir()
}

// Get reads cached JSON metadata for a digest.
func (c *LayerCache) Get(digest string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("cache: nil")
	}
	return os.ReadFile(c.shardPath(digest))
}

// LayerMeta is stored per digest for inspection and checkpoint linking.
type LayerMeta struct {
	Digest       string    `json:"digest"`
	CreatedAt    time.Time `json:"created_at"`
	PackageHint  string    `json:"package_hint,omitempty"`
	CheckpointID string    `json:"checkpoint_id,omitempty"`
}

// Put writes layer metadata bytes (JSON) for a digest.
func (c *LayerCache) Put(digest string, meta LayerMeta) error {
	if c == nil {
		return fmt.Errorf("cache: nil")
	}
	meta.Digest = digest
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	path := c.shardPath(digest)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// PruneOlderThan removes layer files with mtime before cutoff (best-effort).
func (c *LayerCache) PruneOlderThan(cutoff time.Time) (removed int, err error) {
	if c == nil {
		return 0, fmt.Errorf("cache: nil")
	}
	err = filepath.Walk(c.root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if rmErr := os.Remove(path); rmErr == nil {
				removed++
			}
		}
		return nil
	})
	return removed, err
}
