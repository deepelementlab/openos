package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTransitionRules_StandardTransitions(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)
	all := rules.GetAllTransitions()

	if len(all) < 10 {
		t.Errorf("expected at least 10 standard transitions, got %d", len(all))
	}
}

func TestTransitionRules_FindTransition(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	tr, err := rules.FindTransition("created", "schedule")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.To != "scheduled" {
		t.Errorf("expected to=scheduled, got %s", tr.To)
	}

	_, err = rules.FindTransition("created", "ready")
	if err == nil {
		t.Error("should not find direct created->ready transition")
	}
}

func TestTransitionRules_IsValid(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	if !rules.IsValid("created", "scheduled", "schedule") {
		t.Error("created->scheduled via schedule should be valid")
	}
	if rules.IsValid("created", "ready", "schedule") {
		t.Error("created->ready via schedule should not be valid")
	}
	if rules.IsValid("created", "scheduled", "start") {
		t.Error("created->scheduled via start should not be valid")
	}
}

func TestTransitionRules_GetTransitions(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	transitions := rules.GetTransitions("created")
	if len(transitions) == 0 {
		t.Error("created should have transitions")
	}

	transitions = rules.GetTransitions("ready")
	if len(transitions) < 2 {
		t.Errorf("ready should have at least 2 transitions (stop, fail), got %d", len(transitions))
	}
}

func TestTransitionRules_WildcardTransitions(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	transitions := rules.GetTransitions("created")
	hasWildcard := false
	for _, tr := range transitions {
		if tr.From == "*" {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		t.Error("should include wildcard transitions when querying a specific state")
	}
}

func TestTransitionRules_Register(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	err := rules.Register(&Transition{
		ID:    "test-trans",
		From:  "created",
		To:    "ready",
		Event: "skip",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	tr, err := rules.FindTransition("created", "skip")
	if err != nil {
		t.Fatalf("should find registered transition: %v", err)
	}
	if tr.To != "ready" {
		t.Errorf("expected to=ready, got %s", tr.To)
	}
}

func TestTransitionRules_RegisterDuplicate(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	err := rules.Register(&Transition{
		ID:    "dup-trans",
		From:  "created",
		To:    "scheduled",
		Event: "schedule",
	})
	if err == nil {
		t.Error("should not allow duplicate transition")
	}
}

func TestTransitionRules_RegisterInvalidStates(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	err := rules.Register(&Transition{
		ID:    "invalid-from",
		From:  "invalid_state",
		To:    "created",
		Event: "test",
	})
	if err == nil {
		t.Error("should reject invalid from state")
	}

	err = rules.Register(&Transition{
		ID:    "invalid-to",
		From:  "created",
		To:    "invalid_state",
		Event: "test",
	})
	if err == nil {
		t.Error("should reject invalid to state")
	}
}

func TestTransitionExecutor_Execute(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)
	executor := NewTransitionExecutor(rules)

	result, err := executor.Execute(context.Background(), "created", "schedule", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("transition should succeed")
	}
	if result.From != "created" {
		t.Errorf("expected from=created, got %s", result.From)
	}
	if result.To != "scheduled" {
		t.Errorf("expected to=scheduled, got %s", result.To)
	}
}

func TestTransitionExecutor_ExecuteInvalidEvent(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)
	executor := NewTransitionExecutor(rules)

	_, err := executor.Execute(context.Background(), "created", "invalid_event", nil)
	if err == nil {
		t.Error("should fail for invalid event")
	}
}

func TestTransitionExecutor_CanTransition(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)
	executor := NewTransitionExecutor(rules)

	if !executor.CanTransition("created", "schedule") {
		t.Error("should be able to transition created->scheduled")
	}
	if executor.CanTransition("created", "ready") {
		t.Error("should not be able to transition created->ready directly")
	}
}

