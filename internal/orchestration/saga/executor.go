package saga

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SagaExecutor executes saga steps.
type SagaExecutor struct {
	logger *zap.Logger
}

// NewSagaExecutor creates a new saga executor.
func NewSagaExecutor() *SagaExecutor {
	return &SagaExecutor{
		logger: zap.NewNop(),
	}
}

// WithLogger sets the logger.
func (e *SagaExecutor) WithLogger(logger *zap.Logger) *SagaExecutor {
	e.logger = logger
	return e
}

// ExecuteSequential executes saga steps sequentially.
func (e *SagaExecutor) ExecuteSequential(ctx context.Context, instance *SagaInstance, saga *Saga) error {
	for _, step := range saga.Steps {
		select {
		case <-ctx.Done():
			instance.MarkFailed(ctx.Err())
			return ctx.Err()
		default:
		}

		result := e.executeStep(ctx, instance, step)
		instance.AddStepResult(result)

		if !result.Success {
			instance.MarkFailed(fmt.Errorf("step %s failed: %s", step.ID, result.Error))
			return fmt.Errorf("saga step %s failed: %s", step.ID, result.Error)
		}
	}

	instance.MarkCompleted()
	return nil
}

// ExecuteParallel executes saga steps respecting dependencies.
func (e *SagaExecutor) ExecuteParallel(ctx context.Context, instance *SagaInstance, saga *Saga) error {
	// Build dependency graph
	stepMap := make(map[string]*SagaStep)
	dependencies := make(map[string]map[string]bool) // stepID -> {dependencyID -> completed}
	dependents := make(map[string][]string)          // stepID -> list of steps that depend on it

	for _, step := range saga.Steps {
		stepMap[step.ID] = step
		dependencies[step.ID] = make(map[string]bool)
		for _, dep := range step.Dependencies {
			dependencies[step.ID][dep] = false
			dependents[dep] = append(dependents[dep], step.ID)
		}
	}

	// Find initially runnable steps (no dependencies)
	var ready []*SagaStep
	for _, step := range saga.Steps {
		if len(step.Dependencies) == 0 {
			ready = append(ready, step)
		}
	}

	if len(ready) == 0 && len(saga.Steps) > 0 {
		return fmt.Errorf("circular dependency detected or all steps have dependencies")
	}

	// Execute steps with worker pool
	var wg sync.WaitGroup
	resultCh := make(chan *StepResult, len(saga.Steps))
	completed := make(map[string]bool)
	var mu sync.Mutex
	var firstError error

	executeReadySteps := func(steps []*SagaStep) {
		for _, step := range steps {
			wg.Add(1)
			go func(s *SagaStep) {
				defer wg.Done()

				result := e.executeStep(ctx, instance, s)
				resultCh <- result

				mu.Lock()
				completed[s.ID] = true

				// Check if any dependent steps are now ready
				for _, depID := range dependents[s.ID] {
					deps := dependencies[depID]
					deps[s.ID] = true

					// Check if all dependencies are met
					allMet := true
					for _, met := range deps {
						if !met {
							allMet = false
							break
						}
					}

					if allMet && firstError == nil {
						// Execute this step
						go func(depStep *SagaStep) {
							wg.Add(1)
							defer wg.Done()
							result := e.executeStep(ctx, instance, depStep)
							resultCh <- result

							mu.Lock()
							completed[depStep.ID] = true
							mu.Unlock()
						}(stepMap[depID])
					}
				}
				mu.Unlock()
			}(step)
		}
	}

	// Execute initially ready steps
	executeReadySteps(ready)

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for result := range resultCh {
		instance.AddStepResult(result)

		if !result.Success && firstError == nil {
			mu.Lock()
			firstError = fmt.Errorf("step %s failed: %s", result.StepID, result.Error)
			mu.Unlock()
		}
	}

	if firstError != nil {
		instance.MarkFailed(firstError)
		return firstError
	}

	instance.MarkCompleted()
	return nil
}

