package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/agentos/aos/internal/messaging"
	"go.uber.org/zap"
)

// WorkflowEngine is the core workflow orchestration engine.
type WorkflowEngine struct {
	registry      *WorkflowRegistry
	stepRegistry  *StepHandlerRegistry
	store         WorkflowInstanceStore
	compensation  *CompensationManager
	eventBus      messaging.EventBus
	logger        *zap.Logger
	mu            sync.RWMutex
	instances     map[string]*WorkflowInstance // Active instances
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// EngineOptions provides options for creating a workflow engine.
type EngineOptions struct {
	Registry     *WorkflowRegistry
	StepRegistry *StepHandlerRegistry
	Store        WorkflowInstanceStore
	EventBus     messaging.EventBus
	Logger       *zap.Logger
}

// NewWorkflowEngine creates a new workflow engine.
func NewWorkflowEngine(opts EngineOptions) *WorkflowEngine {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	registry := opts.Registry
	if registry == nil {
		registry = NewWorkflowRegistry()
		// Register standard workflows
		_ = registry.Register(AgentDeployWorkflow())
		_ = registry.Register(AgentStopWorkflow())
		_ = registry.Register(AgentDeleteWorkflow())
	}

	stepRegistry := opts.StepRegistry
	if stepRegistry == nil {
		stepRegistry = NewStepHandlerRegistry()
	}

	store := opts.Store
	if store == nil {
		store = NewInMemoryWorkflowInstanceStore()
	}

	compensation := NewCompensationManager(stepRegistry, logger)

	return &WorkflowEngine{
		registry:     registry,
		stepRegistry: stepRegistry,
		store:        store,
		compensation: compensation,
		eventBus:     opts.EventBus,
		logger:       logger,
		instances:    make(map[string]*WorkflowInstance),
		stopCh:       make(chan struct{}),
	}
}

// Start starts the workflow engine.
func (e *WorkflowEngine) Start(ctx context.Context) error {
	e.logger.Info("Starting workflow engine")

	// Register default step handlers
	e.registerDefaultHandlers()

	return nil
}

// Stop stops the workflow engine.
func (e *WorkflowEngine) Stop() error {
	e.logger.Info("Stopping workflow engine")
	close(e.stopCh)
	e.wg.Wait()
	e.logger.Info("Workflow engine stopped")
	return nil
}

// registerDefaultHandlers registers built-in step handlers.
func (e *WorkflowEngine) registerDefaultHandlers() {
	// These are placeholder handlers that will be overridden by actual implementations
	// or integrations with other components

	// Decision handler
	e.stepRegistry.Register("check_running", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// This would check if an agent is running
		return map[string]interface{}{"decision": "running"}, nil
	}))

	// Wait handlers
	e.stepRegistry.Register("wait_ready", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// This would wait for an agent to be ready
		return map[string]interface{}{"status": "ready"}, nil
	}))

	e.stepRegistry.Register("wait_stopped", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// This would wait for an agent to be stopped
		return map[string]interface{}{"status": "stopped"}, nil
	}))
}

// StartWorkflow starts a new workflow instance.
func (e *WorkflowEngine) StartWorkflow(ctx context.Context, workflowID, entityID, entityType string, input map[string]interface{}) (*WorkflowInstance, error) {
	// Get workflow definition
	def, err := e.registry.Get(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found: %w", workflowID, err)
	}

	// Create instance
	instance := NewWorkflowInstance(workflowID, entityID, entityType, input)
	instance.Status = WorkflowStatusRunning
	instance.CurrentStep = def.StartStep

	// Store instance
	if err := e.store.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}

	e.mu.Lock()
	e.instances[instance.ID] = instance
	e.mu.Unlock()

	// Publish event
	if e.eventBus != nil {
		event, _ := messaging.NewEvent(messaging.EventWorkflowStarted, map[string]interface{}{
			"workflow_id": workflowID,
			"instance_id": instance.ID,
			"entity_id":   entityID,
			"entity_type": entityType,
		})
		event.SetAgentID(entityID)
		_ = e.eventBus.Publish(ctx, event)
	}

	e.logger.Info("Workflow started",
		zap.String("workflow_id", workflowID),
		zap.String("instance_id", instance.ID),
		zap.String("entity_id", entityID),
	)

	// Start execution in background
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.executeWorkflow(context.Background(), instance, def)
	}()

	return instance, nil
}

