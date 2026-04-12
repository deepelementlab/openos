package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewRecoveryManager(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	store := NewInMemorySagaStore()
	rm := NewRecoveryManager(c, store, zap.NewNop())
	if rm == nil {
		t.Fatal("recovery manager should not be nil")
	}
}

func TestNewRecoveryManager_NilStore(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	rm := NewRecoveryManager(c, nil, zap.NewNop())
	if rm == nil {
		t.Fatal("recovery manager should not be nil with nil store")
	}
}

func TestDefaultRecoveryOptions(t *testing.T) {
	opts := DefaultRecoveryOptions()
	if opts.MaxRetries != 3 {
		t.Errorf("expected 3 retries, got %d", opts.MaxRetries)
	}
	if opts.RetryTimeout != 5*time.Minute {
		t.Errorf("expected 5m timeout, got %v", opts.RetryTimeout)
	}
	if !opts.AutoCompensate {
		t.Error("AutoCompensate should be true by default")
	}
}

func TestRecoveryManager_CanRecover(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	store := NewInMemorySagaStore()
	rm := NewRecoveryManager(c, store, zap.NewNop())

	recoverable := []SagaStatus{
		SagaStatusFailed,
		SagaStatusPartiallyCompensated,
		SagaStatusRunning,
		SagaStatusCompensating,
	}
	for _, status := range recoverable {
		inst := &SagaInstance{Status: status}
		if !rm.CanRecover(inst) {
			t.Errorf("should be able to recover %s", status)
		}
	}

	notRecoverable := []SagaStatus{
		SagaStatusCompleted,
		SagaStatusCompensated,
	}
	for _, status := range notRecoverable {
		inst := &SagaInstance{Status: status}
		if rm.CanRecover(inst) {
			t.Errorf("should not be able to recover %s", status)
		}
	}
}

func TestRecoveryManager_Recover_CompletedSaga(t *testing.T) {
	store := NewInMemorySagaStore()
	c := NewSagaCoordinator(CoordinatorOptions{Store: store})

	saga := &Saga{
		ID:   "completed-saga",
		Name: "Completed",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	}
	c.RegisterSaga(saga)

	inst := NewSagaInstance("completed-saga", nil)
	inst.MarkCompleted()
	store.Save(context.Background(), inst)

	rm := NewRecoveryManager(c, store, zap.NewNop())
	result, err := rm.Recover(context.Background(), inst.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("should not be able to recover completed saga")
	}
}

func TestRecoveryManager_Recover_FailedSaga(t *testing.T) {
	store := NewInMemorySagaStore()
	c := NewSagaCoordinator(CoordinatorOptions{Store: store})

	saga := &Saga{
		ID:   "failed-saga",
		Name: "Failed",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "ok"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("s2 failed")
			})},
		},
	}
	c.RegisterSaga(saga)

	inst := NewSagaInstance("failed-saga", nil)
	inst.AddStepResult(&StepResult{StepID: "s1", Success: true, Output: map[string]interface{}{"s1": "ok"}})
	inst.AddStepResult(&StepResult{StepID: "s2", Success: false, Error: "s2 failed"})
	inst.MarkFailed(errors.New("s2 failed"))
	store.Save(context.Background(), inst)

	rm := NewRecoveryManager(c, store, zap.NewNop())
	result, err := rm.Recover(context.Background(), inst.ID, &RecoveryOptions{
		MaxRetries:     1,
		AutoCompensate: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OriginalStatus != SagaStatusFailed {
		t.Errorf("expected original status failed, got %s", result.OriginalStatus)
	}
}

func TestRecoveryManager_Recover_NilOptions(t *testing.T) {
	store := NewInMemorySagaStore()
	c := NewSagaCoordinator(CoordinatorOptions{Store: store})

	c.RegisterSaga(&Saga{
		ID:   "nil-opts-saga",
		Name: "Test",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	})

	inst := NewSagaInstance("nil-opts-saga", nil)
	inst.MarkCompleted()
	store.Save(context.Background(), inst)

	rm := NewRecoveryManager(c, store, zap.NewNop())
	result, err := rm.Recover(context.Background(), inst.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestRecoveryManager_Recover_NonexistentInstance(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	store := NewInMemorySagaStore()
	rm := NewRecoveryManager(c, store, zap.NewNop())

	_, err := rm.Recover(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("should error for nonexistent instance")
	}
}

func TestRecoveryResult_Fields(t *testing.T) {
	r := &RecoveryResult{
		InstanceID:     "inst-1",
		Success:        true,
		OriginalStatus: SagaStatusFailed,
		FinalStatus:    SagaStatusCompleted,
		Steps:          []RecoveryStep{},
		StartedAt:      time.Now(),
		CompletedAt:    time.Now(),
		Duration:       5 * time.Second,
	}
	if r.InstanceID != "inst-1" {
		t.Error("InstanceID not set")
	}
	if r.OriginalStatus != SagaStatusFailed {
		t.Error("OriginalStatus not set")
	}
	if r.Duration != 5*time.Second {
		t.Error("Duration not set")
	}
}

func TestRecoveryStep_Fields(t *testing.T) {
	s := &RecoveryStep{
		Action:      "retry_step",
		StepID:      "s1",
		Success:     true,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}
	if s.Action != "retry_step" {
		t.Error("Action not set")
	}
	if s.StepID != "s1" {
		t.Error("StepID not set")
	}
}

func TestRecoveryOptions_Fields(t *testing.T) {
	opts := &RecoveryOptions{
		MaxRetries:     5,
		RetryTimeout:   10 * time.Minute,
		AutoCompensate: false,
	}
	if opts.MaxRetries != 5 {
		t.Errorf("expected 5, got %d", opts.MaxRetries)
	}
	if opts.AutoCompensate {
		t.Error("AutoCompensate should be false")
	}
}
