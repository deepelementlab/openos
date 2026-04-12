package workflow

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// CompensationManager manages compensation for failed workflow steps.
type CompensationManager struct {
	stepRegistry *StepHandlerRegistry
	logger       *zap.Logger
}

// NewCompensationManager creates a new compensation manager.
func NewCompensationManager(registry *StepHandlerRegistry, logger *zap.Logger) *CompensationManager {
	return &CompensationManager{
		stepRegistry: registry,
		logger:       logger,
	}
}

// Compensate compensates a workflow instance by running compensation steps.
func (cm *CompensationManager) Compensate(ctx context.Context, instance *WorkflowInstance, definition *WorkflowDefinition) error {
	cm.logger.Info("Starting workflow compensation",
		zap.String("instance_id", instance.ID),
		zap.String("workflow_id", definition.ID),
	)

	instance.MarkCompensating()

	// Build compensation chain
	compensationChain := cm.buildCompensationChain(instance, definition)

	// Execute compensation steps in reverse order
	for i := len(compensationChain) - 1; i >= 0; i-- {
		compStep := compensationChain[i]

		cm.logger.Info("Executing compensation step",
			zap.String("step_id", compStep.StepID),
			zap.String("compensation_handler", compStep.CompensationHandler),
		)

		// Execute compensation
		if err := cm.executeCompensation(ctx, compStep); err != nil {
			cm.logger.Error("Compensation step failed",
				zap.Error(err),
				zap.String("step_id", compStep.StepID),
			)
			// Log but continue - we need to try to compensate as much as possible
		}

		// Record compensation
		compStep.Record.Status = StepStatusCompensated
		instance.AddStepExecution(compStep.Record)
	}

	instance.MarkCompensated()

	cm.logger.Info("Workflow compensation completed",
		zap.String("instance_id", instance.ID),
	)

	return nil
}

// CompensationStep represents a step that needs to be compensated.
type CompensationStep struct {
	StepID              string
	CompensationHandler string
	Input               map[string]interface{}
	Output              map[string]interface{}
	Record              *StepExecution
}

// buildCompensationChain builds the chain of steps that need compensation.
func (cm *CompensationManager) buildCompensationChain(instance *WorkflowInstance, definition *WorkflowDefinition) []*CompensationStep {
	var chain []*CompensationStep

	// Create step definition map
	stepMap := make(map[string]*StepDefinition)
	for _, step := range definition.Steps {
		stepMap[step.ID] = step
	}

	// Find completed steps that have compensation handlers
	for _, exec := range instance.StepHistory {
		if exec.Status != StepStatusCompleted {
			continue
		}

		stepDef, exists := stepMap[exec.StepID]
		if !exists {
			continue
		}

		if stepDef.Compensation == "" {
			continue // No compensation needed
		}

		compStep := &CompensationStep{
			StepID:              exec.StepID,
			CompensationHandler: stepDef.Compensation,
			Input:               exec.Input,
			Output:              exec.Output,
			Record: &StepExecution{
				ID:         generateStepExecutionID(),
				WorkflowID: instance.WorkflowID,
				InstanceID: instance.ID,
				StepID:     exec.StepID,
				Status:     StepStatusRunning,
				StartedAt:  time.Now().UTC(),
			},
		}

		chain = append(chain, compStep)
	}

	return chain
}

// executeCompensation executes a single compensation step.
func (cm *CompensationManager) executeCompensation(ctx context.Context, compStep *CompensationStep) error {
	handler, err := cm.stepRegistry.Get(compStep.CompensationHandler)
	if err != nil {
		return fmt.Errorf("compensation handler %s not found: %w", compStep.CompensationHandler, err)
	}

	// Prepare compensation input (combine original input and output)
	input := make(map[string]interface{})
	for k, v := range compStep.Input {
		input[k] = v
	}
	for k, v := range compStep.Output {
		input["output_"+k] = v
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	output, err := handler.Execute(ctx, input)
	if err != nil {
		compStep.Record.Status = StepStatusFailed
		compStep.Record.Error = err.Error()
		completedAt := time.Now().UTC()
		compStep.Record.CompletedAt = &completedAt
		return fmt.Errorf("compensation failed: %w", err)
	}

	compStep.Output = output
	completedAt := time.Now().UTC()
	compStep.Record.CompletedAt = &completedAt

	return nil
}

// CanCompensate checks if a workflow instance can be compensated.
func (cm *CompensationManager) CanCompensate(instance *WorkflowInstance, definition *WorkflowDefinition) bool {
	// Cannot compensate if already compensated or compensating
	if instance.Status == WorkflowStatusCompensated ||
		instance.Status == WorkflowStatusCompensating {
		return false
	}

	// Check if any executed steps have compensation handlers
	stepMap := make(map[string]*StepDefinition)
	for _, step := range definition.Steps {
		stepMap[step.ID] = step
	}

	for _, exec := range instance.StepHistory {
		if exec.Status != StepStatusCompleted {
			continue
		}
		if stepDef, exists := stepMap[exec.StepID]; exists {
			if stepDef.Compensation != "" {
				return true
			}
		}
	}

	return false
}

// CompensationResult represents the result of a compensation operation.
type CompensationResult struct {
	Success         bool
	CompensatedSteps []string
	FailedSteps      []string
	Duration         time.Duration
	StartedAt        time.Time
	CompletedAt      time.Time
}

// CompensateWithResult compensates a workflow and returns detailed results.
func (cm *CompensationManager) CompensateWithResult(ctx context.Context, instance *WorkflowInstance, definition *WorkflowDefinition) *CompensationResult {
	result := &CompensationResult{
		Success:          true,
		CompensatedSteps: make([]string, 0),
		FailedSteps:      make([]string, 0),
		StartedAt:        time.Now().UTC(),
	}

	instance.MarkCompensating()

	compensationChain := cm.buildCompensationChain(instance, definition)

	for i := len(compensationChain) - 1; i >= 0; i-- {
		compStep := compensationChain[i]

		if err := cm.executeCompensation(ctx, compStep); err != nil {
			result.Success = false
			result.FailedSteps = append(result.FailedSteps, compStep.StepID)
		} else {
			result.CompensatedSteps = append(result.CompensatedSteps, compStep.StepID)
		}

		instance.AddStepExecution(compStep.Record)
	}

	result.CompletedAt = time.Now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if result.Success {
		instance.MarkCompensated()
	}

	return result
}

// CompensationConfig provides configuration for compensation behavior.
type CompensationConfig struct {
	// ContinueOnFailure allows compensation to continue even if a step fails.
	ContinueOnFailure bool

	// MaxCompensationTime is the maximum time allowed for all compensation.
	MaxCompensationTime time.Duration

	// RetryFailedCompensation allows retrying failed compensation steps.
	RetryFailedCompensation bool

	// MaxRetries is the maximum number of retries for failed compensation.
	MaxRetries int
}

// DefaultCompensationConfig returns the default compensation configuration.
func DefaultCompensationConfig() *CompensationConfig {
	return &CompensationConfig{
		ContinueOnFailure:       true,
		MaxCompensationTime:     5 * time.Minute,
		RetryFailedCompensation: false,
		MaxRetries:              1,
	}
}
