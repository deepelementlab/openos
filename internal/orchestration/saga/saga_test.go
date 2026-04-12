package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutionModeConstants(t *testing.T) {
	if Sequential != "sequential" {
		t.Errorf("expected sequential, got %s", string(Sequential))
	}
	if Parallel != "parallel" {
		t.Errorf("expected parallel, got %s", string(Parallel))
	}
}

func TestBackoffTypeConstants(t *testing.T) {
	if BackoffFixed != "fixed" {
		t.Errorf("expected fixed, got %s", string(BackoffFixed))
	}
	if BackoffLinear != "linear" {
		t.Errorf("expected linear, got %s", string(BackoffLinear))
	}
	if BackoffExponential != "exponential" {
		t.Errorf("expected exponential, got %s", string(BackoffExponential))
	}
}

func TestSagaActionFunc(t *testing.T) {
	action := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"echo": input["msg"]}, nil
	})

	if action.GetName() != "SagaActionFunc" {
		t.Errorf("expected SagaActionFunc, got %s", action.GetName())
	}

	result, err := action.Execute(context.Background(), map[string]interface{}{"msg": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["echo"] != "hello" {
		t.Errorf("expected hello, got %v", result["echo"])
	}
}

func TestSagaStatus_IsTerminal(t *testing.T) {
	terminal := []SagaStatus{
		SagaStatusCompleted,
		SagaStatusFailed,
		SagaStatusCompensated,
		SagaStatusPartiallyCompensated,
	}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}

	nonTerminal := []SagaStatus{SagaStatusPending, SagaStatusRunning, SagaStatusCompensating}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestNewSagaInstance(t *testing.T) {
	inst := NewSagaInstance("saga-1", map[string]interface{}{"key": "value"})

	if inst.ID == "" {
		t.Error("ID should be generated")
	}
	if inst.SagaID != "saga-1" {
		t.Errorf("expected saga-1, got %s", inst.SagaID)
	}
	if inst.Status != SagaStatusPending {
		t.Errorf("expected pending, got %s", inst.Status)
	}
	if inst.Input["key"] != "value" {
		t.Error("input not set")
	}
	if inst.Output == nil {
		t.Error("output should be initialized")
	}
	if inst.StepResults == nil {
		t.Error("step results should be initialized")
	}
	if inst.CompensationLog == nil {
		t.Error("compensation log should be initialized")
	}
}

func TestSagaInstance_Duration(t *testing.T) {
	inst := NewSagaInstance("saga", nil)

	if inst.Duration() < 0 {
		t.Error("duration should be non-negative")
	}

	now := time.Now()
	inst.CompletedAt = &now
	if inst.Duration() < 0 {
		t.Error("duration with CompletedAt should be non-negative")
	}
}

func TestSagaInstance_StepResults(t *testing.T) {
	inst := NewSagaInstance("saga", nil)

	sr := &StepResult{
		StepID:  "s1",
		Success: true,
		Output:  map[string]interface{}{"result": "ok"},
	}
	inst.AddStepResult(sr)

	retrieved, exists := inst.GetStepResult("s1")
	if !exists {
		t.Error("step result should exist")
	}
	if retrieved.Output["result"] != "ok" {
		t.Error("output not set correctly")
	}

	_, exists = inst.GetStepResult("nonexistent")
	if exists {
		t.Error("nonexistent step should not exist")
	}
}

func TestSagaInstance_CompensationLog(t *testing.T) {
	inst := NewSagaInstance("saga", nil)

	record := &CompensationRecord{
		StepID:     "s1",
		Success:    true,
		ExecutedAt: time.Now(),
	}
	inst.AddCompensationRecord(record)

	if len(inst.CompensationLog) != 1 {
		t.Errorf("expected 1 record, got %d", len(inst.CompensationLog))
	}
}

func TestSagaInstance_MarkCompleted(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkCompleted()

	if inst.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestSagaInstance_MarkFailed(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkFailed(errors.New("saga test error"))

	if inst.Status != SagaStatusFailed {
		t.Errorf("expected failed, got %s", inst.Status)
	}
	if inst.Error != "saga test error" {
		t.Errorf("expected 'saga test error', got %s", inst.Error)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestSagaInstance_MarkFailedNil(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkFailed(nil)

	if inst.Error != "" {
		t.Error("error should be empty for nil")
	}
}

func TestSagaInstance_MarkCompensating(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkCompensating()
	if inst.Status != SagaStatusCompensating {
		t.Errorf("expected compensating, got %s", inst.Status)
	}
}

func TestSagaInstance_MarkCompensated(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkCompensated()
	if inst.Status != SagaStatusCompensated {
		t.Errorf("expected compensated, got %s", inst.Status)
	}
	if inst.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestSagaInstance_MarkPartiallyCompensated(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.MarkPartiallyCompensated()
	if inst.Status != SagaStatusPartiallyCompensated {
		t.Errorf("expected partially_compensated, got %s", inst.Status)
	}
}

func TestSagaInstance_GetCompletedSteps(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.AddStepResult(&StepResult{StepID: "s1", Success: true})
	inst.AddStepResult(&StepResult{StepID: "s2", Success: false})
	inst.AddStepResult(&StepResult{StepID: "s3", Success: true})

	completed := inst.GetCompletedSteps()
	if len(completed) != 2 {
		t.Errorf("expected 2 completed, got %d", len(completed))
	}
}

func TestSagaInstance_GetFailedStep(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.AddStepResult(&StepResult{StepID: "s1", Success: true})
	inst.AddStepResult(&StepResult{StepID: "s2", Success: false})

	failed := inst.GetFailedStep()
	if failed == nil {
		t.Fatal("should find failed step")
	}
	if failed.StepID != "s2" {
		t.Errorf("expected s2, got %s", failed.StepID)
	}
}

func TestSagaInstance_GetFailedStep_None(t *testing.T) {
	inst := NewSagaInstance("saga", nil)
	inst.AddStepResult(&StepResult{StepID: "s1", Success: true})

	if inst.GetFailedStep() != nil {
		t.Error("should return nil when no failed steps")
	}
}

func TestSagaBuilder(t *testing.T) {
	action := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"done": true}, nil
	})
	compensation := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	saga, err := NewSagaBuilder("test-saga", "Test Saga").
		Description("A test saga").
		Version("2.0").
		Timeout(10*time.Minute).
		Step("s1", "Step 1", action, compensation).
		Step("s2", "Step 2", action, nil).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saga.ID != "test-saga" {
		t.Errorf("expected test-saga, got %s", saga.ID)
	}
	if saga.Name != "Test Saga" {
		t.Errorf("expected Test Saga, got %s", saga.Name)
	}
	if saga.Version != "2.0" {
		t.Errorf("expected 2.0, got %s", saga.Version)
	}
	if saga.Timeout != 10*time.Minute {
		t.Errorf("expected 10m timeout, got %v", saga.Timeout)
	}
	if len(saga.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(saga.Steps))
	}
}

func TestSagaBuilder_Parallel(t *testing.T) {
	action := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	saga := NewSagaBuilder("p-saga", "Parallel Saga").
		Parallel().
		StepWithDeps("s1", "Step 1", action, nil, nil).
		StepWithDeps("s2", "Step 2", action, nil, []string{"s1"}).
		MustBuild()

	if saga.ExecutionMode != Parallel {
		t.Errorf("expected parallel, got %s", saga.ExecutionMode)
	}
}

func TestSagaBuilder_NoSteps(t *testing.T) {
	_, err := NewSagaBuilder("empty", "Empty").Build()
	if err == nil {
		t.Error("should reject saga without steps")
	}
}

func TestSagaBuilder_DuplicateStepID(t *testing.T) {
	action := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	_, err := NewSagaBuilder("dup", "Dup").
		Step("s1", "S1", action, nil).
		Step("s1", "S1 again", action, nil).
		Build()
	if err == nil {
		t.Error("should reject duplicate step IDs")
	}
}

func TestSagaBuilder_InvalidDependency(t *testing.T) {
	action := SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	_, err := NewSagaBuilder("bad-dep", "Bad Dep").
		StepWithDeps("s1", "S1", action, nil, []string{"nonexistent"}).
		Build()
	if err == nil {
		t.Error("should reject invalid dependency")
	}
}

func TestSagaBuilder_MustBuild(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic for invalid saga")
		}
	}()

	NewSagaBuilder("panic", "Panic").MustBuild()
}

