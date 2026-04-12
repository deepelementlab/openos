package saga

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Saga represents a distributed transaction definition.
type Saga struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Steps       []*SagaStep `json:"steps"`
	ExecutionMode ExecutionMode `json:"execution_mode"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

// ExecutionMode defines how saga steps are executed.
type ExecutionMode string

const (
	// Sequential executes steps one by one.
	Sequential ExecutionMode = "sequential"
	// Parallel executes independent steps in parallel.
	Parallel ExecutionMode = "parallel"
)

// SagaStep represents a single step in a saga.
type SagaStep struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Action          SagaAction             `json:"-"`
	Compensation    SagaAction             `json:"-"`
	CompensationID  string                 `json:"compensation_id,omitempty"`
	Dependencies    []string               `json:"dependencies,omitempty"`
	Timeout         time.Duration          `json:"timeout,omitempty"`
	RetryPolicy     *RetryPolicy           `json:"retry_policy,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SagaAction is the action to execute for a saga step.
type SagaAction interface {
	// Execute executes the action.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	// GetName returns the action name.
	GetName() string
}

// SagaActionFunc is a function that implements SagaAction.
type SagaActionFunc func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

// Execute executes the action.
func (f SagaActionFunc) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return f(ctx, input)
}

// GetName returns the action name.
func (f SagaActionFunc) GetName() string {
	return "SagaActionFunc"
}

// RetryPolicy defines retry behavior for saga steps.
type RetryPolicy struct {
	MaxAttempts  int           `json:"max_attempts"`
	BackoffType  BackoffType   `json:"backoff_type"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
}

// BackoffType defines the backoff strategy.
type BackoffType string

const (
	BackoffFixed       BackoffType = "fixed"
	BackoffLinear      BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
)

// SagaInstance represents a running saga instance.
type SagaInstance struct {
	ID           string                 `json:"id"`
	SagaID       string                 `json:"saga_id"`
	Status       SagaStatus             `json:"status"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	StepResults  map[string]*StepResult `json:"step_results,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CompensationLog []*CompensationRecord `json:"compensation_log,omitempty"`
}

// SagaStatus represents the status of a saga instance.
type SagaStatus string

const (
	SagaStatusPending      SagaStatus = "pending"
	SagaStatusRunning      SagaStatus = "running"
	SagaStatusCompleted    SagaStatus = "completed"
	SagaStatusFailed       SagaStatus = "failed"
	SagaStatusCompensating SagaStatus = "compensating"
	SagaStatusCompensated  SagaStatus = "compensated"
	SagaStatusPartiallyCompensated SagaStatus = "partially_compensated"
)

// IsTerminal checks if the saga has reached a terminal status.
func (s SagaStatus) IsTerminal() bool {
	return s == SagaStatusCompleted || s == SagaStatusFailed ||
		s == SagaStatusCompensated || s == SagaStatusPartiallyCompensated
}

// StepResult represents the result of a saga step execution.
type StepResult struct {
	StepID      string                 `json:"step_id"`
	Success     bool                   `json:"success"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Compensated bool                   `json:"compensated,omitempty"`
}

