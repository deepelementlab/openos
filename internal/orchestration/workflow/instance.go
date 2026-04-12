package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkflowInstance represents a running instance of a workflow.
type WorkflowInstance struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflow_id"`
	EntityID     string                 `json:"entity_id"`     // e.g., Agent ID
	EntityType   string                 `json:"entity_type"`
	Status       WorkflowStatus         `json:"status"`
	CurrentStep  string                 `json:"current_step"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	StepHistory  []*StepExecution     `json:"step_history,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	ParentID     string                 `json:"parent_id,omitempty"` // For subflows
}

// WorkflowStatus represents the status of a workflow instance.
type WorkflowStatus string

const (
	WorkflowStatusPending    WorkflowStatus = "pending"
	WorkflowStatusRunning    WorkflowStatus = "running"
	WorkflowStatusCompleted  WorkflowStatus = "completed"
	WorkflowStatusFailed     WorkflowStatus = "failed"
	WorkflowStatusCancelled  WorkflowStatus = "cancelled"
	WorkflowStatusCompensating WorkflowStatus = "compensating"
	WorkflowStatusCompensated  WorkflowStatus = "compensated"
)

// IsTerminal checks if the workflow has reached a terminal status.
func (s WorkflowStatus) IsTerminal() bool {
	return s == WorkflowStatusCompleted || s == WorkflowStatusFailed ||
		s == WorkflowStatusCancelled || s == WorkflowStatusCompensated
}

// IsSuccessful checks if the workflow completed successfully.
func (s WorkflowStatus) IsSuccessful() bool {
	return s == WorkflowStatusCompleted
}

// NewWorkflowInstance creates a new workflow instance.
func NewWorkflowInstance(workflowID, entityID, entityType string, input map[string]interface{}) *WorkflowInstance {
	return &WorkflowInstance{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		EntityID:    entityID,
		EntityType:  entityType,
		Status:      WorkflowStatusPending,
		Input:       input,
		Variables:   make(map[string]interface{}),
		StepHistory: make([]*StepExecution, 0),
		StartedAt:   time.Now().UTC(),
	}
}

// Duration returns the duration of the workflow execution.
func (wi *WorkflowInstance) Duration() time.Duration {
	if wi.CompletedAt != nil {
		return wi.CompletedAt.Sub(wi.StartedAt)
	}
	return time.Since(wi.StartedAt)
}

// GetVariable retrieves a variable value.
func (wi *WorkflowInstance) GetVariable(name string) (interface{}, bool) {
	val, exists := wi.Variables[name]
	return val, exists
}

// SetVariable sets a variable value.
func (wi *WorkflowInstance) SetVariable(name string, value interface{}) {
	if wi.Variables == nil {
		wi.Variables = make(map[string]interface{})
	}
	wi.Variables[name] = value
}

// AddStepExecution adds a step execution to the history.
func (wi *WorkflowInstance) AddStepExecution(exec *StepExecution) {
	wi.StepHistory = append(wi.StepHistory, exec)
}

// GetLastStepExecution returns the most recent step execution.
func (wi *WorkflowInstance) GetLastStepExecution() *StepExecution {
	if len(wi.StepHistory) == 0 {
		return nil
	}
	return wi.StepHistory[len(wi.StepHistory)-1]
}

// MarkCompleted marks the workflow as completed.
func (wi *WorkflowInstance) MarkCompleted(output map[string]interface{}) {
	wi.Status = WorkflowStatusCompleted
	wi.Output = output
	now := time.Now().UTC()
	wi.CompletedAt = &now
}

// MarkFailed marks the workflow as failed.
func (wi *WorkflowInstance) MarkFailed(err error) {
	wi.Status = WorkflowStatusFailed
	if err != nil {
		wi.Error = err.Error()
	}
	now := time.Now().UTC()
	wi.CompletedAt = &now
}

// MarkCancelled marks the workflow as cancelled.
func (wi *WorkflowInstance) MarkCancelled() {
	wi.Status = WorkflowStatusCancelled
	now := time.Now().UTC()
	wi.CompletedAt = &now
}

// MarkCompensating marks the workflow as in compensation.
func (wi *WorkflowInstance) MarkCompensating() {
	wi.Status = WorkflowStatusCompensating
}