func TestSagaStep_Fields(t *testing.T) {
	step := &SagaStep{
		ID:             "step-1",
		Name:           "Test Step",
		CompensationID: "comp-1",
		Dependencies:   []string{"dep1"},
		Timeout:        30 * time.Second,
		RetryPolicy: &RetryPolicy{
			MaxAttempts:  3,
			BackoffType:  BackoffExponential,
			InitialDelay: time.Second,
			MaxDelay:     time.Minute,
		},
		Metadata: map[string]interface{}{"priority": "high"},
	}

	if step.ID != "step-1" {
		t.Error("ID not set")
	}
	if step.Timeout != 30*time.Second {
		t.Error("Timeout not set")
	}
	if step.RetryPolicy.MaxAttempts != 3 {
		t.Error("RetryPolicy not set")
	}
}

func TestRetryPolicy_Fields(t *testing.T) {
	p := &RetryPolicy{
		MaxAttempts:  5,
		BackoffType:  BackoffLinear,
		InitialDelay: 2 * time.Second,
		MaxDelay:     time.Minute,
	}
	if p.MaxAttempts != 5 {
		t.Errorf("expected 5, got %d", p.MaxAttempts)
	}
}

func TestCompensationRecord(t *testing.T) {
	r := &CompensationRecord{
		StepID:     "s1",
		Success:    true,
		ExecutedAt: time.Now(),
	}
	if !r.Success {
		t.Error("should be successful")
	}
}

func TestStepResult_Fields(t *testing.T) {
	now := time.Now()
	sr := &StepResult{
		StepID:      "s1",
		Success:     true,
		Output:      map[string]interface{}{"key": "val"},
		StartedAt:   now,
		CompletedAt: &now,
		Compensated: false,
	}
	if sr.StepID != "s1" {
		t.Error("StepID not set")
	}
	if sr.Compensated {
		t.Error("should not be compensated")
	}
}