// CompensationRecord records a compensation attempt.
type CompensationRecord struct {
	StepID      string    `json:"step_id"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	ExecutedAt  time.Time `json:"executed_at"`
}

// NewSagaInstance creates a new saga instance.
func NewSagaInstance(sagaID string, input map[string]interface{}) *SagaInstance {
	return &SagaInstance{
		ID:           uuid.New().String(),
		SagaID:       sagaID,
		Status:       SagaStatusPending,
		Input:        input,
		Output:       make(map[string]interface{}),
		StepResults:  make(map[string]*StepResult),
		StartedAt:    time.Now().UTC(),
		CompensationLog: make([]*CompensationRecord, 0),
	}
}

// Duration returns the duration of the saga execution.
func (si *SagaInstance) Duration() time.Duration {
	if si.CompletedAt != nil {
		return si.CompletedAt.Sub(si.StartedAt)
	}
	return time.Since(si.StartedAt)
}

// AddStepResult adds a step result.
func (si *SagaInstance) AddStepResult(result *StepResult) {
	si.StepResults[result.StepID] = result
}

// GetStepResult retrieves a step result.
func (si *SagaInstance) GetStepResult(stepID string) (*StepResult, bool) {
	result, exists := si.StepResults[stepID]
	return result, exists
}

// AddCompensationRecord adds a compensation record.
func (si *SagaInstance) AddCompensationRecord(record *CompensationRecord) {
	si.CompensationLog = append(si.CompensationLog, record)
}

// MarkCompleted marks the saga as completed.
func (si *SagaInstance) MarkCompleted() {
	si.Status = SagaStatusCompleted
	now := time.Now().UTC()
	si.CompletedAt = &now
}

// MarkFailed marks the saga as failed.
func (si *SagaInstance) MarkFailed(err error) {
	si.Status = SagaStatusFailed
	if err != nil {
		si.Error = err.Error()
	}
	now := time.Now().UTC()
	si.CompletedAt = &now
}

// MarkCompensating marks the saga as compensating.
func (si *SagaInstance) MarkCompensating() {
	si.Status = SagaStatusCompensating
}

// MarkCompensated marks the saga as fully compensated.
func (si *SagaInstance) MarkCompensated() {
	si.Status = SagaStatusCompensated
	now := time.Now().UTC()
	si.CompletedAt = &now
}

// MarkPartiallyCompensated marks the saga as partially compensated.
func (si *SagaInstance) MarkPartiallyCompensated() {
	si.Status = SagaStatusPartiallyCompensated
	now := time.Now().UTC()
	si.CompletedAt = &now
}

// GetCompletedSteps returns IDs of successfully completed steps.
func (si *SagaInstance) GetCompletedSteps() []string {
	var completed []string
	for stepID, result := range si.StepResults {
		if result.Success {
			completed = append(completed, stepID)
		}
	}
	return completed
}

// GetFailedStep returns the first failed step.
func (si *SagaInstance) GetFailedStep() *StepResult {
	for _, result := range si.StepResults {
		if !result.Success {
			return result
		}
	}
	return nil
}

// SagaBuilder provides a fluent API for building sagas.
type SagaBuilder struct {
	saga *Saga
}

// NewSagaBuilder creates a new saga builder.
func NewSagaBuilder(id, name string) *SagaBuilder {
	return &SagaBuilder{
		saga: &Saga{
			ID:            id,
			Name:          name,
			Version:       "1.0",
			Steps:         make([]*SagaStep, 0),
			ExecutionMode: Sequential,
			Timeout:       5 * time.Minute,
		},
	}
}

// Description sets the saga description.
func (b *SagaBuilder) Description(desc string) *SagaBuilder {
	b.saga.Description = desc
	return b
}

// Version sets the saga version.
func (b *SagaBuilder) Version(version string) *SagaBuilder {
	b.saga.Version = version
	return b
}

// Parallel sets execution mode to parallel.
func (b *SagaBuilder) Parallel() *SagaBuilder {
	b.saga.ExecutionMode = Parallel
	return b
}

// Timeout sets the saga timeout.
func (b *SagaBuilder) Timeout(timeout time.Duration) *SagaBuilder {
	b.saga.Timeout = timeout
	return b
}

// Step adds a step to the saga.
func (b *SagaBuilder) Step(id, name string, action, compensation SagaAction) *SagaBuilder {
	step := &SagaStep{
		ID:           id,
		Name:         name,
		Action:       action,
		Compensation: compensation,
	}
	b.saga.Steps = append(b.saga.Steps, step)
	return b
}

// StepWithDeps adds a step with dependencies.
func (b *SagaBuilder) StepWithDeps(id, name string, action, compensation SagaAction, deps []string) *SagaBuilder {
	step := &SagaStep{
		ID:           id,
		Name:         name,
		Action:       action,
		Compensation: compensation,
		Dependencies: deps,
	}
	b.saga.Steps = append(b.saga.Steps, step)
	return b
}

// Build builds the saga.
func (b *SagaBuilder) Build() (*Saga, error) {
	if len(b.saga.Steps) == 0 {
		return nil, fmt.Errorf("saga must have at least one step")
	}

	// Validate step IDs are unique
	stepIDs := make(map[string]bool)
	for _, step := range b.saga.Steps {
		if stepIDs[step.ID] {
			return nil, fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Validate dependencies
	for _, step := range b.saga.Steps {
		for _, dep := range step.Dependencies {
			if !stepIDs[dep] {
				return nil, fmt.Errorf("dependency %s not found for step %s", dep, step.ID)
			}
		}
	}

	return b.saga, nil
}

// MustBuild builds the saga or panics.
func (b *SagaBuilder) MustBuild() *Saga {
	saga, err := b.Build()
	if err != nil {
		panic(err)
	}
	return saga
}
