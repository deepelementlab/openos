package saga

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RecoveryManager manages recovery for failed or incomplete sagas.
type RecoveryManager struct {
	coordinator *SagaCoordinator
	store       SagaStore
	logger      *zap.Logger
	maxRetries  int
}

// NewRecoveryManager creates a new recovery manager.
func NewRecoveryManager(coordinator *SagaCoordinator, store SagaStore, logger *zap.Logger) *RecoveryManager {
	return &RecoveryManager{
		coordinator: coordinator,
		store:       store,
		logger:      logger,
		maxRetries:  3,
	}
}

// RecoveryOptions provides options for recovery.
type RecoveryOptions struct {
	MaxRetries   int
	RetryTimeout time.Duration
	AutoCompensate bool
}

// DefaultRecoveryOptions returns default recovery options.
func DefaultRecoveryOptions() *RecoveryOptions {
	return &RecoveryOptions{
		MaxRetries:     3,
		RetryTimeout:   5 * time.Minute,
		AutoCompensate: true,
	}
}

// Recover recovers a failed saga instance.
func (rm *RecoveryManager) Recover(ctx context.Context, instanceID string, opts *RecoveryOptions) (*RecoveryResult, error) {
	if opts == nil {
		opts = DefaultRecoveryOptions()
	}

	// Get the instance
	instance, err := rm.coordinator.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga instance: %w", err)
	}

	// Get the saga definition
	saga, err := rm.coordinator.GetSaga(instance.SagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga definition: %w", err)
	}

	result := &RecoveryResult{
		InstanceID:   instanceID,
		OriginalStatus: instance.Status,
		StartedAt:    time.Now().UTC(),
		Steps:        make([]RecoveryStep, 0),
	}

	// Check if recovery is possible
	if !rm.CanRecover(instance) {
		result.Success = false
		result.Error = "saga cannot be recovered from current state"
		result.CompletedAt = time.Now().UTC()
		return result, nil
	}

	rm.logger.Info("Starting saga recovery",
		zap.String("instance_id", instanceID),
		zap.String("saga_id", saga.ID),
		zap.String("original_status", string(instance.Status)),
	)

	// Handle based on current status
	switch instance.Status {
	case SagaStatusFailed:
		result = rm.recoverFailedSaga(ctx, instance, saga, opts, result)

	case SagaStatusPartiallyCompensated:
		result = rm.recoverPartiallyCompensated(ctx, instance, saga, opts, result)

	case SagaStatusRunning:
		// Saga is still running but might be stuck
		result = rm.recoverStuckSaga(ctx, instance, saga, opts, result)

	case SagaStatusCompensating:
		// Compensation is in progress, check if stuck
		result = rm.recoverStuckCompensation(ctx, instance, saga, opts, result)
	}

	result.CompletedAt = time.Now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Save final state
	if rm.store != nil {
		_ = rm.store.Save(ctx, instance)
	}

	rm.logger.Info("Saga recovery completed",
		zap.String("instance_id", instanceID),
		zap.Bool("success", result.Success),
		zap.String("final_status", string(instance.Status)),
	)

	return result, nil
}

// RecoveryResult represents the result of a recovery attempt.
type RecoveryResult struct {
	InstanceID     string         `json:"instance_id"`
	Success        bool           `json:"success"`
	OriginalStatus SagaStatus     `json:"original_status"`
	FinalStatus    SagaStatus     `json:"final_status"`
	Steps          []RecoveryStep `json:"steps"`
	Error          string         `json:"error,omitempty"`
	StartedAt      time.Time      `json:"started_at"`
	CompletedAt    time.Time      `json:"completed_at"`
	Duration       time.Duration  `json:"duration"`
}

