// Package federation implements multi-cluster registration and capability discovery.
package federation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ClusterCapabilities describes what a remote cluster exposes.
type ClusterCapabilities struct {
	GPU        bool              `json:"gpu"`
	RDMA       bool              `json:"rdma"`
	Edge       bool              `json:"edge"`
	Labels     map[string]string `json:"labels"`
	Region     string            `json:"region"`
	Zone       string            `json:"zone"`
	CostTier   string            `json:"cost_tier"`
	MaxAgents  int               `json:"max_agents"`
}

// ClusterRecord is a registered peer cluster.
type ClusterRecord struct {
	ID            string
	Name          string
	Endpoint      string
	BootstrapHash string
	Capabilities  ClusterCapabilities
	RegisteredAt  time.Time
	LastSeen      time.Time
	Healthy       bool
}

// Registry tracks federated clusters in-memory (backed by persistent store in production).
type Registry struct {
	mu       sync.RWMutex
	clusters map[string]*ClusterRecord
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{clusters: make(map[string]*ClusterRecord)}
}

// Register upserts a cluster record (bootstrap token verified upstream).
func (r *Registry) Register(ctx context.Context, rec *ClusterRecord) error {
	if rec == nil || rec.ID == "" {
		return fmt.Errorf("federation: invalid cluster record")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if rec.RegisteredAt.IsZero() {
		rec.RegisteredAt = now
	}
	rec.LastSeen = now
	rec.Healthy = true
	r.clusters[rec.ID] = rec
	return nil
}

// List returns all known clusters.
func (r *Registry) List() []*ClusterRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ClusterRecord, 0, len(r.clusters))
	for _, c := range r.clusters {
		cp := *c
		out = append(out, &cp)
	}
	return out
}

// Get returns one cluster by ID.
func (r *Registry) Get(id string) (*ClusterRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clusters[id]
	if !ok {
		return nil, false
	}
	cp := *c
	return &cp, true
}
