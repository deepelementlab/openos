package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ServiceRegistry manages service registration and discovery.
type ServiceRegistry interface {
	// Register registers a service instance.
	Register(ctx context.Context, instance *ServiceInstance) error

	// Deregister removes a service instance.
	Deregister(ctx context.Context, instanceID string) error

	// Get retrieves a service instance by ID.
	Get(ctx context.Context, instanceID string) (*ServiceInstance, error)

	// Query queries for service instances.
	Query(ctx context.Context, query ServiceQuery) ([]*ServiceInstance, error)

	// GetService returns all instances of a service.
	GetService(ctx context.Context, serviceName string) (*ServiceSet, error)

	// Heartbeat updates the heartbeat for an instance.
	Heartbeat(ctx context.Context, instanceID string) error

	// UpdateHealth updates the health status of an instance.
	UpdateHealth(ctx context.Context, instanceID string, status HealthStatus) error

	// ListServices returns all registered service names.
	ListServices(ctx context.Context) ([]string, error)

	// Watch watches for changes to a service.
	Watch(ctx context.Context, serviceName string) (<-chan *ServiceSet, error)
}

// InMemoryRegistry implements ServiceRegistry in memory.
type InMemoryRegistry struct {
	instances   map[string]*ServiceInstance // instanceID -> instance
	services    map[string]*ServiceSet      // serviceName -> service set
	watchers    map[string][]chan *ServiceSet
	heartbeatTimeout time.Duration
	logger      *zap.Logger
	mu          sync.RWMutex
	version     int64
}

// NewInMemoryRegistry creates a new in-memory registry.
func NewInMemoryRegistry(logger *zap.Logger) *InMemoryRegistry {
	return &InMemoryRegistry{
		instances:        make(map[string]*ServiceInstance),
		services:         make(map[string]*ServiceSet),
		watchers:         make(map[string][]chan *ServiceSet),
		heartbeatTimeout: 30 * time.Second,
		logger:           logger,
	}
}

// Register implements ServiceRegistry.
func (r *InMemoryRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	if instance.ID == "" {
		return fmt.Errorf("instance ID is required")
	}
	if instance.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Store instance
	r.instances[instance.ID] = instance

	// Update service set
	serviceSet, exists := r.services[instance.ServiceName]
	if !exists {
		serviceSet = &ServiceSet{
			ServiceName: instance.ServiceName,
			Instances:   make([]*ServiceInstance, 0),
		}
		r.services[instance.ServiceName] = serviceSet
	}

	// Check if instance already exists
	found := false
	for i, inst := range serviceSet.Instances {
		if inst.ID == instance.ID {
			serviceSet.Instances[i] = instance
			found = true
			break
		}
	}

	if !found {
		serviceSet.Instances = append(serviceSet.Instances, instance)
	}

	r.version++
	serviceSet.Version = r.version
	serviceSet.UpdatedAt = time.Now().UTC()

	r.logger.Info("Service instance registered",
		zap.String("instance_id", instance.ID),
		zap.String("service", instance.ServiceName),
		zap.String("address", instance.Address()),
	)

	// Notify watchers
	r.notifyWatchers(instance.ServiceName, serviceSet)

	return nil
}

// Deregister implements ServiceRegistry.
func (r *InMemoryRegistry) Deregister(ctx context.Context, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, exists := r.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	delete(r.instances, instanceID)

	// Update service set
	serviceSet := r.services[instance.ServiceName]
	if serviceSet != nil {
		filtered := make([]*ServiceInstance, 0, len(serviceSet.Instances)-1)
		for _, inst := range serviceSet.Instances {
			if inst.ID != instanceID {
				filtered = append(filtered, inst)
			}
		}
		serviceSet.Instances = filtered

		r.version++
		serviceSet.Version = r.version
		serviceSet.UpdatedAt = time.Now().UTC()

		// Clean up empty services
		if len(serviceSet.Instances) == 0 {
			delete(r.services, instance.ServiceName)
		} else {
			// Notify watchers
			r.notifyWatchers(instance.ServiceName, serviceSet)
		}
	}

	r.logger.Info("Service instance deregistered",
		zap.String("instance_id", instanceID),
		zap.String("service", instance.ServiceName),
	)

	return nil
}

// Get implements ServiceRegistry.
func (r *InMemoryRegistry) Get(ctx context.Context, instanceID string) (*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, exists := r.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	// Return a copy
	return r.copyInstance(instance), nil
}

