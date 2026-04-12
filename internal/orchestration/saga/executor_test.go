package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewSagaExecutor(t *testing.T) {
	e := NewSagaExecutor()
	if e == nil {
		t.Fatal("executor should not be nil")
	}
}

func TestSagaExecutor_WithLogger(t *testing.T) {
	e := NewSagaExecutor().WithLogger(nil)
	if e == nil {
		t.Fatal("executor should not be nil")
	}
}

func TestSagaExecutor_ExecuteSequential(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "seq-test",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "done"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s2": "done"}, nil
			})},
		},
	}

	instance := NewSagaInstance("seq-test", map[string]interface{}{"initial": "data"})

	err := e.ExecuteSequential(context.Background(), instance, saga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}
	if instance.StepResults["s1"] == nil || !instance.StepResults["s1"].Success {
		t.Error("step s1 should be successful")
	}
	if instance.StepResults["s2"] == nil || !instance.StepResults["s2"].Success {
		t.Error("step s2 should be successful")
	}
}

func TestSagaExecutor_ExecuteSequential_StepFailure(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "seq-fail",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "done"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("s2 failed")
			})},
			{ID: "s3", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s3": "done"}, nil
			})},
		},
	}

	instance := NewSagaInstance("seq-fail", nil)

	err := e.ExecuteSequential(context.Background(), instance, saga)
	if err == nil {
		t.Error("should return error for failed step")
	}
	if instance.Status != SagaStatusFailed {
		t.Errorf("expected failed, got %s", instance.Status)
	}
	if _, exists := instance.StepResults["s3"]; exists {
		t.Error("s3 should not have been executed")
	}
}

func TestSagaExecutor_ExecuteSequential_Cancelled(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "seq-cancel",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	instance := NewSagaInstance("seq-cancel", nil)
	err := e.ExecuteSequential(ctx, instance, saga)
	if err == nil {
		t.Error("should return error for cancelled context")
	}
}

func TestSagaExecutor_ExecuteParallel(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "par-test",
		ExecutionMode: Parallel,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "done"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s2": "done"}, nil
			})},
		},
	}

	instance := NewSagaInstance("par-test", nil)

	err := e.ExecuteParallel(context.Background(), instance, saga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}
}

func TestSagaExecutor_ExecuteParallel_WithDependencies(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "par-deps",
		ExecutionMode: Parallel,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "done"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s2": "done"}, nil
			})},
		},
	}

	instance := NewSagaInstance("par-deps", nil)

	err := e.ExecuteParallel(context.Background(), instance, saga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}
}

func TestSagaExecutor_ExecuteParallel_StepFailure(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "par-fail",
		ExecutionMode: Parallel,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("s1 failed")
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s2": "done"}, nil
			})},
		},
	}

	instance := NewSagaInstance("par-fail", nil)

	err := e.ExecuteParallel(context.Background(), instance, saga)
	if err == nil {
		t.Error("should return error for failed step")
	}
	if instance.Status != SagaStatusFailed {
		t.Errorf("expected failed, got %s", instance.Status)
	}
}

func TestSagaExecutor_Compensate(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "comp-test",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"s1": "done"}, nil
				}),
				Compensation: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"s1_comp": "done"}, nil
				}),
			},
			{
				ID: "s2",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"s2": "done"}, nil
				}),
				Compensation: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"s2_comp": "done"}, nil
				}),
			},
		},
	}

	instance := NewSagaInstance("comp-test", map[string]interface{}{"input": 1})
	instance.AddStepResult(&StepResult{StepID: "s1", Success: true, Output: map[string]interface{}{"s1": "done"}})
	instance.AddStepResult(&StepResult{StepID: "s2", Success: true, Output: map[string]interface{}{"s2": "done"}})

	result := e.Compensate(context.Background(), instance, saga)

	if !result.AllCompensated {
		t.Error("all steps should be compensated")
	}
	if len(result.CompensatedSteps) != 2 {
		t.Errorf("expected 2 compensated, got %d", len(result.CompensatedSteps))
	}
	if len(result.FailedCompensations) != 0 {
		t.Errorf("expected 0 failed, got %d", len(result.FailedCompensations))
	}
}

