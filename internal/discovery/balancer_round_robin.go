package discovery

import (
	"context"
	"sync"
	"sync/atomic"
)

// RoundRobinBalancer implements round-robin load balancing.
type RoundRobinBalancer struct {
	BaseBalancer
	counter uint64
	mu      sync.RWMutex
}

// Init initializes the balancer.
func (b *RoundRobinBalancer) Init() {
	b.SetHealthyOnly(true)
}

// Select selects an instance using round-robin.
func (b *RoundRobinBalancer) Select(ctx context.Context, serviceSet *ServiceSet) (*ServiceInstance, error) {
	if len(serviceSet.Instances) == 0 {
		return nil, ErrEmptyServiceSet
	}

	candidates := b.GetCandidates(serviceSet)
	if len(candidates) == 0 {
		return nil, ErrNoHealthyInstances
	}

	// Atomically increment counter
	counter := atomic.AddUint64(&b.counter, 1)
	index := (counter - 1) % uint64(len(candidates))

	return candidates[index], nil
}

// Name returns the load balancer name.
func (b *RoundRobinBalancer) Name() string {
	return "round_robin"
}

// Reset resets the counter.
func (b *RoundRobinBalancer) Reset() {
	atomic.StoreUint64(&b.counter, 0)
}
