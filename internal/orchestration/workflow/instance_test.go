package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkflowStatus_IsTerminal(t *testing.T) {
	terminal := []WorkflowStatus{
		WorkflowStatusCompleted,
		WorkflowStatusFailed,
		WorkflowStatusCancelled,
		WorkflowStatusCompensated,
	}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}

	nonTerminal := []WorkflowStatus{WorkflowStatusPending, WorkflowStatusRunning, WorkflowStatusCompensating}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestWorkflowStatus_IsSuccessful(t *testing.T) {
	if !WorkflowStatusCompleted.IsSuccessful() {
		t.Error("completed should be successful")
	}
	if WorkflowStatusFailed.IsSuccessful() {
		t.Error("failed should not be successful")
	}
}

func TestNewWorkflowInstance(t *testing.T) {
	inst := NewWorkflowInstance("wf-1", "agent-1", "agent", map[string]interface{}{"key": "value"})

	if inst.ID == "" {
		t.Error("ID should be generated")
	}
	if inst.WorkflowID != "wf-1" {
		t.Errorf("expected wf-1, got %s", inst.WorkflowID)
	}
	if inst.EntityID != "agent-1" {
		t.Errorf("expected agent-1, got %s", inst.EntityID)
	}
	if inst.Status != WorkflowStatusPending {
		t.Errorf("expected pending, got %s", inst.Status)
	}
	if inst.Input["key"] != "value" {
		t.Error("input not set")
	}
	if inst.Variables == nil {
		t.Error("variables should be initialized")
	}
	if inst.StepHistory == nil {
		t.Error("step history should be initialized")
	}
}

func TestWorkflowInstance_Duration(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)

	if inst.Duration() < 0 {
		t.Error("duration should be non-negative")
	}

	now := time.Now()
	inst.CompletedAt = &now
	if inst.Duration() < 0 {
		t.Error("duration with CompletedAt should be non-negative")
	}
}

func TestWorkflowInstance_Variables(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)

	inst.SetVariable("foo", "bar")
	val, ok := inst.GetVariable("foo")
	if !ok {
		t.Error("variable should exist")
	}
	if val != "bar" {
		t.Errorf("expected bar, got %v", val)
	}

	_, ok = inst.GetVariable("nonexistent")
	if ok {
		t.Error("nonexistent variable should not exist")
	}
}

func TestWorkflowInstance_SetVariableNilMap(t *testing.T) {
	inst := &WorkflowInstance{}
	inst.SetVariable("key", "val")
	if inst.Variables["key"] != "val" {
		t.Error("should create map and set variable")
	}
}

func TestWorkflowInstance_StepHistory(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)

	exec1 := &StepExecution{ID: "exec-1", StepID: "s1", Status: StepStatusCompleted}
	exec2 := &StepExecution{ID: "exec-2", StepID: "s2", Status: StepStatusFailed}

	inst.AddStepExecution(exec1)
	inst.AddStepExecution(exec2)

	if inst.GetStepCount() != 2 {
		t.Errorf("expected 2 steps, got %d", inst.GetStepCount())
	}

	last := inst.GetLastStepExecution()
	if last.StepID != "s2" {
		t.Errorf("expected s2 as last step, got %s", last.StepID)
	}
}

func TestWorkflowInstance_GetLastStepExecutionEmpty(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	if inst.GetLastStepExecution() != nil {
		t.Error("should return nil for empty history")
	}
}

func TestWorkflowInstance_GetFailedSteps(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.AddStepExecution(&StepExecution{ID: "e1", StepID: "s1", Status: StepStatusCompleted})
	inst.AddStepExecution(&StepExecution{ID: "e2", StepID: "s2", Status: StepStatusFailed})
	inst.AddStepExecution(&StepExecution{ID: "e3", StepID: "s3", Status: StepStatusFailed})

	failed := inst.GetFailedSteps()
	if len(failed) != 2 {
		t.Errorf("expected 2 failed steps, got %d", len(failed))
	}
}

func TestWorkflowInstance_MarkCompleted(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkCompleted(map[string]interface{}{"result": "ok"})

	if inst.Status != WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
	if inst.Output["result"] != "ok" {
		t.Error("output not set")
	}
}

