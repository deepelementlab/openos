package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewSagaCoordinator(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	if c == nil {
		t.Fatal("coordinator should not be nil")
	}
}

func TestNewSagaCoordinator_WithLogger(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{
		Logger: zap.NewNop(),
	})
	if c == nil {
		t.Fatal("coordinator should not be nil")
	}
}

func TestSagaCoordinator_RegisterSaga(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})

	saga := &Saga{
		ID:   "test-saga",
		Name: "Test",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
		ExecutionMode: Sequential,
	}

	if err := c.RegisterSaga(saga); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
}

func TestSagaCoordinator_RegisterSagaNoID(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})

	err := c.RegisterSaga(&Saga{Name: "No ID"})
	if err == nil {
		t.Error("should reject saga without ID")
	}
}

func TestSagaCoordinator_GetSaga(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:   "get-test",
		Name: "Test",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	})

	saga, err := c.GetSaga("get-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saga.ID != "get-test" {
		t.Errorf("expected get-test, got %s", saga.ID)
	}

	_, err = c.GetSaga("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent saga")
	}
}

func TestSagaCoordinator_StartSaga(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:            "start-test",
		Name:          "Test",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"result": "ok"}, nil
			})},
		},
	})

	instance, err := c.StartSaga(context.Background(), "start-test", map[string]interface{}{"input": 1})
	if err != nil {
		t.Fatalf("failed to start saga: %v", err)
	}
	if instance.SagaID != "start-test" {
		t.Errorf("expected start-test, got %s", instance.SagaID)
	}

	time.Sleep(100 * time.Millisecond)

	if instance.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}
}

func TestSagaCoordinator_StartSagaNotFound(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})

	_, err := c.StartSaga(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("should error for nonexistent saga")
	}
}

func TestSagaCoordinator_StartSagaWithFailure(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:            "fail-saga",
		Name:          "Fail",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("step failed")
			})},
		},
	})

	instance, _ := c.StartSaga(context.Background(), "fail-saga", nil)

	time.Sleep(100 * time.Millisecond)

	if instance.Status != SagaStatusCompensated {
		t.Errorf("expected compensated after failure (no compensation handlers), got %s", instance.Status)
	}
}

func TestSagaCoordinator_StartSagaWithCompensation(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	compensated := false

	c.RegisterSaga(&Saga{
		ID:            "comp-saga",
		Name:          "Compensate",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{
				ID: "s1",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"s1_done": true}, nil
				}),
				Compensation: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					compensated = true
					return nil, nil
				}),
			},
			{
				ID: "s2",
				Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, errors.New("s2 failed")
				}),
			},
		},
	})

	c.StartSaga(context.Background(), "comp-saga", nil)

	time.Sleep(100 * time.Millisecond)

	if !compensated {
		t.Error("compensation should have been called for s1")
	}
}

func TestSagaCoordinator_StartSagaWithPersistence(t *testing.T) {
	store := NewInMemorySagaStore()
	c := NewSagaCoordinator(CoordinatorOptions{
		Store: store,
	})

	c.RegisterSaga(&Saga{
		ID:            "persist-saga",
		Name:          "Persist",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	})

	instance, _ := c.StartSaga(context.Background(), "persist-saga", nil)

	time.Sleep(100 * time.Millisecond)

	saved, err := store.Get(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("should be persisted: %v", err)
	}
	if saved.SagaID != "persist-saga" {
		t.Errorf("expected persist-saga, got %s", saved.SagaID)
	}
}

func TestSagaCoordinator_GetInstance(t *testing.T) {
	store := NewInMemorySagaStore()
	c := NewSagaCoordinator(CoordinatorOptions{
		Store: store,
	})

	instance := NewSagaInstance("saga-1", nil)
	store.Save(context.Background(), instance)

	retrieved, err := c.GetInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.ID != instance.ID {
		t.Error("IDs should match")
	}
}

func TestSagaCoordinator_GetInstanceNotFound(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})

	_, err := c.GetInstance(context.Background(), "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent instance")
	}
}

func TestSagaCoordinator_ListActiveInstances(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:            "active-test",
		Name:          "Active",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				time.Sleep(5 * time.Second)
				return nil, nil
			})},
		},
	})

	c.StartSaga(context.Background(), "active-test", nil)
	time.Sleep(20 * time.Millisecond)

	active := c.ListActiveInstances()
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
}

func TestSagaCoordinator_Stats(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:   "stats-saga",
		Name: "Stats",
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			})},
		},
	})

	stats := c.Stats()
	if stats["registered_sagas"].(int) != 1 {
		t.Errorf("expected 1 registered saga, got %d", stats["registered_sagas"])
	}
}

func TestInMemorySagaStore_CRUD(t *testing.T) {
	store := NewInMemorySagaStore()
	ctx := context.Background()

	inst := NewSagaInstance("saga-1", nil)
	if err := store.Save(ctx, inst); err != nil {
		t.Fatalf("failed to save: %v", err)
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

func TestInMemorySagaStore_List(t *testing.T) {
	store := NewInMemorySagaStore()
	ctx := context.Background()

	inst1 := NewSagaInstance("saga-1", nil)
	inst1.Status = SagaStatusCompleted
	inst2 := NewSagaInstance("saga-1", nil)
	inst2.Status = SagaStatusRunning

	store.Save(ctx, inst1)
	store.Save(ctx, inst2)

	all, err := store.List(ctx, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}

	completed, _ := store.List(ctx, "saga-1", SagaStatusCompleted)
	if len(completed) != 1 {
		t.Errorf("expected 1 completed, got %d", len(completed))
	}
}

func TestInMemorySagaStore_Delete(t *testing.T) {
	store := NewInMemorySagaStore()
	ctx := context.Background()

	inst := NewSagaInstance("saga-1", nil)
	store.Save(ctx, inst)

	if err := store.Delete(ctx, inst.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := store.Get(ctx, inst.ID)
	if err == nil {
		t.Error("should error after delete")
	}
}

func TestSagaCoordinator_StartSagaMultiStep(t *testing.T) {
	c := NewSagaCoordinator(CoordinatorOptions{})
	c.RegisterSaga(&Saga{
		ID:            "multi-step",
		Name:          "Multi",
		ExecutionMode: Sequential,
		Steps: []*SagaStep{
			{ID: "s1", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s1": "done"}, nil
			})},
			{ID: "s2", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s2": "done"}, nil
			})},
			{ID: "s3", Action: SagaActionFunc(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"s3": "done"}, nil
			})},
		},
	})

	instance, _ := c.StartSaga(context.Background(), "multi-step", nil)
	time.Sleep(150 * time.Millisecond)

	if instance.Status != SagaStatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}
	if len(instance.StepResults) != 3 {
		t.Errorf("expected 3 step results, got %d", len(instance.StepResults))
	}
}