// Query implements ServiceRegistry.
func (r *InMemoryRegistry) Query(ctx context.Context, query ServiceQuery) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ServiceInstance

	for _, instance := range r.instances {
		// Filter by service name
		if query.ServiceName != "" && instance.ServiceName != query.ServiceName {
			continue
		}

		// Filter by health status
		if query.HealthyOnly && !instance.IsHealthy() {
			continue
		}

		// Filter by tags
		if len(query.Tags) > 0 {
			hasTag := false
			for _, queryTag := range query.Tags {
				for _, instanceTag := range instance.Tags {
					if queryTag == instanceTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Filter by metadata
		if len(query.Metadata) > 0 {
			match := true
			for k, v := range query.Metadata {
				if instance.Metadata[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		// Filter by zone
		if query.Zone != "" && instance.Zone != query.Zone {
			continue
		}

		// Filter by region
		if query.Region != "" && instance.Region != query.Region {
			continue
		}

		results = append(results, r.copyInstance(instance))
	}

	return results, nil
}

// GetService implements ServiceRegistry.
func (r *InMemoryRegistry) GetService(ctx context.Context, serviceName string) (*ServiceSet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	serviceSet, exists := r.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Return a copy
	return r.copyServiceSet(serviceSet), nil
}

// Heartbeat implements ServiceRegistry.
func (r *InMemoryRegistry) Heartbeat(ctx context.Context, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, exists := r.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	instance.LastHeartbeat = time.Now().UTC()
	return nil
}

// UpdateHealth implements ServiceRegistry.
func (r *InMemoryRegistry) UpdateHealth(ctx context.Context, instanceID string, status HealthStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, exists := r.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	oldStatus := instance.HealthStatus
	instance.HealthStatus = status

	if oldStatus != status {
		r.logger.Info("Instance health status changed",
			zap.String("instance_id", instanceID),
			zap.String("old_status", string(oldStatus)),
			zap.String("new_status", string(status)),
		)

		// Notify watchers
		serviceSet := r.services[instance.ServiceName]
		if serviceSet != nil {
			r.notifyWatchers(instance.ServiceName, serviceSet)
		}
	}

	return nil
}

// ListServices implements ServiceRegistry.
func (r *InMemoryRegistry) ListServices(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}

	return services, nil
}

// Watch implements ServiceRegistry.
func (r *InMemoryRegistry) Watch(ctx context.Context, serviceName string) (<-chan *ServiceSet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan *ServiceSet, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], ch)

	// Send initial state
	if serviceSet, exists := r.services[serviceName]; exists {
		ch <- r.copyServiceSet(serviceSet)
	}

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()

		watchers := r.watchers[serviceName]
		for i, w := range watchers {
			if w == ch {
				r.watchers[serviceName] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}

// notifyWatchers notifies all watchers of a service change.
func (r *InMemoryRegistry) notifyWatchers(serviceName string, serviceSet *ServiceSet) {
	watchers := r.watchers[serviceName]
	if len(watchers) == 0 {
		return
	}

	copy := r.copyServiceSet(serviceSet)

	for _, ch := range watchers {
		select {
		case ch <- copy:
		default:
			// Channel full, skip this notification
		}
	}
}

// copyInstance creates a copy of a service instance.
func (r *InMemoryRegistry) copyInstance(instance *ServiceInstance) *ServiceInstance {
	copy := &ServiceInstance{
		ID:            instance.ID,
		ServiceName:   instance.ServiceName,
		Host:          instance.Host,
		Port:          instance.Port,
		Metadata:      make(map[string]string),
		Tags:          make([]string, len(instance.Tags)),
		HealthStatus:  instance.HealthStatus,
		Weight:        instance.Weight,
		Zone:          instance.Zone,
		Region:        instance.Region,
		LastHeartbeat: instance.LastHeartbeat,
		RegisteredAt:  instance.RegisteredAt,
		Version:       instance.Version,
	}

	for k, v := range instance.Metadata {
		copy.Metadata[k] = v
	}
	copy.Tags = append(copy.Tags, instance.Tags...)

	return copy
}

// copyServiceSet creates a copy of a service set.
func (r *InMemoryRegistry) copyServiceSet(serviceSet *ServiceSet) *ServiceSet {
	copy := &ServiceSet{
		ServiceName: serviceSet.ServiceName,
		Instances:   make([]*ServiceInstance, len(serviceSet.Instances)),
		Version:     serviceSet.Version,
		UpdatedAt:   serviceSet.UpdatedAt,
	}

	for i, instance := range serviceSet.Instances {
		copy.Instances[i] = r.copyInstance(instance)
	}

	return copy
}

// CleanupExpired removes expired instances.
func (r *InMemoryRegistry) CleanupExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := 0
	now := time.Now().UTC()

	for id, instance := range r.instances {
		if now.Sub(instance.LastHeartbeat) > r.heartbeatTimeout {
			r.logger.Warn("Removing expired instance",
				zap.String("instance_id", id),
				zap.String("service", instance.ServiceName),
			)

			delete(r.instances, id)

			// Update service set
			serviceSet := r.services[instance.ServiceName]
			if serviceSet != nil {
				filtered := make([]*ServiceInstance, 0)
				for _, inst := range serviceSet.Instances {
					if inst.ID != id {
						filtered = append(filtered, inst)
					}
				}
				serviceSet.Instances = filtered

				if len(serviceSet.Instances) == 0 {
					delete(r.services, instance.ServiceName)
				} else {
					r.notifyWatchers(instance.ServiceName, serviceSet)
				}
			}

			removed++
		}
	}

	return removed
}

// StartCleanup starts a background cleanup task.
func (r *InMemoryRegistry) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				removed := r.CleanupExpired()
				if removed > 0 {
					r.logger.Info("Cleaned up expired instances", zap.Int("removed", removed))
				}
			}
		}
	}()
}
