package federation

import (
	"context"
	"time"
)

// HeartbeatLoop periodically marks cluster liveness; in production this receives gRPC heartbeats.
func (r *Registry) HeartbeatLoop(ctx context.Context, clusterID string, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.mu.Lock()
			if c, ok := r.clusters[clusterID]; ok {
				c.LastSeen = time.Now()
				c.Healthy = true
			}
			r.mu.Unlock()
		}
	}
}

// MarkStale flags clusters that missed heartbeats beyond maxAge.
func (r *Registry) MarkStale(maxAge time.Duration) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []string
	now := time.Now()
	for id, c := range r.clusters {
		if now.Sub(c.LastSeen) > maxAge {
			c.Healthy = false
			ids = append(ids, id)
		}
	}
	return ids
}
