package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentos/aos/internal/kernel/memory"
)

// CheckpointBuildState snapshots builder-related state via the kernel memory manager (stub payload).
func CheckpointBuildState(ctx context.Context, mm memory.Manager, agentID string, plan *BuildPlan) (*memory.Checkpoint, error) {
	if mm == nil {
		return nil, fmt.Errorf("checkpoint: nil memory manager")
	}
	if agentID == "" {
		agentID = "build-" + plan.CacheKeyRoot
	}
	return mm.Checkpoint(ctx, agentID)
}

// RestoreBuildState restores from a kernel checkpoint (stub).
func RestoreBuildState(ctx context.Context, mm memory.Manager, cp *memory.Checkpoint) (string, error) {
	if mm == nil {
		return "", fmt.Errorf("checkpoint: nil memory manager")
	}
	return mm.Restore(ctx, cp)
}

// AttachCheckpointToCache writes checkpoint id into layer cache metadata for the given digest.
func AttachCheckpointToCache(c *LayerCache, digest string, cp *memory.Checkpoint) error {
	if c == nil || cp == nil {
		return nil
	}
	var meta LayerMeta
	if b, err := c.Get(digest); err == nil {
		_ = json.Unmarshal(b, &meta)
	}
	meta.CheckpointID = cp.ID
	return c.Put(digest, meta)
}
