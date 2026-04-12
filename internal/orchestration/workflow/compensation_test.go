package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewCompensationManager(t *testing.T) {
	registry := NewStepHandlerRegistry()
	cm := NewCompensationManager(registry, zap.NewNop())
	if cm == nil {
		t.Fatal("compensation manager should not be nil")
	}
}

func TestCompensationManager_CanCompensate_False(t *testing.T) {
	registry := NewStepHandlerRegistry()
	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)

	if cm.CanCompensate(inst, def) {
		t.Error("should not compensate with no completed steps")
	}
}

func TestCompensationManager_CanCompensate_AlreadyCompensated(t *testing.T) {
	registry := NewStepHandlerRegistry()
	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp1"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{StepID: "s1", Status: StepStatusCompleted})
	inst.MarkCompensated()

	if cm.CanCompensate(inst, def) {
		t.Error("should not compensate already compensated instance")
	}
}

func TestCompensationManager_CanCompensate_AlreadyCompensating(t *testing.T) {
	registry := NewStepHandlerRegistry()
	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp1"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{StepID: "s1", Status: StepStatusCompleted})
	inst.MarkCompensating()

	if cm.CanCompensate(inst, def) {
		t.Error("should not compensate already compensating instance")
	}
}

func TestCompensationManager_CanCompensate_True(t *testing.T) {
	registry := NewStepHandlerRegistry()
	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp1"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{StepID: "s1", Status: StepStatusCompleted})

	if !cm.CanCompensate(inst, def) {
		t.Error("should be able to compensate with completed step having compensation")
	}
}

func TestCompensationManager_Compensate(t *testing.T) {
	registry := NewStepHandlerRegistry()
	compensationCalled := false

	registry.Register("comp1", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		compensationCalled = true
		return map[string]interface{}{"compensated": true}, nil
	}))

	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp1"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{
		StepID: "s1",
		Status: StepStatusCompleted,
		Input:  map[string]interface{}{"key": "value"},
		Output: map[string]interface{}{"result": "ok"},
	})

	err := cm.Compensate(context.Background(), inst, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !compensationCalled {
		t.Error("compensation handler should have been called")
	}
	if inst.Status != WorkflowStatusCompensated {
		t.Errorf("expected compensated, got %s", inst.Status)
	}
}

func TestCompensationManager_Compensate_MultipleSteps(t *testing.T) {
	registry := NewStepHandlerRegistry()
	callOrder := []string{}

	registry.Register("comp1", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		callOrder = append(callOrder, "comp1")
		return nil, nil
	}))
	registry.Register("comp2", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		callOrder = append(callOrder, "comp2")
		return nil, nil
	}))

	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp1"},
			{ID: "s2", StepType: StepTypeTask, Handler: "h2", Compensation: "comp2"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{StepID: "s1", Status: StepStatusCompleted, Input: map[string]interface{}{}, Output: map[string]interface{}{}})
	inst.AddStepExecution(&StepExecution{StepID: "s2", Status: StepStatusCompleted, Input: map[string]interface{}{}, Output: map[string]interface{}{}})

	cm.Compensate(context.Background(), inst, def)

	if len(callOrder) != 2 {
		t.Fatalf("expected 2 compensation calls, got %d", len(callOrder))
	}
	if callOrder[0] != "comp2" || callOrder[1] != "comp1" {
		t.Errorf("expected reverse order [comp2, comp1], got %v", callOrder)
	}
}

func TestCompensationManager_Compensate_StepFails(t *testing.T) {
	registry := NewStepHandlerRegistry()
	registry.Register("comp_fail", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("compensation failed")
	}))

	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp_fail"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{
		StepID: "s1",
		Status: StepStatusCompleted,
		Input:  map[string]interface{}{},
		Output: map[string]interface{}{},
	})

	err := cm.Compensate(context.Background(), inst, def)
	if err != nil {
		t.Fatalf("should not return error, continues compensating: %v", err)
	}
	if inst.Status != WorkflowStatusCompensated {
		t.Errorf("expected compensated (best effort), got %s", inst.Status)
	}
}

func TestCompensationManager_CompensateWithResult(t *testing.T) {
	registry := NewStepHandlerRegistry()
	registry.Register("comp_wr", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"undone": true}, nil
	}))

	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp_wr"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{
		StepID: "s1",
		Status: StepStatusCompleted,
		Input:  map[string]interface{}{},
		Output: map[string]interface{}{},
	})

	result := cm.CompensateWithResult(context.Background(), inst, def)

	if !result.Success {
		t.Error("compensation should succeed")
	}
	if len(result.CompensatedSteps) != 1 {
		t.Errorf("expected 1 compensated step, got %d", len(result.CompensatedSteps))
	}
	if len(result.FailedSteps) != 0 {
		t.Errorf("expected 0 failed steps, got %d", len(result.FailedSteps))
	}
}

func TestCompensationManager_CompensateWithResult_Failure(t *testing.T) {
	registry := NewStepHandlerRegistry()
	registry.Register("comp_wr_fail", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("fail")
	}))

	cm := NewCompensationManager(registry, zap.NewNop())

	def := &WorkflowDefinition{
		ID: "test",
		Steps: []*StepDefinition{
			{ID: "s1", StepType: StepTypeTask, Handler: "h1", Compensation: "comp_wr_fail"},
		},
	}

	inst := NewWorkflowInstance("wf", "agent", "agent", nil)
	inst.AddStepExecution(&StepExecution{
		StepID: "s1",
		Status: StepStatusCompleted,
		Input:  map[string]interface{}{},
		Output: map[string]interface{}{},
	})

	result := cm.CompensateWithResult(context.Background(), inst, def)

	if result.Success {
		t.Error("compensation should fail")
	}
	if len(result.FailedSteps) != 1 {
		t.Errorf("expected 1 failed step, got %d", len(result.FailedSteps))
	}
	if inst.Status != WorkflowStatusCompensating {
		t.Errorf("expected compensating (not fully compensated), got %s", inst.Status)
	}
}

func TestDefaultCompensationConfig(t *testing.T) {
	cfg := DefaultCompensationConfig()
	if !cfg.ContinueOnFailure {
		t.Error("ContinueOnFailure should be true by default")
	}
	if cfg.MaxCompensationTime != 5*time.Minute {
		t.Errorf("expected 5m max time, got %v", cfg.MaxCompensationTime)
	}
	if cfg.RetryFailedCompensation {
		t.Error("RetryFailedCompensation should be false by default")
	}
}

func TestCompensationConfig_Fields(t *testing.T) {
	cfg := &CompensationConfig{
		ContinueOnFailure:       false,
		MaxCompensationTime:     time.Minute,
		RetryFailedCompensation: true,
		MaxRetries:              5,
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("expected 5, got %d", cfg.MaxRetries)
	}
}

func TestCompensationResult_Fields(t *testing.T) {
	r := &CompensationResult{
		Success:          true,
		CompensatedSteps: []string{"s1", "s2"},
		FailedSteps:      []string{},
	}
	if !r.Success {
		t.Error("should be successful")
	}
	if len(r.CompensatedSteps) != 2 {
		t.Errorf("expected 2, got %d", len(r.CompensatedSteps))
	}
}

func TestCompensationStep_Fields(t *testing.T) {
	cs := &CompensationStep{
		StepID:              "s1",
		CompensationHandler: "comp1",
		Input:               map[string]interface{}{"in": 1},
		Output:              map[string]interface{}{"out": 1},
	}
	if cs.StepID != "s1" {
		t.Error("StepID not set")
	}
}
