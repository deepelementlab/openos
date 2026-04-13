// Package algorithm provides production-grade scheduling caches (node state, score precomputation).
package algorithm

import (
	"sync"
	"time"

	"github.com/agentos/aos/internal/scheduler"
)

// NodeStateCache holds a read-optimized snapshot of cluster nodes with incremental updates.
// Reduces repeated full-cluster fetches during scheduling bursts.
type NodeStateCache struct {
	mu          sync.RWMutex
	nodes       map[string]scheduler.NodeState
	lastRefresh time.Time
	ttl         time.Duration
}

// NewNodeStateCache creates a cache with optional TTL for List snapshot freshness.
func NewNodeStateCache(ttl time.Duration) *NodeStateCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &NodeStateCache{
		nodes: make(map[string]scheduler.NodeState),
		ttl:   ttl,
	}
}

// Upsert merges or replaces a single node (incremental update path).
func (c *NodeStateCache) Upsert(node scheduler.NodeState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes[node.NodeID] = node
	c.lastRefresh = time.Now()
}

// Remove drops a node from cache (e.g. node decommissioned).
func (c *NodeStateCache) Remove(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodes, nodeID)
	c.lastRefresh = time.Now()
}

// ReplaceAll replaces the entire snapshot (full sync from control plane).
func (c *NodeStateCache) ReplaceAll(nodes []scheduler.NodeState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes = make(map[string]scheduler.NodeState, len(nodes))
	for _, n := range nodes {
		c.nodes[n.NodeID] = n
	}
	c.lastRefresh = time.Now()
}

// Snapshot returns a copy of all cached nodes.
func (c *NodeStateCache) Snapshot() []scheduler.NodeState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]scheduler.NodeState, 0, len(c.nodes))
	for _, n := range c.nodes {
		out = append(out, n)
	}
	return out
}

// Get returns one node if present.
func (c *NodeStateCache) Get(nodeID string) (scheduler.NodeState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n, ok := c.nodes[nodeID]
	return n, ok
}

// Len returns number of cached nodes.
func (c *NodeStateCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes)
}

// Stale reports whether the cache exceeded TTL since last write.
func (c *NodeStateCache) Stale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastRefresh.IsZero() {
		return true
	}
	return time.Since(c.lastRefresh) > c.ttl
}

// LastRefresh returns timestamp of last Upsert/Remove/ReplaceAll.
func (c *NodeStateCache) LastRefresh() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}
