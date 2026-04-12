package saga

import (
	"context"
	"fmt"
	"sync"

	"github.com/agentos/aos/internal/messaging"
	"go.uber.org/zap"
)

// SagaCoordinator coordinates the execution of sagas.
type SagaCoordinator struct {
	sagas       map[string]*Saga
	instances   map[string]*SagaInstance
	store       SagaStore
	eventBus    messaging.EventBus
	logger      *zap.Logger
	mu          sync.RWMutex
}

// CoordinatorOptions provides options for creating a coordinator.
type CoordinatorOptions struct {
	Store    SagaStore
	EventBus messaging.EventBus
	Logger   *zap.Logger
}

// NewSagaCoordinator creates a new saga coordinator.
func NewSagaCoordinator(opts CoordinatorOptions) *SagaCoordinator {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SagaCoordinator{
		sagas:     make(map[string]*Saga),
		instances: make(map[string]*SagaInstance),
		store:     opts.Store,
		eventBus:  opts.EventBus,
		logger:    logger,
	}
}

// RegisterSaga registers a saga definition.
func (c *SagaCoordinator) RegisterSaga(saga *Saga) error {
	if saga.ID == "" {
		return fmt.Errorf("saga ID is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sagas[saga.ID] = saga
	c.logger.Info("Registered saga", zap.String("saga_id", saga.ID), zap.String("name", saga.Name))
	return nil
}

// StartSaga starts a new saga instance.
func (c *SagaCoordinator) StartSaga(ctx context.Context, sagaID string, input map[string]interface{}) (*SagaInstance, error) {
	c.mu.RLock()
	saga, exists := c.sagas[sagaID]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("saga %s not found", sagaID)
	}

	instance := NewSagaInstance(sagaID, input)
	instance.Status = SagaStatusRunning

	// Persist instance
	if c.store != nil {
		if err := c.store.Save(ctx, instance); err != nil {
			return nil, fmt.Errorf("failed to save saga instance: %w", err)
		}
	}

	c.mu.Lock()
	c.instances[instance.ID] = instance
	c.mu.Unlock()

	// Publish event
	if c.eventBus != nil {
		event, _ := messaging.NewEvent(messaging.EventSagaStarted, map[string]interface{}{
			"saga_id":     sagaID,
			"instance_id": instance.ID,
		})
		_ = c.eventBus.Publish(ctx, event)
	}

	c.logger.Info("Saga started",
		zap.String("saga_id", sagaID),
		zap.String("instance_id", instance.ID),
	)

	// Execute saga in background
	go c.executeSaga(context.Background(), instance, saga)

	return instance, nil
}

// executeSaga executes a saga instance.
func (c *SagaCoordinator) executeSaga(ctx context.Context, instance *SagaInstance, saga *Saga) {
	executor := NewSagaExecutor()

	// Apply timeout if specified
	if saga.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, saga.Timeout)
		defer cancel()
	}

	var err error
	switch saga.ExecutionMode {
	case Sequential:
		err = executor.ExecuteSequential(ctx, instance, saga)
	case Parallel:
		err = executor.ExecuteParallel(ctx, instance, saga)
	default:
		err = executor.ExecuteSequential(ctx, instance, saga)
	}

	if err != nil {
		c.logger.Error("Saga execution failed",
			zap.Error(err),
			zap.String("instance_id", instance.ID),
		)

		// Attempt compensation
		if instance.Status != SagaStatusCompensating && instance.Status != SagaStatusCompensated {
			c.compensateSaga(ctx, instance, saga)
		}
	}

	// Finalize
	c.finalizeSaga(ctx, instance, saga)
}

// compensateSaga compensates a failed saga.
func (c *SagaCoordinator) compensateSaga(ctx context.Context, instance *SagaInstance, saga *Saga) {
	c.logger.Info("Compensating saga",
		zap.String("instance_id", instance.ID),
		zap.String("saga_id", saga.ID),
	)

	instance.MarkCompensating()

	executor := NewSagaExecutor()
	result := executor.Compensate(ctx, instance, saga)

	if result.AllCompensated {
		instance.MarkCompensated()
	} else {
		instance.MarkPartiallyCompensated()
	}

	if c.store != nil {
		_ = c.store.Save(ctx, instance)
	}

	// Publish compensation event
	if c.eventBus != nil {
		event, _ := messaging.NewEvent(messaging.EventSagaCompensated, map[string]interface{}{
			"saga_id":           saga.ID,
			"instance_id":       instance.ID,
			"all_compensated":   result.AllCompensated,
			"failed_compensations": result.FailedCompensations,
		})
		_ = c.eventBus.Publish(ctx, event)
	}
}