// MarkCompensated marks the workflow as fully compensated.
func (wi *WorkflowInstance) MarkCompensated() {
	wi.Status = WorkflowStatusCompensated
	now := time.Now().UTC()
	wi.CompletedAt = &now
}

// GetStepCount returns the number of executed steps.
func (wi *WorkflowInstance) GetStepCount() int {
	return len(wi.StepHistory)
}

// GetFailedSteps returns all failed step executions.
func (wi *WorkflowInstance) GetFailedSteps() []*StepExecution {
	var failed []*StepExecution
	for _, exec := range wi.StepHistory {
		if exec.Status == StepStatusFailed {
			failed = append(failed, exec)
		}
	}
	return failed
}

// WorkflowInstanceStore manages workflow instances.
type WorkflowInstanceStore interface {
	// Create creates a new workflow instance.
	Create(ctx context.Context, instance *WorkflowInstance) error

	// Get retrieves a workflow instance by ID.
	Get(ctx context.Context, instanceID string) (*WorkflowInstance, error)

	// GetByEntity retrieves a workflow instance by entity ID.
	GetByEntity(ctx context.Context, entityID string, status WorkflowStatus) (*WorkflowInstance, error)

	// Update updates a workflow instance.
	Update(ctx context.Context, instance *WorkflowInstance) error

	// List lists workflow instances.
	List(ctx context.Context, filter WorkflowInstanceFilter) ([]*WorkflowInstance, error)

	// Delete deletes a workflow instance.
	Delete(ctx context.Context, instanceID string) error
}

// WorkflowInstanceFilter provides filtering for listing instances.
type WorkflowInstanceFilter struct {
	WorkflowID  string
	EntityID    string
	EntityType  string
	Status      WorkflowStatus
	Limit       int
	Offset      int
}

// InMemoryWorkflowInstanceStore implements WorkflowInstanceStore in memory.
type InMemoryWorkflowInstanceStore struct {
	instances map[string]*WorkflowInstance // ID -> instance
	byEntity  map[string]string            // entityID -> instanceID (latest)
}

// NewInMemoryWorkflowInstanceStore creates an in-memory store.
func NewInMemoryWorkflowInstanceStore() WorkflowInstanceStore {
	return &InMemoryWorkflowInstanceStore{
		instances: make(map[string]*WorkflowInstance),
		byEntity:  make(map[string]string),
	}
}

// Create implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) Create(ctx context.Context, instance *WorkflowInstance) error {
	s.instances[instance.ID] = instance
	s.byEntity[instance.EntityID] = instance.ID
	return nil
}

// Get implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) Get(ctx context.Context, instanceID string) (*WorkflowInstance, error) {
	instance, exists := s.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	return instance, nil
}

// GetByEntity implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) GetByEntity(ctx context.Context, entityID string, status WorkflowStatus) (*WorkflowInstance, error) {
	instanceID, exists := s.byEntity[entityID]
	if !exists {
		return nil, fmt.Errorf("no workflow found for entity %s", entityID)
	}
	return s.Get(ctx, instanceID)
}

// Update implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) Update(ctx context.Context, instance *WorkflowInstance) error {
	s.instances[instance.ID] = instance
	return nil
}

// List implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) List(ctx context.Context, filter WorkflowInstanceFilter) ([]*WorkflowInstance, error) {
	var result []*WorkflowInstance

	for _, instance := range s.instances {
		if filter.WorkflowID != "" && instance.WorkflowID != filter.WorkflowID {
			continue
		}
		if filter.EntityID != "" && instance.EntityID != filter.EntityID {
			continue
		}
		if filter.EntityType != "" && instance.EntityType != filter.EntityType {
			continue
		}
		if filter.Status != "" && instance.Status != filter.Status {
			continue
		}

		result = append(result, instance)
	}

	// Apply pagination
	if filter.Offset >= len(result) {
		return []*WorkflowInstance{}, nil
	}

	end := filter.Offset + filter.Limit
	if filter.Limit == 0 || end > len(result) {
		end = len(result)
	}

	return result[filter.Offset:end], nil
}

// Delete implements WorkflowInstanceStore.
func (s *InMemoryWorkflowInstanceStore) Delete(ctx context.Context, instanceID string) error {
	instance, exists := s.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	delete(s.instances, instanceID)
	delete(s.byEntity, instance.EntityID)
	return nil
}