// RecoveryStep represents a single recovery step.
type RecoveryStep struct {
	Action      string    `json:"action"`
	StepID      string    `json:"step_id,omitempty"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// CanRecover checks if a saga instance can be recovered.
func (rm *RecoveryManager) CanRecover(instance *SagaInstance) bool {
	// Cannot recover completed or fully compensated sagas
	if instance.Status == SagaStatusCompleted ||
		instance.Status == SagaStatusCompensated {
		return false
	}

	// Can recover failed, partially compensated, stuck running, or stuck compensating sagas
	return instance.Status == SagaStatusFailed ||
		instance.Status == SagaStatusPartiallyCompensated ||
		instance.Status == SagaStatusRunning ||
		instance.Status == SagaStatusCompensating
}

// recoverFailedSaga recovers a failed saga.
func (rm *RecoveryManager) recoverFailedSaga(ctx context.Context, instance *SagaInstance, saga *Saga, opts *RecoveryOptions, result *RecoveryResult) *RecoveryResult {
	// Option 1: Retry from failed step
	failedStep := instance.GetFailedStep()
	if failedStep == nil {
		result.Success = false
		result.Error = "no failed step found"
		return result
	}

	rm.logger.Info("Retrying failed step",
		zap.String("step_id", failedStep.StepID),
	)

	// Find the step definition
	var stepDef *SagaStep
	for _, step := range saga.Steps {
		if step.ID == failedStep.StepID {
			stepDef = step
			break
		}
	}

	if stepDef == nil {
		result.Success = false
		result.Error = fmt.Sprintf("step definition not found: %s", failedStep.StepID)
		return result
	}

	// Attempt retry
	step := RecoveryStep{
		Action:    "retry_step",
		StepID:    failedStep.StepID,
		StartedAt: time.Now().UTC(),
	}

	executor := NewSagaExecutor().WithLogger(rm.logger)

	// Retry the step
	for attempt := 0; attempt < opts.MaxRetries; attempt++ {
		rm.logger.Info("Retry attempt",
			zap.String("step_id", failedStep.StepID),
			zap.Int("attempt", attempt+1),
		)

		newResult := executor.executeStep(ctx, instance, stepDef)
		instance.AddStepResult(newResult)

		if newResult.Success {
			step.Success = true
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)

			// Continue with remaining steps
			return rm.continueSaga(ctx, instance, saga, stepDef, executor, opts, result)
		}

		if attempt < opts.MaxRetries-1 {
			delay := calculateBackoff(&RetryPolicy{
				MaxAttempts:  opts.MaxRetries,
				BackoffType:  BackoffExponential,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
			}, attempt+1)
			time.Sleep(delay)
		}
	}

	step.Success = false
	step.Error = "max retry attempts exceeded"
	step.CompletedAt = time.Now().UTC()
	result.Steps = append(result.Steps, step)

	// Step still failing, try full compensation
	if opts.AutoCompensate {
		return rm.performFullCompensation(ctx, instance, saga, result)
	}

	result.Success = false
	result.Error = "step retry failed and auto-compensate is disabled"
	return result
}

// continueSaga continues executing remaining saga steps.
func (rm *RecoveryManager) continueSaga(ctx context.Context, instance *SagaInstance, saga *Saga, fromStep *SagaStep, executor *SagaExecutor, opts *RecoveryOptions, result *RecoveryResult) *RecoveryResult {
	// Find the index of the completed step
	startIdx := -1
	for i, step := range saga.Steps {
		if step.ID == fromStep.ID {
			startIdx = i + 1
			break
		}
	}

	if startIdx < 0 || startIdx >= len(saga.Steps) {
		// All steps completed
		instance.MarkCompleted()
		result.Success = true
		result.FinalStatus = instance.Status
		return result
	}

	// Execute remaining steps
	for i := startIdx; i < len(saga.Steps); i++ {
		step := saga.Steps[i]

		stepResult := executor.executeStep(ctx, instance, step)
		instance.AddStepResult(stepResult)

		if !stepResult.Success {
			instance.MarkFailed(fmt.Errorf("step %s failed during recovery", step.ID))

			recoveryStep := RecoveryStep{
				Action:      "continue_step",
				StepID:      step.ID,
				Success:     false,
				Error:       stepResult.Error,
				StartedAt:   stepResult.StartedAt,
				CompletedAt: *stepResult.CompletedAt,
			}
			result.Steps = append(result.Steps, recoveryStep)

			if opts.AutoCompensate {
				return rm.performFullCompensation(ctx, instance, saga, result)
			}

			result.Success = false
			result.Error = fmt.Sprintf("step %s failed during recovery", step.ID)
			return result
		}

		recoveryStep := RecoveryStep{
			Action:      "continue_step",
			StepID:      step.ID,
			Success:     true,
			StartedAt:   stepResult.StartedAt,
			CompletedAt: *stepResult.CompletedAt,
		}
		result.Steps = append(result.Steps, recoveryStep)
	}

	instance.MarkCompleted()
	result.Success = true
	result.FinalStatus = instance.Status
	return result
}

// recoverPartiallyCompensated recovers a partially compensated saga.
func (rm *RecoveryManager) recoverPartiallyCompensated(ctx context.Context, instance *SagaInstance, saga *Saga, opts *RecoveryOptions, result *RecoveryResult) *RecoveryResult {
	rm.logger.Info("Attempting to complete partial compensation",
		zap.String("instance_id", instance.ID),
	)

	step := RecoveryStep{
		Action:    "complete_compensation",
		StartedAt: time.Now().UTC(),
	}

	// Retry compensation for failed steps
	compResult := executor.Compensate(ctx, instance, saga)

	if compResult.AllCompensated {
		instance.MarkCompensated()
		step.Success = true
	} else {
		instance.MarkPartiallyCompensated()
		step.Success = false
		step.Error = fmt.Sprintf("compensation failed for steps: %v", compResult.FailedCompensations)
		result.Success = false
		result.Error = "could not complete compensation"
	}

	step.CompletedAt = time.Now().UTC()
	result.Steps = append(result.Steps, step)
	result.FinalStatus = instance.Status

	return result
}

// recoverStuckSaga recovers a saga that appears to be stuck.
func (rm *RecoveryManager) recoverStuckSaga(ctx context.Context, instance *SagaInstance, saga *Saga, opts *RecoveryOptions, result *RecoveryResult) *RecoveryResult {
	rm.logger.Info("Recovering stuck saga",
		zap.String("instance_id", instance.ID),
		zap.Duration("duration", instance.Duration()),
	)

	// Check if saga is actually stuck (timeout)
	if saga.Timeout > 0 && instance.Duration() > saga.Timeout {
		// Saga timed out
		instance.MarkFailed(fmt.Errorf("saga timeout"))

		if opts.AutoCompensate {
			return rm.performFullCompensation(ctx, instance, saga, result)
		}
	}

	// Try to continue from current state
	// Find the last executed step
	var lastStep *SagaStep
	for i := len(saga.Steps) - 1; i >= 0; i-- {
		step := saga.Steps[i]
		if _, exists := instance.StepResults[step.ID]; exists {
			lastStep = step
			break
		}
	}

	if lastStep == nil {
		// No steps executed yet, start from beginning
		executor := NewSagaExecutor().WithLogger(rm.logger)
		return rm.continueSaga(ctx, instance, saga, &SagaStep{}, executor, opts, result)
	}

	// Continue from last completed step
	executor := NewSagaExecutor().WithLogger(rm.logger)
	return rm.continueSaga(ctx, instance, saga, lastStep, executor, opts, result)
}

// recoverStuckCompensation recovers a stuck compensation.
func (rm *RecoveryManager) recoverStuckCompensation(ctx context.Context, instance *SagaInstance, saga *Saga, opts *RecoveryOptions, result *RecoveryResult) *RecoveryResult {
	rm.logger.Info("Recovering stuck compensation",
		zap.String("instance_id", instance.ID),
	)

	step := RecoveryStep{
		Action:    "retry_compensation",
		StartedAt: time.Now().UTC(),
	}

	// Retry compensation
	compResult := executor.Compensate(ctx, instance, saga)

	if compResult.AllCompensated {
		instance.MarkCompensated()
		step.Success = true
		result.Success = true
	} else {
		instance.MarkPartiallyCompensated()
		step.Success = false
		step.Error = "compensation still failing"
		result.Success = false
		result.Error = "could not complete stuck compensation"
	}

	step.CompletedAt = time.Now().UTC()
	result.Steps = append(result.Steps, step)
	result.FinalStatus = instance.Status

	return result
}

// performFullCompensation performs full compensation for a failed saga.
func (rm *RecoveryManager) performFullCompensation(ctx context.Context, instance *SagaInstance, saga *Saga, result *RecoveryResult) *RecoveryResult {
	rm.logger.Info("Performing full compensation",
		zap.String("instance_id", instance.ID),
	)

	step := RecoveryStep{
		Action:    "full_compensation",
		StartedAt: time.Now().UTC(),
	}

	instance.MarkCompensating()
	executor := NewSagaExecutor().WithLogger(rm.logger)
	compResult := executor.Compensate(ctx, instance, saga)

	if compResult.AllCompensated {
		instance.MarkCompensated()
		step.Success = true
		result.Success = true
	} else {
		instance.MarkPartiallyCompensated()
		step.Success = false
		step.Error = fmt.Sprintf("compensation failed for steps: %v", compResult.FailedCompensations)
		result.Success = false
		result.Error = "full compensation failed"
	}

	step.CompletedAt = time.Now().UTC()
	result.Steps = append(result.Steps, step)
	result.FinalStatus = instance.Status

	return result
}

// executor is a package-level variable for recovery functions
var executor = &SagaExecutor{}
