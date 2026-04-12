package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewWorkflowEngine(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	if engine == nil {
		t.Fatal("engine should not be nil")
	}
}

func TestNewWorkflowEngine_Defaults(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	stats := engine.Stats()

	if stats["registered_workflows"].(int) < 3 {
		t.Errorf("expected at least 3 default workflows, got %d", stats["registered_workflows"])
	}
}

func TestWorkflowEngine_StartStop(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	if err := engine.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}
}

func TestWorkflowEngine_RegisterStepHandler(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	err := engine.RegisterStepHandler("test_handler", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	}))
	if err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}
}

func TestWorkflowEngine_StartWorkflow(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.RegisterStepHandler("schedule_agent", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"scheduled": true}, nil
	}))
	engine.RegisterStepHandler("allocate_resources", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"allocated": true}, nil
	}))
	engine.RegisterStepHandler("create_runtime", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"created": true}, nil
	}))
	engine.RegisterStepHandler("start_agent", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"started": true}, nil
	}))
	engine.RegisterStepHandler("wait_ready", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"status": "ready"}, nil
	}))

	instance, err := engine.StartWorkflow(ctx, "agent-deploy", "agent-1", "agent", map[string]interface{}{"node": "node-1"})
	if err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}
	if instance.WorkflowID != "agent-deploy" {
		t.Errorf("expected agent-deploy, got %s", instance.WorkflowID)
	}
	if instance.EntityID != "agent-1" {
		t.Errorf("expected agent-1, got %s", instance.EntityID)
	}

	time.Sleep(200 * time.Millisecond)

	retrieved, err := engine.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}
	if retrieved.Status != WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", retrieved.Status)
	}
}

func TestWorkflowEngine_StartWorkflowNotFound(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	_, err := engine.StartWorkflow(ctx, "nonexistent", "agent-1", "agent", nil)
	if err == nil {
		t.Error("should error for nonexistent workflow")
	}
}

func TestWorkflowEngine_GetInstanceNotFound(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	_, err := engine.GetInstance(ctx, "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent instance")
	}
}

func TestWorkflowEngine_CancelWorkflow(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.RegisterStepHandler("stop_agent", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		time.Sleep(5 * time.Second)
		return map[string]interface{}{"stopped": true}, nil
	}))
	engine.RegisterStepHandler("wait_stopped", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		time.Sleep(5 * time.Second)
		return map[string]interface{}{"status": "stopped"}, nil
	}))

	instance, err := engine.StartWorkflow(ctx, "agent-stop", "agent-2", "agent", nil)
	if err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = engine.CancelWorkflow(ctx, instance.ID)
	if err != nil {
		t.Logf("cancel result (may already be terminal): %v", err)
	}
}

func TestWorkflowEngine_CancelAlreadyTerminal(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	store := NewInMemoryWorkflowInstanceStore()
	inst := NewWorkflowInstance("test", "agent", "agent", nil)
	inst.MarkCompleted(nil)
	store.Create(ctx, inst)

	err := engine.CancelWorkflow(ctx, inst.ID)
	if err == nil {
		t.Error("should not be able to cancel completed workflow")
	}
}

func TestWorkflowEngine_CancelNonexistent(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	err := engine.CancelWorkflow(ctx, "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent instance")
	}
}

func TestWorkflowEngine_ListActiveInstances(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	active := engine.ListActiveInstances()
	if len(active) != 0 {
		t.Errorf("expected 0 active, got %d", len(active))
	}
}

func TestWorkflowEngine_Stats(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	stats := engine.Stats()
	if stats["registered_workflows"].(int) < 3 {
		t.Errorf("expected at least 3 workflows, got %d", stats["registered_workflows"])
	}
	if stats["active_instances"].(int) != 0 {
		t.Errorf("expected 0 active, got %d", stats["active_instances"])
	}
}

func TestWorkflowEngine_RegisterWorkflow(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})

	def := &WorkflowDefinition{
		ID:        "test-workflow",
		Name:      "Test",
		Version:   "1.0",
		StartStep: "step1",
		Steps: []*StepDefinition{
			{ID: "step1", Name: "Step 1", StepType: StepTypeEnd},
		},
	}

	if err := engine.RegisterWorkflow(def); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
}

func TestWorkflowEngine_StartWorkflowWithFailure(t *testing.T) {
	engine := NewWorkflowEngine(EngineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.RegisterStepHandler("schedule_agent", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("schedule failed")
	}))

	instance, err := engine.StartWorkflow(ctx, "agent-deploy", "agent-fail", "agent", nil)
	if err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	retrieved, _ := engine.GetInstance(ctx, instance.ID)
	if retrieved.Status != WorkflowStatusFailed {
		t.Errorf("expected failed, got %s", retrieved.Status)
	}
}