// executeWorkflow executes a workflow instance.
func (e *WorkflowEngine) executeWorkflow(ctx context.Context, instance *WorkflowInstance, def *WorkflowDefinition) {
	// Create step executor
	stepExecutor := NewStepExecutor(e.stepRegistry)

	// Build step map
	stepMap := make(map[string]*StepDefinition)
	for _, step := range def.Steps {
		stepMap[step.ID] = step
	}

	workflowTimeout := def.Timeout
	if workflowTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, workflowTimeout)
		defer cancel()
	}

	currentStepID := instance.CurrentStep

	for {
		select {
		case <-ctx.Done():
			e.logger.Warn("Workflow timeout",
				zap.String("instance_id", instance.ID),
			)
			instance.MarkFailed(fmt.Errorf("workflow timeout"))
			e.finalizeWorkflow(ctx, instance, def)
			return

		case <-e.stopCh:
			e.logger.Info("Workflow engine stopping, pausing workflow",
				zap.String("instance_id", instance.ID),
			)
			// Save current state for resumption
			instance.CurrentStep = currentStepID
			_ = e.store.Update(ctx, instance)
			return

		default:
		}

		step, exists := stepMap[currentStepID]
		if !exists {
			instance.MarkFailed(fmt.Errorf("step %s not found", currentStepID))
			e.finalizeWorkflow(ctx, instance, def)
			return
		}

		// Check if end step
		if step.StepType == StepTypeEnd {
			if step.ID == "failed" {
				instance.MarkFailed(fmt.Errorf("workflow failed at step %s", instance.GetLastStepExecution().StepID))
				e.finalizeWorkflow(ctx, instance, def)
				return
			}
			// Success
			instance.MarkCompleted(instance.Variables)
			e.finalizeWorkflow(ctx, instance, def)
			return
		}

		// Execute step
		e.logger.Debug("Executing step",
			zap.String("instance_id", instance.ID),
			zap.String("step_id", step.ID),
		)

		// Prepare input
		input := e.prepareStepInput(instance, step)

		// Execute with retry
		execution, err := stepExecutor.ExecuteStepWithRetry(ctx, step, input)
		instance.AddStepExecution(execution)

		if err != nil {
			e.logger.Error("Step execution failed",
				zap.Error(err),
				zap.String("instance_id", instance.ID),
				zap.String("step_id", step.ID),
			)

			// Try compensation
			if e.compensation.CanCompensate(instance, def) {
				e.logger.Info("Compensating workflow",
					zap.String("instance_id", instance.ID),
				)
				e.compensation.Compensate(ctx, instance, def)
			}

			instance.MarkFailed(err)
			e.finalizeWorkflow(ctx, instance, def)
			return
		}

		// Update instance with outputs
		if execution.Output != nil {
			for k, v := range execution.Output {
				instance.SetVariable(k, v)
			}
		}

		// Determine next step
		nextStepID := e.determineNextStep(instance, step, execution)
		if nextStepID == "" {
			// No next step, workflow complete
			instance.MarkCompleted(instance.Variables)
			e.finalizeWorkflow(ctx, instance, def)
			return
		}

		currentStepID = nextStepID
		instance.CurrentStep = currentStepID

		// Save progress
		_ = e.store.Update(ctx, instance)

		// Publish step completed event
		if e.eventBus != nil {
			event, _ := messaging.NewEvent(messaging.EventWorkflowStepCompleted, map[string]interface{}{
				"workflow_id":  instance.WorkflowID,
				"instance_id":  instance.ID,
				"step_id":      step.ID,
				"next_step":    nextStepID,
				"entity_id":    instance.EntityID,
			})
			event.SetAgentID(instance.EntityID)
			_ = e.eventBus.Publish(ctx, event)
		}
	}
}

// prepareStepInput prepares the input for a step.
func (e *WorkflowEngine) prepareStepInput(instance *WorkflowInstance, step *StepDefinition) map[string]interface{} {
	input := make(map[string]interface{})

	// Add workflow input
	for k, v := range instance.Input {
		input[k] = v
	}

	// Add variables
	for k, v := range instance.Variables {
		input[k] = v
	}

	// Add step-specific inputs
	for k, v := range step.Inputs {
		input[k] = v
	}

	return input
}