// executeStep executes a single saga step.
func (e *SagaExecutor) executeStep(ctx context.Context, instance *SagaInstance, step *SagaStep) *StepResult {
	result := &StepResult{
		StepID:    step.ID,
		Success:   true,
		StartedAt: time.Now().UTC(),
	}

	// Prepare input
	input := make(map[string]interface{})
	for k, v := range instance.Input {
		input[k] = v
	}
	for k, v := range instance.Output {
		input[k] = v
	}

	// Execute with retry
	var output map[string]interface{}
	var err error

	maxAttempts := 1
	if step.RetryPolicy != nil && step.RetryPolicy.MaxAttempts > 0 {
		maxAttempts = step.RetryPolicy.MaxAttempts
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(step.RetryPolicy, attempt)
			time.Sleep(delay)
		}

		// Apply timeout if specified
		execCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		output, err = step.Action.Execute(execCtx, input)
		if err == nil {
			break
		}

		e.logger.Warn("Step execution failed, retrying",
			zap.String("step_id", step.ID),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	completedAt := time.Now().UTC()
	result.CompletedAt = &completedAt

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Output = output

	// Update instance output
	for k, v := range output {
		instance.Output[k] = v
	}

	return result
}

// CompensationResult represents the result of a compensation.
type CompensationResult struct {
	AllCompensated       bool
	FailedCompensations  []string
	CompensatedSteps     []string
}

// Compensate compensates a failed saga.
func (e *SagaExecutor) Compensate(ctx context.Context, instance *SagaInstance, saga *Saga) *CompensationResult {
	result := &CompensationResult{
		AllCompensated:      true,
		FailedCompensations: make([]string, 0),
		CompensatedSteps:    make([]string, 0),
	}

	// Get completed steps in reverse order
	completedSteps := e.getCompletedStepsInOrder(instance, saga)

	// Compensate in reverse order
	for i := len(completedSteps) - 1; i >= 0; i-- {
		step := completedSteps[i]

		if step.Compensation == nil {
			e.logger.Debug("Step has no compensation, skipping",
				zap.String("step_id", step.ID),
			)
			continue
		}

		e.logger.Info("Compensating step",
			zap.String("step_id", step.ID),
		)

		// Prepare compensation input
		input := make(map[string]interface{})
		for k, v := range instance.Input {
			input[k] = v
		}

		// Add original output
		if stepResult, exists := instance.StepResults[step.ID]; exists {
			for k, v := range stepResult.Output {
				input["original_"+k] = v
			}
		}

		// Execute compensation with timeout
		compCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			compCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		_, err := step.Compensation.Execute(compCtx, input)

		record := &CompensationRecord{
			StepID:     step.ID,
			ExecutedAt: time.Now().UTC(),
		}

		if err != nil {
			result.AllCompensated = false
			result.FailedCompensations = append(result.FailedCompensations, step.ID)
			record.Success = false
			record.Error = err.Error()

			e.logger.Error("Compensation failed",
				zap.String("step_id", step.ID),
				zap.Error(err),
			)
		} else {
			result.CompensatedSteps = append(result.CompensatedSteps, step.ID)
			record.Success = true

			// Update step result
			if stepResult, exists := instance.StepResults[step.ID]; exists {
				stepResult.Compensated = true
			}

			e.logger.Info("Compensation succeeded",
				zap.String("step_id", step.ID),
			)
		}

		instance.AddCompensationRecord(record)
	}

	return result
}

// getCompletedStepsInOrder returns completed steps in their original execution order.
func (e *SagaExecutor) getCompletedStepsInOrder(instance *SagaInstance, saga *Saga) []*SagaStep {
	var completed []*SagaStep

	for _, step := range saga.Steps {
		if result, exists := instance.StepResults[step.ID]; exists && result.Success {
			completed = append(completed, step)
		}
	}

	return completed
}

// calculateBackoff calculates the backoff delay.
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
