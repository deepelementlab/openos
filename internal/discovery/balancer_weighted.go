package discovery

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// WeightedBalancer implements weighted round-robin load balancing.
type WeightedBalancer struct {
	BaseBalancer
	rng *rand.Rand
	mu  sync.Mutex
}

// NewWeightedBalancer creates a new weighted load balancer.
func NewWeightedBalancer() *WeightedBalancer {
	b := &WeightedBalancer{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	b.SetHealthyOnly(true)
	return b
}

// Select selects an instance using weighted random selection.
func (b *WeightedBalancer) Select(ctx context.Context, serviceSet *ServiceSet) (*ServiceInstance, error) {
	if len(serviceSet.Instances) == 0 {
		return nil, ErrEmptyServiceSet
	}

	candidates := b.GetCandidates(serviceSet)
	if len(candidates) == 0 {
		return nil, ErrNoHealthyInstances
	}

	// Calculate total weight
	totalWeight := 0
	for _, instance := range candidates {
		weight := instance.Weight
		if weight <= 0 {
			weight = 100 // Default weight
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		// Fall back to random selection
		b.mu.Lock()
		index := b.rng.Intn(len(candidates))
		b.mu.Unlock()
		return candidates[index], nil
	}

	// Weighted random selection
	b.mu.Lock()
	randomWeight := b.rng.Intn(totalWeight)
	b.mu.Unlock()

	currentWeight := 0
	for _, instance := range candidates {
		weight := instance.Weight
		if weight <= 0 {
			weight = 100
		}
		currentWeight += weight
		if randomWeight < currentWeight {
			return instance, nil
		}
	}

	// Should not reach here, but return first as fallback
	return candidates[0], nil
}

// Name returns the load balancer name.
func (b *WeightedBalancer) Name() string {
	return "weighted"
}
