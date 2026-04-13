// Package edge manages edge nodes and offline-capable agents.
package edge

import (
	"context"
	"sync"
	"time"
)

// EdgeNode is a constrained worker joining from edge sites.
type EdgeNode struct {
	ID         string
	LastSync   time.Time
	OfflineCap bool
	BandwidthK int // KB/s estimate
}

// Manager tracks edge inventory.
type Manager struct {
	mu    sync.RWMutex
	nodes map[string]*EdgeNode
}

// NewManager creates an empty manager.
func NewManager() *Manager {
	return &Manager{nodes: make(map[string]*EdgeNode)}
}

// Register adds or updates an edge node heartbeat.
func (m *Manager) Register(ctx context.Context, n EdgeNode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n.LastSync = time.Now()
	m.nodes[n.ID] = &n
}

// List returns known edge nodes.
func (m *Manager) List() []EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]EdgeNode, 0, len(m.nodes))
	for _, n := range m.nodes {
		cp := *n
		out = append(out, cp)
	}
	return out
}