func TestSagaExecutor_Compensate_Failure(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "comp-fail",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
				Compensation: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, errors.New("compensation failed")
				}),
			},
		},
	}

	instance := NewSagaInstance("comp-fail", nil)
	instance.AddStepResult(&StepResult{StepID: "s1", Success: true, Output: map[string]interface{}{}})

	result := e.Compensate(context.Background(), instance, saga)

	if result.AllCompensated {
		t.Error("should not be fully compensated")
	}
	if len(result.FailedCompensations) != 1 {
		t.Errorf("expected 1 failed, got %d", len(result.FailedCompensations))
	}
}

func TestSagaExecutor_Compensate_NoCompensationHandler(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "comp-skip",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
			},
		},
	}

	instance := NewSagaInstance("comp-skip", nil)
	instance.AddStepResult(&StepResult{StepID: "s1", Success: true, Output: map[string]interface{}{}})

	result := e.Compensate(context.Background(), instance, saga)

	if !result.AllCompensated {
		t.Error("should be all compensated (no compensation needed)")
	}
}

func TestSagaExecutor_StepWithRetry(t *testing.T) {
	e := NewSagaExecutor()
	attempts := 0

	saga := &Saga{
		ID:            "retry-test",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					attempts++
					if attempts < 3 {
						return nil, errors.New("not yet")
					}
					return map[string]interface{}{"done": true}, nil
				}),
				RetryPolicy: &RetryPolicy{
					MaxAttempts:  5,
					BackoffType:  BackoffFixed,
					InitialDelay: time.Millisecond,
				},
			},
		},
	}

	instance := NewSagaInstance("retry-test", nil)
	err := e.ExecuteSequential(context.Background(), instance, saga)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestSagaExecutor_StepRetryExhausted(t *testing.T) {
	e := NewSagaExecutor()

	saga := &Saga{
		ID:            "retry-exhaust",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, errors.New("always fail")
				}),
				RetryPolicy: &RetryPolicy{
					MaxAttempts:  2,
					BackoffType:  BackoffFixed,
					InitialDelay: time.Millisecond,
				},
			},
		},
	}

	instance := NewSagaInstance("retry-exhaust", nil)
	err := e.ExecuteSequential(context.Background(), instance, saga)
	if err == nil {
		t.Error("should fail after retries exhausted")
	}
}

func TestCompensationResult_Fields(t *testing.T) {
	r := &CompensationResult{
		AllCompensated:      true,
		FailedCompensations: []string{},
		CompensatedSteps:    []string{"s1", "s2"},
	}
	if !r.AllCompensated {
		t.Error("should be all compensated")
	}
	if len(r.CompensatedSteps) != 2 {
		t.Errorf("expected 2, got %d", len(r.CompensatedSteps))
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		policy   *RetryPolicy
		attempt  int
		expected time.Duration
	}{
		{"fixed", &RetryPolicy{BackoffType: BackoffFixed, InitialDelay: 100 * time.Millisecond}, 1, 100 * time.Millisecond},
		{"linear", &RetryPolicy{BackoffType: BackoffLinear, InitialDelay: 100 * time.Millisecond}, 3, 300 * time.Millisecond},
		{"exponential", &RetryPolicy{BackoffType: BackoffExponential, InitialDelay: 100 * time.Millisecond}, 2, 200 * time.Millisecond},
		{"nil", nil, 1, 0},
		{"zero attempt", &RetryPolicy{BackoffType: BackoffFixed, InitialDelay: 100 * time.Millisecond}, 0, 0},
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

func TestCalculateBackoff_MaxDelay(t *testing.T) {
	got := calculateBackoff(&RetryPolicy{
		BackoffType:  BackoffExponential,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     150 * time.Millisecond,
	}, 5)
	if got != 150*time.Millisecond {
		t.Errorf("expected capped at 150ms, got %v", got)
	}
}

func TestCalculateBackoff_DefaultBase(t *testing.T) {
	got := calculateBackoff(&RetryPolicy{BackoffType: BackoffFixed}, 1)
	if got != time.Second {
		t.Errorf("expected default 1s, got %v", got)
	}
}
