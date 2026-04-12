package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// LoadBalancer provides load balancing for service instances.
type LoadBalancer interface {
	// Select selects an instance from the service set.
	Select(ctx context.Context, serviceSet *ServiceSet) (*ServiceInstance, error)

	// Name returns the load balancer name.
	Name() string
}

// ErrNoHealthyInstances is returned when no healthy instances are available.
var ErrNoHealthyInstances = errors.New("no healthy instances available")

// ErrEmptyServiceSet is returned when the service set is empty.
var ErrEmptyServiceSet = errors.New("service set is empty")

// BaseBalancer provides common functionality for load balancers.
type BaseBalancer struct {
	healthyOnly bool
}

// SetHealthyOnly sets whether to only consider healthy instances.
func (b *BaseBalancer) SetHealthyOnly(healthyOnly bool) {
	b.healthyOnly = healthyOnly
}

// GetCandidates returns the candidate instances based on health status.
func (b *BaseBalancer) GetCandidates(serviceSet *ServiceSet) []*ServiceInstance {
	if b.healthyOnly {
		candidates := serviceSet.GetHealthy()
		if len(candidates) > 0 {
			return candidates
		}
		// Fall back to all instances if no healthy ones
	}

	return serviceSet.Instances
}

// LoadBalancerRegistry manages load balancer instances.
type LoadBalancerRegistry struct {
	balancers map[string]LoadBalancer
	defaultLB LoadBalancer
	mu        sync.RWMutex
}

// NewLoadBalancerRegistry creates a new load balancer registry.
func NewLoadBalancerRegistry() *LoadBalancerRegistry {
	rr := &RoundRobinBalancer{}
	rr.Init()

	return &LoadBalancerRegistry{
		balancers: map[string]LoadBalancer{
			"round_robin":      rr,
			"random":           &RandomBalancer{},
			"weighted":         &WeightedBalancer{},
			"least_connection": &LeastConnectionBalancer{},
		},
		defaultLB: rr,
	}
}

// Register registers a load balancer.
func (r *LoadBalancerRegistry) Register(name string, lb LoadBalancer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.balancers[name] = lb
}

// Get retrieves a load balancer by name.
func (r *LoadBalancerRegistry) Get(name string) (LoadBalancer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lb, exists := r.balancers[name]
	if !exists {
		return nil, fmt.Errorf("load balancer %s not found", name)
	}

	return lb, nil
}

// GetDefault returns the default load balancer.
func (r *LoadBalancerRegistry) GetDefault() LoadBalancer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultLB
}

// SetDefault sets the default load balancer.
func (r *LoadBalancerRegistry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lb, exists := r.balancers[name]
	if !exists {
		return fmt.Errorf("load balancer %s not found", name)
	}

	r.defaultLB = lb
	return nil
}

// List returns all registered load balancer names.
func (r *LoadBalancerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.balancers))
	for name := range r.balancers {
		names = append(names, name)
	}

	return names
}

// ClientSideLoadBalancer provides client-side load balancing.
type ClientSideLoadBalancer struct {
	registry   ServiceRegistry
	lbRegistry *LoadBalancerRegistry
	cache      *ServiceCache
}

// NewClientSideLoadBalancer creates a new client-side load balancer.
func NewClientSideLoadBalancer(registry ServiceRegistry) *ClientSideLoadBalancer {
	return &ClientSideLoadBalancer{
		registry:   registry,
		lbRegistry: NewLoadBalancerRegistry(),
		cache:      NewServiceCache(30 * time.Second),
	}
}

// Select selects an instance for a service using the specified load balancer.
func (cslb *ClientSideLoadBalancer) Select(ctx context.Context, serviceName string, lbName string) (*ServiceInstance, error) {
	// Get load balancer
	lb, err := cslb.lbRegistry.Get(lbName)
	if err != nil {
		lb = cslb.lbRegistry.GetDefault()
	}

	// Get service set (with caching)
	var serviceSet *ServiceSet
	if cached, ok := cslb.cache.Get(serviceName); ok {
		serviceSet = cached
	} else {
		serviceSet, err = cslb.registry.GetService(ctx, serviceName)
		if err != nil {
			return nil, err
		}
		cslb.cache.Set(serviceName, serviceSet)
	}

	// Select instance
	return lb.Select(ctx, serviceSet)
}

// SelectWithQuery selects an instance matching a query.
func (cslb *ClientSideLoadBalancer) SelectWithQuery(ctx context.Context, query ServiceQuery, lbName string) (*ServiceInstance, error) {
	// Get load balancer
	lb, err := cslb.lbRegistry.Get(lbName)
	if err != nil {
		lb = cslb.lbRegistry.GetDefault()
	}

	// Query instances
	instances, err := cslb.registry.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	// Create a temporary service set
	serviceSet := &ServiceSet{
		ServiceName: query.ServiceName,
		Instances:   instances,
	}

	// Select instance
	return lb.Select(ctx, serviceSet)
}

// WatchAndUpdate watches a service and updates the cache.
func (cslb *ClientSideLoadBalancer) WatchAndUpdate(ctx context.Context, serviceName string) error {
	ch, err := cslb.registry.Watch(ctx, serviceName)
	if err != nil {
		return err
	}

	go func() {
		for serviceSet := range ch {
			cslb.cache.Set(serviceName, serviceSet)
		}
	}()

	return nil
}
