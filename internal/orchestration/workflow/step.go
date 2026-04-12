package workflow

import (
	"context"
	"fmt"
	"time"
)

// StepExecution represents the execution of a workflow step.
type StepExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	InstanceID  string                 `json:"instance_id"`
	StepID      string                 `json:"step_id"`
	Status      StepStatus             `json:"status"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
}

// StepStatus represents the status of a step execution.
type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusRunning    StepStatus = "running"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSkipped    StepStatus = "skipped"
	StepStatusCompensated StepStatus = "compensated"
)

// IsTerminal checks if the step has reached a terminal status.
func (s StepStatus) IsTerminal() bool {
	return s == StepStatusCompleted || s == StepStatusFailed || s == StepStatusSkipped || s == StepStatusCompensated
}

// Duration returns the duration of the step execution.
func (se *StepExecution) Duration() time.Duration {
	if se.CompletedAt != nil {
		return se.CompletedAt.Sub(se.StartedAt)
	}
	if !se.StartedAt.IsZero() {
		return time.Since(se.StartedAt)
	}
	return 0
}

// StepHandler is the interface for workflow step handlers.
type StepHandler interface {
	// Execute executes the step logic.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

	// GetName returns the handler name.
	GetName() string
}

// StepHandlerFunc is a function that implements StepHandler.
type StepHandlerFunc func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

// Execute executes the step.
func (f StepHandlerFunc) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return f(ctx, input)
}

// GetName returns the handler name.
func (f StepHandlerFunc) GetName() string {
	return "StepHandlerFunc"
}

// StepHandlerRegistry manages step handlers.
type StepHandlerRegistry struct {
	handlers map[string]StepHandler
}

// NewStepHandlerRegistry creates a new handler registry.
func NewStepHandlerRegistry() *StepHandlerRegistry {
	return &StepHandlerRegistry{
		handlers: make(map[string]StepHandler),
	}
}

// Register registers a step handler.
func (r *StepHandlerRegistry) Register(name string, handler StepHandler) error {
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("handler %s already registered", name)
	}
	r.handlers[name] = handler
	return nil
}

// RegisterFunc registers a function as a step handler.
func (r *StepHandlerRegistry) RegisterFunc(name string, fn StepHandlerFunc) error {
	return r.Register(name, fn)
}

// Get retrieves a step handler by name.
func (r *StepHandlerRegistry) Get(name string) (StepHandler, error) {
	handler, exists := r.handlers[name]
	if !exists {
		return nil, fmt.Errorf("handler %s not found", name)
	}
	return handler, nil
}

// Has checks if a handler exists.
func (r *StepHandlerRegistry) Has(name string) bool {
	_, exists := r.handlers[name]
	return exists
}

// StepExecutor executes workflow steps.
type StepExecutor struct {
	registry *StepHandlerRegistry
}

// NewStepExecutor creates a new step executor.
func NewStepExecutor(registry *StepHandlerRegistry) *StepExecutor {
	return &StepExecutor{
		registry: registry,
	}
}

// ExecuteStep executes a single workflow step.
func (e *StepExecutor) ExecuteStep(ctx context.Context, step *StepDefinition, input map[string]interface{}) (*StepExecution, error) {
	execution := &StepExecution{
		ID:         generateStepExecutionID(),
		StepID:     step.ID,
		Status:     StepStatusPending,
		Input:      input,
		RetryCount: 0,
	}

	// Get handler
	handler, err := e.registry.Get(step.Handler)
	if err != nil {
		execution.Status = StepStatusFailed
		execution.Error = err.Error()
		return execution, err
	}

	// Execute with timeout if specified
	execution.Status = StepStatusRunning
	execution.StartedAt = time.Now().UTC()

	var result map[string]interface{}
	var execErr error

	if step.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, step.Timeout)
		defer cancel()
		result, execErr = handler.Execute(ctx, input)
	} else {
		result, execErr = handler.Execute(ctx, input)
	}

	completedAt := time.Now().UTC()
	execution.CompletedAt = &completedAt

	if execErr != nil {
		execution.Status = StepStatusFailed
		execution.Error = execErr.Error()
		return execution, execErr
	}

	execution.Status = StepStatusCompleted
	execution.Output = result
	return execution, nil
}

// ExecuteStepWithRetry executes a step with retry logic.
func (e *StepExecutor) ExecuteStepWithRetry(ctx context.Context, step *StepDefinition, input map[string]interface{}) (*StepExecution, error) {
	execution := &StepExecution{
		ID:         generateStepExecutionID(),
		StepID:     step.ID,
		Status:     StepStatusPending,
		Input:      input,
		RetryCount: 0,
	}

	retryPolicy := step.RetryPolicy
	if retryPolicy == nil {
		// Default: no retry
		return e.ExecuteStep(ctx, step, input)
	}

	maxAttempts := retryPolicy.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			execution.RetryCount = attempt
			// Calculate backoff
			delay := calculateBackoff(retryPolicy, attempt)
			time.Sleep(delay)
		}

		result, err := e.ExecuteStep(ctx, step, input)
		if err == nil {
			return result, nil
		}

		lastErr = err
		execution.Error = err.Error()
	}

	execution.Status = StepStatusFailed
	execution.Error = fmt.Sprintf("failed after %d attempts: %v", maxAttempts, lastErr)
	completedAt := time.Now().UTC()
	execution.CompletedAt = &completedAt

	return execution, fmt.Errorf("step %s failed after %d attempts: %w", step.ID, maxAttempts, lastErr)
}

// calculateBackoff calculates the backoff delay for a retry attempt.
func calculateBackoff(policy *RetryPolicy, attempt int) time.Duration {
	if policy == nil || attempt <= 0 {
		return 0
	}

	base := policy.InitialDelay
	if base <= 0 {
		base = time.Second
	}

	var delay time.Duration
	switch policy.BackoffType {
	case BackoffFixed:
		delay = base
	case BackoffLinear:
		delay = base * time.Duration(attempt)
	case BackoffExponential:
		delay = base * time.Duration(1<<uint(attempt-1))
	default:
		delay = base
	}

	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	return delay
}

// generateStepExecutionID generates a unique step execution ID.
func generateStepExecutionID() string {
	return fmt.Sprintf("step-exec-%d", time.Now().UnixNano())
}

// StepResult represents the result of a step execution.
type StepResult struct {
	Success bool
	Output  map[string]interface{}
	Error   error
	NextStep string // For decision steps
}

// DecisionHandler is a specialized handler for decision steps.
type DecisionHandler interface {
	StepHandler
	// Decide evaluates conditions and returns the next step ID.
	Decide(ctx context.Context, input map[string]interface{}) (string, error)
}

// ParallelStepHandler is a specialized handler for parallel steps.
type ParallelStepHandler interface {
	StepHandler
	// ExecuteParallel executes multiple branches in parallel.
	ExecuteParallel(ctx context.Context, branches []map[string]interface{}) ([]map[string]interface{}, error)
}
