package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStepStatus_IsTerminal(t *testing.T) {
	terminal := []StepStatus{StepStatusCompleted, StepStatusFailed, StepStatusSkipped, StepStatusCompensated}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}

	nonTerminal := []StepStatus{StepStatusPending, StepStatusRunning}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestStepExecution_Duration(t *testing.T) {
	now := time.Now()
	se := &StepExecution{
		StartedAt:   now,
		CompletedAt: nil,
	}
	if se.Duration() < 0 {
		t.Error("duration should be non-negative for running step")
	}

	completed := now.Add(100 * time.Millisecond)
	se.CompletedAt = &completed
	if se.Duration() != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", se.Duration())
	}

	seZero := &StepExecution{}
	if seZero.Duration() != 0 {
		t.Errorf("zero step should have 0 duration, got %v", seZero.Duration())
	}
}

func TestStepHandlerFunc(t *testing.T) {
	handler := StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"echo": input["msg"]}, nil
	})

	if handler.GetName() != "StepHandlerFunc" {
		t.Errorf("expected StepHandlerFunc, got %s", handler.GetName())
	}

	result, err := handler.Execute(context.Background(), map[string]interface{}{"msg": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["echo"] != "hello" {
		t.Errorf("expected hello, got %v", result["echo"])
	}
}

func TestStepHandlerRegistry_Register(t *testing.T) {
	r := NewStepHandlerRegistry()

	handler := StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	if err := r.Register("handler1", handler); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
}

func TestStepHandlerRegistry_RegisterDuplicate(t *testing.T) {
	r := NewStepHandlerRegistry()
	handler := StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	})

	r.Register("dup", handler)
	if err := r.Register("dup", handler); err == nil {
		t.Error("should reject duplicate handler")
	}
}

func TestStepHandlerRegistry_RegisterFunc(t *testing.T) {
	r := NewStepHandlerRegistry()
	err := r.RegisterFunc("fn_handler", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})
	if err != nil {
		t.Fatalf("failed to register func: %v", err)
	}
}

func TestStepHandlerRegistry_Get(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("get_test", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"found": true}, nil
	}))

	h, err := r.Get("get_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.GetName() != "StepHandlerFunc" {
		t.Errorf("expected StepHandlerFunc, got %s", h.GetName())
	}

	_, err = r.Get("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent handler")
	}
}

func TestStepHandlerRegistry_Has(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("exists", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	}))

	if !r.Has("exists") {
		t.Error("should find existing handler")
	}
	if r.Has("nonexistent") {
		t.Error("should not find nonexistent handler")
	}
}

func TestStepExecutor_ExecuteStep(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("test_exec", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	}))

	executor := NewStepExecutor(r)
	step := &StepDefinition{
		ID:       "step1",
		Handler:  "test_exec",
		StepType: StepTypeTask,
	}

	exec, err := executor.ExecuteStep(context.Background(), step, map[string]interface{}{"input": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != StepStatusCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if exec.Output["result"] != "ok" {
		t.Errorf("expected ok, got %v", exec.Output["result"])
	}
}

func TestStepExecutor_ExecuteStepWithTimeout(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("slow_handler", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return map[string]interface{}{"done": true}, nil
		}
	}))

	executor := NewStepExecutor(r)
	step := &StepDefinition{
		ID:       "step-slow",
		Handler:  "slow_handler",
		StepType: StepTypeTask,
		Timeout:  10 * time.Millisecond,
	}

	_, err := executor.ExecuteStep(context.Background(), step, nil)
	if err == nil {
		t.Error("should fail due to context timeout")
	}
}

func TestStepExecutor_ExecuteStepHandlerNotFound(t *testing.T) {
	r := NewStepHandlerRegistry()
	executor := NewStepExecutor(r)
	step := &StepDefinition{ID: "s1", Handler: "nonexistent"}

	exec, err := executor.ExecuteStep(context.Background(), step, nil)
	if err == nil {
		t.Error("should error for missing handler")
	}
	if exec.Status != StepStatusFailed {
		t.Errorf("expected failed, got %s", exec.Status)
	}
}

