package federation

import (
	"fmt"
	"sync"
)

// GlobalQuota tracks federated limits across clusters.
type GlobalQuota struct {
	mu          sync.Mutex
	MaxAgents   int
	PerCluster  map[string]int
	PerTenant   map[string]int
}

// NewGlobalQuota creates quota state with limits.
func NewGlobalQuota(maxAgents int) *GlobalQuota {
	return &GlobalQuota{
		MaxAgents:  maxAgents,
		PerCluster: make(map[string]int),
		PerTenant:  make(map[string]int),
	}
}

// Reserve attempts to allocate agent slots for tenant on cluster.
func (g *GlobalQuota) Reserve(clusterID, tenantID string, delta int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	total := 0
	for _, v := range g.PerCluster {
		total += v
	}
	if total+delta > g.MaxAgents {
		return fmt.Errorf("federation: global agent cap exceeded")
	}
	g.PerCluster[clusterID] += delta
	g.PerTenant[tenantID] += delta
	return nil
}

// Release returns quota when agent terminates or migrates away.
func (g *GlobalQuota) Release(clusterID, tenantID string, delta int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.PerCluster[clusterID] -= delta
	if g.PerCluster[clusterID] < 0 {
		g.PerCluster[clusterID] = 0
	}
	g.PerTenant[tenantID] -= delta
	if g.PerTenant[tenantID] < 0 {
		g.PerTenant[tenantID] = 0
	}
}