// determineNextStep determines the next step based on step type and result.
func (e *WorkflowEngine) determineNextStep(instance *WorkflowInstance, step *StepDefinition, execution *StepExecution) string {
	switch step.StepType {
	case StepTypeDecision:
		// Get decision from output
		if decision, ok := execution.Output["decision"].(string); ok {
			if nextStep, exists := step.Conditions[decision]; exists {
				return nextStep
			}
		}
		// Check for status-based conditions
		if status, ok := execution.Output["status"].(string); ok {
			if nextStep, exists := step.Conditions[status]; exists {
				return nextStep
			}
		}
		// Default condition
		if nextStep, exists := step.Conditions["default"]; exists {
			return nextStep
		}

	case StepTypeWait:
		// Check conditions based on wait result
		if status, ok := execution.Output["status"].(string); ok {
			if nextStep, exists := step.Conditions[status]; exists {
				return nextStep
			}
		}
		// Default: complete
		if nextStep, exists := step.Conditions["ready"]; exists {
			return nextStep
		}

	default:
		// Task or other types: use next steps
		if len(step.NextSteps) > 0 {
			return step.NextSteps[0]
		}
	}

	return ""
}

// finalizeWorkflow finalizes a workflow instance.
func (e *WorkflowEngine) finalizeWorkflow(ctx context.Context, instance *WorkflowInstance, def *WorkflowDefinition) {
	// Update store
	_ = e.store.Update(ctx, instance)

	// Remove from active instances
	e.mu.Lock()
	delete(e.instances, instance.ID)
	e.mu.Unlock()

	// Publish completion event
	var eventType string
	if instance.Status.IsSuccessful() {
		eventType = messaging.EventWorkflowCompleted
	} else if instance.Status == WorkflowStatusCompensated {
		eventType = messaging.EventSagaCompensated
	} else {
		eventType = messaging.EventWorkflowFailed
	}

	if e.eventBus != nil {
		event, _ := messaging.NewEvent(eventType, map[string]interface{}{
			"workflow_id":   instance.WorkflowID,
			"instance_id":   instance.ID,
			"entity_id":     instance.EntityID,
			"status":        instance.Status,
			"duration_ms":   instance.Duration().Milliseconds(),
			"step_count":    instance.GetStepCount(),
			"error":         instance.Error,
		})
		event.SetAgentID(instance.EntityID)
		_ = e.eventBus.Publish(ctx, event)
	}

	status := "completed"
	if !instance.Status.IsSuccessful() {
		status = "failed"
	}

	e.logger.Info("Workflow finalized",
		zap.String("instance_id", instance.ID),
		zap.String("status", status),
		zap.Duration("duration", instance.Duration()),
	)
}

// GetInstance retrieves a workflow instance by ID.
func (e *WorkflowEngine) GetInstance(ctx context.Context, instanceID string) (*WorkflowInstance, error) {
	e.mu.RLock()
	if instance, exists := e.instances[instanceID]; exists {
		e.mu.RUnlock()
		return instance, nil
	}
	e.mu.RUnlock()

	return e.store.Get(ctx, instanceID)
}

// CancelWorkflow cancels a running workflow.
func (e *WorkflowEngine) CancelWorkflow(ctx context.Context, instanceID string) error {
	instance, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return err
	}

	if instance.Status.IsTerminal() {
		return fmt.Errorf("workflow is already in terminal state: %s", instance.Status)
	}

	instance.MarkCancelled()

	if err := e.store.Update(ctx, instance); err != nil {
		return err
	}

	e.mu.Lock()
	delete(e.instances, instanceID)
	e.mu.Unlock()

	e.logger.Info("Workflow cancelled",
		zap.String("instance_id", instanceID),
	)

	return nil
}

// RegisterWorkflow registers a custom workflow definition.
func (e *WorkflowEngine) RegisterWorkflow(def *WorkflowDefinition) error {
	return e.registry.Register(def)
}

// RegisterStepHandler registers a step handler.
func (e *WorkflowEngine) RegisterStepHandler(name string, handler StepHandler) error {
	return e.stepRegistry.Register(name, handler)
}

// ListActiveInstances returns all active workflow instances.
func (e *WorkflowEngine) ListActiveInstances() []*WorkflowInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*WorkflowInstance, 0, len(e.instances))
	for _, instance := range e.instances {
		if !instance.Status.IsTerminal() {
			result = append(result, instance)
		}
	}

	return result
}

// Stats returns engine statistics.
func (e *WorkflowEngine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := 0
	for _, instance := range e.instances {
		if !instance.Status.IsTerminal() {
			active++
		}
	}

	return map[string]interface{}{
		"registered_workflows": len(e.registry.List()),
		"registered_handlers":  len(e.stepRegistry.handlers),
		"active_instances":     active,
		"total_instances":      len(e.instances),
	}
}