func TestTransitionExecutor_GuardPasses(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	rules.Register(&Transition{
		ID:    "guarded-pass",
		From:  "created",
		To:    "scheduled",
		Event: "guarded_ok",
		Guard: func(ctx context.Context, from, to string, data interface{}) error {
			return nil
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "guarded_ok", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("guard should pass")
	}
}

func TestTransitionExecutor_GuardFails(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	rules.Register(&Transition{
		ID:    "guarded-fail",
		From:  "created",
		To:    "scheduled",
		Event: "guarded_fail",
		Guard: func(ctx context.Context, from, to string, data interface{}) error {
			return errors.New("guard condition failed")
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "guarded_fail", nil)
	if err != nil {
		t.Fatalf("should not return error, result should contain failure: %v", err)
	}
	if result.Success {
		t.Error("transition should fail due to guard")
	}
	if result.Error == "" {
		t.Error("result should contain guard error message")
	}
}

func TestTransitionExecutor_Actions(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	actionCalled := false
	rules.Register(&Transition{
		ID:    "with-action",
		From:  "created",
		To:    "scheduled",
		Event: "action_test",
		Actions: []TransitionAction{
			func(ctx context.Context, from, to string, data interface{}) error {
				actionCalled = true
				return nil
			},
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "action_test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("transition should succeed")
	}
	if !actionCalled {
		t.Error("action should have been called")
	}
}

func TestTransitionExecutor_ActionFailure(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	rules.Register(&Transition{
		ID:    "action-fail",
		From:  "created",
		To:    "scheduled",
		Event: "action_fail_test",
		Actions: []TransitionAction{
			func(ctx context.Context, from, to string, data interface{}) error {
				return errors.New("action error")
			},
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "action_fail_test", nil)
	if err != nil {
		t.Fatalf("should not return error directly: %v", err)
	}
	if result.Success {
		t.Error("transition should fail due to action error")
	}
}

func TestTransitionExecutor_RetryPolicy(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	attempts := 0
	rules.Register(&Transition{
		ID:    "retry-trans",
		From:  "created",
		To:    "scheduled",
		Event: "retry_test",
		Actions: []TransitionAction{
			func(ctx context.Context, from, to string, data interface{}) error {
				attempts++
				if attempts < 3 {
					return errors.New("not yet")
				}
				return nil
			},
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:  3,
			BackoffBase: time.Millisecond,
			BackoffMax:  time.Millisecond * 10,
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "retry_test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("should succeed after retries")
	}
	if result.RetryCount < 2 {
		t.Errorf("expected at least 2 retries, got %d", result.RetryCount)
	}
}

func TestTransitionExecutor_RetryExhausted(t *testing.T) {
	v := NewStateValidator()
	rules := NewTransitionRules(v)

	rules.Register(&Transition{
		ID:    "retry-exhaust",
		From:  "created",
		To:    "scheduled",
		Event: "retry_exhaust",
		Actions: []TransitionAction{
			func(ctx context.Context, from, to string, data interface{}) error {
				return errors.New("always fail")
			},
		},
		RetryPolicy: &RetryPolicy{
			MaxRetries:  2,
			BackoffBase: time.Millisecond,
			BackoffMax:  time.Millisecond * 10,
		},
	})

	executor := NewTransitionExecutor(rules)
	result, err := executor.Execute(context.Background(), "created", "retry_exhaust", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("should fail after retries exhausted")
	}
}

func TestTransitionResult_Duration(t *testing.T) {
	now := time.Now()
	r := &TransitionResult{
		StartedAt:   now,
		CompletedAt: now.Add(100 * time.Millisecond),
	}
	if r.Duration() != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", r.Duration())
	}

	r2 := &TransitionResult{
		StartedAt: time.Now(),
	}
	if r2.Duration() < 0 {
		t.Error("duration should be non-negative for incomplete result")
	}
}

func TestTransition_Fields(t *testing.T) {
	tr := &Transition{
		ID:    "test",
		From:  "a",
		To:    "b",
		Event: "go",
		Metadata: map[string]interface{}{
			"priority": "high",
		},
	}
	if tr.ID != "test" || tr.From != "a" || tr.To != "b" || tr.Event != "go" {
		t.Error("transition fields not set correctly")
	}
	if tr.Metadata["priority"] != "high" {
		t.Error("metadata not set correctly")
	}
}
