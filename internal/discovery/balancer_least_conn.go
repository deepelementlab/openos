package discovery

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// LeastConnectionBalancer implements least-connections load balancing.
type LeastConnectionBalancer struct {
	BaseBalancer
	connections map[string]int64
	mu          sync.RWMutex
}

// NewLeastConnectionBalancer creates a new least-connection load balancer.
func NewLeastConnectionBalancer() *LeastConnectionBalancer {
	b := &LeastConnectionBalancer{
		connections: make(map[string]int64),
	}
	b.SetHealthyOnly(true)
	return b
}

// Select selects the instance with the fewest connections.
func (b *LeastConnectionBalancer) Select(ctx context.Context, serviceSet *ServiceSet) (*ServiceInstance, error) {
	if len(serviceSet.Instances) == 0 {
		return nil, ErrEmptyServiceSet
	}

	candidates := b.GetCandidates(serviceSet)
	if len(candidates) == 0 {
		return nil, ErrNoHealthyInstances
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Find instance with minimum connections
	var selected *ServiceInstance
	minConnections := int64(math.MaxInt64)

	for _, instance := range candidates {
		connCount := b.connections[instance.ID]
		if connCount < minConnections {
			minConnections = connCount
			selected = instance
		}
	}

	if selected == nil {
		selected = candidates[0]
	}

	return selected, nil
}

// Name returns the load balancer name.
func (b *LeastConnectionBalancer) Name() string {
	return "least_connection"
}

// IncrementConnections increments the connection count for an instance.
func (b *LeastConnectionBalancer) IncrementConnections(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connections[instanceID]++
}

// DecrementConnections decrements the connection count for an instance.
func (b *LeastConnectionBalancer) DecrementConnections(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connections[instanceID] > 0 {
		b.connections[instanceID]--
	}
}

// GetConnectionCount returns the connection count for an instance.
func (b *LeastConnectionBalancer) GetConnectionCount(instanceID string) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.connections[instanceID]
}

// RandomBalancer implements random load balancing.
type RandomBalancer struct {
	BaseBalancer
	rng *rand.Rand
	mu  sync.Mutex
}

// NewRandomBalancer creates a new random load balancer.
func NewRandomBalancer() *RandomBalancer {
	b := &RandomBalancer{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	b.SetHealthyOnly(true)
	return b
}

// Select selects a random instance.
func (b *RandomBalancer) Select(ctx context.Context, serviceSet *ServiceSet) (*ServiceInstance, error) {
	if len(serviceSet.Instances) == 0 {
		return nil, ErrEmptyServiceSet
	}

	candidates := b.GetCandidates(serviceSet)
	if len(candidates) == 0 {
		return nil, ErrNoHealthyInstances
	}

	b.mu.Lock()
	index := b.rng.Intn(len(candidates))
	b.mu.Unlock()

	return candidates[index], nil
}

// Name returns the load balancer name.
func (b *RandomBalancer) Name() string {
	return "random"
}