// finalizeSaga finalizes a saga instance.
func (c *SagaCoordinator) finalizeSaga(ctx context.Context, instance *SagaInstance, saga *Saga) {
	// Save final state
	if c.store != nil {
		_ = c.store.Save(ctx, instance)
	}

	// Remove from active instances
	c.mu.Lock()
	delete(c.instances, instance.ID)
	c.mu.Unlock()

	// Publish completion event
	var eventType string
	if instance.Status == SagaStatusCompleted {
		eventType = messaging.EventSagaCompleted
	} else if instance.Status == SagaStatusCompensated {
		eventType = messaging.EventSagaCompensated
	} else {
		eventType = messaging.EventSagaFailed
	}

	if c.eventBus != nil {
		event, _ := messaging.NewEvent(eventType, map[string]interface{}{
			"saga_id":      saga.ID,
			"instance_id":  instance.ID,
			"status":       instance.Status,
			"duration_ms":  instance.Duration().Milliseconds(),
			"error":        instance.Error,
		})
		_ = c.eventBus.Publish(ctx, event)
	}

	c.logger.Info("Saga finalized",
		zap.String("instance_id", instance.ID),
		zap.String("status", string(instance.Status)),
		zap.Duration("duration", instance.Duration()),
	)
}

// GetInstance retrieves a saga instance.
func (c *SagaCoordinator) GetInstance(ctx context.Context, instanceID string) (*SagaInstance, error) {
	c.mu.RLock()
	if instance, exists := c.instances[instanceID]; exists {
		c.mu.RUnlock()
		return instance, nil
	}
	c.mu.RUnlock()

	if c.store != nil {
		return c.store.Get(ctx, instanceID)
	}

	return nil, fmt.Errorf("saga instance %s not found", instanceID)
}

// GetSaga retrieves a saga definition.
func (c *SagaCoordinator) GetSaga(sagaID string) (*Saga, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	saga, exists := c.sagas[sagaID]
	if !exists {
		return nil, fmt.Errorf("saga %s not found", sagaID)
	}

	return saga, nil
}

// ListActiveInstances returns all active saga instances.
func (c *SagaCoordinator) ListActiveInstances() []*SagaInstance {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var active []*SagaInstance
	for _, instance := range c.instances {
		if !instance.Status.IsTerminal() {
			active = append(active, instance)
		}
	}

	return active
}

// Stats returns coordinator statistics.
func (c *SagaCoordinator) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := 0
	for _, instance := range c.instances {
		if !instance.Status.IsTerminal() {
			active++
		}
	}

	return map[string]interface{}{
		"registered_sagas": len(c.sagas),
		"active_instances": active,
		"total_instances":  len(c.instances),
	}
}

// SagaStore defines the interface for saga persistence.
type SagaStore interface {
	// Save saves a saga instance.
	Save(ctx context.Context, instance *SagaInstance) error

	// Get retrieves a saga instance by ID.
	Get(ctx context.Context, instanceID string) (*SagaInstance, error)

	// List lists saga instances.
	List(ctx context.Context, sagaID string, status SagaStatus) ([]*SagaInstance, error)

	// Delete deletes a saga instance.
	Delete(ctx context.Context, instanceID string) error
}

// InMemorySagaStore implements SagaStore in memory.
type InMemorySagaStore struct {
	instances map[string]*SagaInstance
	mu        sync.RWMutex
}

// NewInMemorySagaStore creates an in-memory saga store.
func NewInMemorySagaStore() SagaStore {
	return &InMemorySagaStore{
		instances: make(map[string]*SagaInstance),
	}
}

// Save implements SagaStore.
func (s *InMemorySagaStore) Save(ctx context.Context, instance *SagaInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances[instance.ID] = instance
	return nil
}

// Get implements SagaStore.
func (s *InMemorySagaStore) Get(ctx context.Context, instanceID string) (*SagaInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("saga instance %s not found", instanceID)
	}

	return instance, nil
}

// List implements SagaStore.
func (s *InMemorySagaStore) List(ctx context.Context, sagaID string, status SagaStatus) ([]*SagaInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*SagaInstance
	for _, instance := range s.instances {
		if (sagaID == "" || instance.SagaID == sagaID) &&
			(status == "" || instance.Status == status) {
			result = append(result, instance)
		}
	}

	return result, nil
}

// Delete implements SagaStore.
func (s *InMemorySagaStore) Delete(ctx context.Context, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.instances, instanceID)
	return nil
}