func TestStepExecutor_ExecuteStepWithRetry(t *testing.T) {
	r := NewStepHandlerRegistry()
	attempts := 0
	r.Register("retry_handler", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("not yet")
		}
		return map[string]interface{}{"success": true}, nil
	}))

	executor := NewStepExecutor(r)
	step := &StepDefinition{
		ID:       "retry-step",
		Handler:  "retry_handler",
		StepType: StepTypeTask,
		RetryPolicy: &RetryPolicy{
			MaxAttempts:  5,
			BackoffType:  BackoffFixed,
			InitialDelay: time.Millisecond,
		},
	}

	exec, err := executor.ExecuteStepWithRetry(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != StepStatusCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestStepExecutor_ExecuteStepWithRetryExhausted(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("always_fail", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("always fails")
	}))

	executor := NewStepExecutor(r)
	step := &StepDefinition{
		ID:       "fail-step",
		Handler:  "always_fail",
		StepType: StepTypeTask,
		RetryPolicy: &RetryPolicy{
			MaxAttempts:  2,
			BackoffType:  BackoffFixed,
			InitialDelay: time.Millisecond,
		},
	}

	_, err := executor.ExecuteStepWithRetry(context.Background(), step, nil)
	if err == nil {
		t.Error("should fail after retries")
	}
}

func TestStepExecutor_ExecuteStepNoRetryPolicy(t *testing.T) {
	r := NewStepHandlerRegistry()
	r.Register("ok_handler", StepHandlerFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	}))

	executor := NewStepExecutor(r)
	step := &StepDefinition{
		ID:       "no-retry",
		Handler:  "ok_handler",
		StepType: StepTypeTask,
	}

	exec, err := executor.ExecuteStepWithRetry(context.Background(), step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != StepStatusCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		policy   *RetryPolicy
		attempt  int
		expected time.Duration
	}{
		{
			"fixed",
			&RetryPolicy{BackoffType: BackoffFixed, InitialDelay: 100 * time.Millisecond},
			1,
			100 * time.Millisecond,
		},
		{
			"linear",
			&RetryPolicy{BackoffType: BackoffLinear, InitialDelay: 100 * time.Millisecond},
			3,
			300 * time.Millisecond,
		},
		{
			"exponential",
			&RetryPolicy{BackoffType: BackoffExponential, InitialDelay: 100 * time.Millisecond},
			2,
			200 * time.Millisecond,
		},
		{
			"max delay cap",
			&RetryPolicy{BackoffType: BackoffExponential, InitialDelay: 100 * time.Millisecond, MaxDelay: 150 * time.Millisecond},
			3,
			150 * time.Millisecond,
		},
		{
			"nil policy",
			nil,
			1,
			0,
		},
		{
			"zero attempt",
			&RetryPolicy{BackoffType: BackoffFixed, InitialDelay: 100 * time.Millisecond},
			0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBackoff(tt.policy, tt.attempt)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestCalculateBackoff_DefaultBase(t *testing.T) {
	got := calculateBackoff(&RetryPolicy{BackoffType: BackoffFixed}, 1)
	if got != time.Second {
		t.Errorf("expected default 1s base, got %v", got)
	}
}

func TestGenerateStepExecutionID(t *testing.T) {
	id1 := generateStepExecutionID()
	if id1 == "" {
		t.Error("ID should not be empty")
	}
	if len(id1) == 0 {
		t.Error("ID should have content")
	}
}

func TestStepResult(t *testing.T) {
	sr := &StepResult{
		Success:  true,
		Output:   map[string]interface{}{"key": "val"},
		NextStep: "next",
	}
	if !sr.Success {
		t.Error("should be successful")
	}
	if sr.NextStep != "next" {
		t.Error("NextStep not set")
	}
}