func TestWorkflowInstance_MarkFailed(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkFailed(errors.New("test failure"))

	if inst.Status != WorkflowStatusFailed {
		t.Errorf("expected failed, got %s", inst.Status)
	}
	if inst.Error != "test failure" {
		t.Errorf("expected 'test failure', got %s", inst.Error)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestWorkflowInstance_MarkFailedNilError(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkFailed(nil)

	if inst.Error != "" {
		t.Error("error should be empty for nil error")
	}
}

func TestWorkflowInstance_MarkCancelled(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkCancelled()

	if inst.Status != WorkflowStatusCancelled {
		t.Errorf("expected cancelled, got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestWorkflowInstance_MarkCompensating(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkCompensating()

	if inst.Status != WorkflowStatusCompensating {
		t.Errorf("expected compensating, got %s", inst.Status)
	}
}

func TestWorkflowInstance_MarkCompensated(t *testing.T) {
	inst := NewWorkflowInstance("wf", "e", "agent", nil)
	inst.MarkCompensated()

	if inst.Status != WorkflowStatusCompensated {
		t.Errorf("expected compensated, got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestWorkflowInstance_ParentID(t *testing.T) {
	inst := &WorkflowInstance{
		ID:       "inst-1",
		ParentID: "parent-1",
	}
	if inst.ParentID != "parent-1" {
		t.Errorf("expected parent-1, got %s", inst.ParentID)
	}
}

func TestInMemoryWorkflowInstanceStore_CRUD(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst := NewWorkflowInstance("wf", "agent-1", "agent", nil)

	if err := store.Create(ctx, inst); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	retrieved, err := store.Get(ctx, inst.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if retrieved.ID != inst.ID {
		t.Error("IDs should match")
	}

	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent")
	}
}

func TestInMemoryWorkflowInstanceStore_GetByEntity(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst := NewWorkflowInstance("wf", "agent-1", "agent", nil)
	store.Create(ctx, inst)

	retrieved, err := store.GetByEntity(ctx, "agent-1", WorkflowStatusPending)
	if err != nil {
		t.Fatalf("failed to get by entity: %v", err)
	}
	if retrieved.ID != inst.ID {
		t.Error("IDs should match")
	}

	_, err = store.GetByEntity(ctx, "nonexistent", WorkflowStatusPending)
	if err == nil {
		t.Error("should error for nonexistent entity")
	}
}

func TestInMemoryWorkflowInstanceStore_Update(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst := NewWorkflowInstance("wf", "agent-1", "agent", nil)
	store.Create(ctx, inst)

	inst.Status = WorkflowStatusRunning
	if err := store.Update(ctx, inst); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	retrieved, _ := store.Get(ctx, inst.ID)
	if retrieved.Status != WorkflowStatusRunning {
		t.Errorf("expected running, got %s", retrieved.Status)
	}
}

func TestInMemoryWorkflowInstanceStore_List(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	store.Create(ctx, NewWorkflowInstance("wf1", "a1", "agent", nil))
	store.Create(ctx, NewWorkflowInstance("wf2", "a2", "agent", nil))

	instances, err := store.List(ctx, WorkflowInstanceFilter{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("expected 2, got %d", len(instances))
	}
}

func TestInMemoryWorkflowInstanceStore_ListWithFilter(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst1 := NewWorkflowInstance("wf1", "a1", "agent", nil)
	inst1.Status = WorkflowStatusRunning
	store.Create(ctx, inst1)

	inst2 := NewWorkflowInstance("wf2", "a2", "workflow", nil)
	inst2.Status = WorkflowStatusPending
	store.Create(ctx, inst2)

	instances, _ := store.List(ctx, WorkflowInstanceFilter{Status: WorkflowStatusRunning})
	if len(instances) != 1 {
		t.Errorf("expected 1 running, got %d", len(instances))
	}

	instances, _ = store.List(ctx, WorkflowInstanceFilter{EntityType: "agent"})
	if len(instances) != 1 {
		t.Errorf("expected 1 agent, got %d", len(instances))
	}
}

func TestInMemoryWorkflowInstanceStore_Pagination(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		store.Create(ctx, NewWorkflowInstance("wf", "agent", "agent", nil))
	}

	instances, _ := store.List(ctx, WorkflowInstanceFilter{Limit: 2})
	if len(instances) != 2 {
		t.Errorf("expected 2 with limit, got %d", len(instances))
	}

	instances, _ = store.List(ctx, WorkflowInstanceFilter{Offset: 10})
	if len(instances) != 0 {
		t.Errorf("expected 0 with offset beyond results, got %d", len(instances))
	}
}

func TestInMemoryWorkflowInstanceStore_Delete(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst := NewWorkflowInstance("wf", "agent-1", "agent", nil)
	store.Create(ctx, inst)

	if err := store.Delete(ctx, inst.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := store.Get(ctx, inst.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}

func TestInMemoryWorkflowInstanceStore_DeleteNonexistent(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("should error deleting nonexistent")
	}
}

func TestInMemoryWorkflowInstanceStore_ListWithWorkflowFilter(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst1 := NewWorkflowInstance("wf-1", "a1", "agent", nil)
	inst2 := NewWorkflowInstance("wf-2", "a2", "agent", nil)
	store.Create(ctx, inst1)
	store.Create(ctx, inst2)

	result, _ := store.List(ctx, WorkflowInstanceFilter{WorkflowID: "wf-1"})
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

func TestInMemoryWorkflowInstanceStore_ListWithEntityIDFilter(t *testing.T) {
	store := NewInMemoryWorkflowInstanceStore()
	ctx := context.Background()

	inst1 := NewWorkflowInstance("wf", "a1", "agent", nil)
	inst2 := NewWorkflowInstance("wf", "a2", "agent", nil)
	store.Create(ctx, inst1)
	store.Create(ctx, inst2)

	result, _ := store.List(ctx, WorkflowInstanceFilter{EntityID: "a1"})
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}
