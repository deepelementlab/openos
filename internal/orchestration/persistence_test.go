package orchestration

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestInMemoryPersistence_SaveAndGetState(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	state := &StateMachineState{
		EntityID:     "agent-1",
		EntityType:   "agent",
		CurrentState: "created",
		StateData:    map[string]interface{}{"key": "value"},
	}

	if err := p.SaveState(ctx, state); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	retrieved, err := p.GetState(ctx, "agent-1")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if retrieved.CurrentState != "created" {
		t.Errorf("expected created, got %s", retrieved.CurrentState)
	}
	if retrieved.EntityID != "agent-1" {
		t.Errorf("expected agent-1, got %s", retrieved.EntityID)
	}
}

func TestInMemoryPersistence_GetStateNotFound(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	_, err := p.GetState(ctx, "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent state")
	}
}

func TestInMemoryPersistence_UpdateState(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	state := &StateMachineState{
		EntityID:     "agent-2",
		EntityType:   "agent",
		CurrentState: "created",
		StateData:    map[string]interface{}{},
	}
	p.SaveState(ctx, state)

	state.CurrentState = "scheduled"
	if err := p.SaveState(ctx, state); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	retrieved, _ := p.GetState(ctx, "agent-2")
	if retrieved.CurrentState != "scheduled" {
		t.Errorf("expected scheduled, got %s", retrieved.CurrentState)
	}
}

func TestInMemoryPersistence_SaveTransitionAndGetHistory(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	record := &TransitionHistoryRecord{
		StateMachineID: "sm-1",
		EntityID:       "agent-3",
		FromState:      "created",
		ToState:        "scheduled",
		Event:          "schedule",
		Success:        true,
		DurationMs:     50,
		StartedAt:      time.Now(),
		CompletedAt:    time.Now(),
	}

	if err := p.SaveTransition(ctx, record); err != nil {
		t.Fatalf("failed to save transition: %v", err)
	}

	records, err := p.GetTransitionHistory(ctx, "sm-1")
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].FromState != "created" {
		t.Errorf("expected from created, got %s", records[0].FromState)
	}
}

func TestInMemoryPersistence_GetTransitionHistoryEmpty(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	records, err := p.GetTransitionHistory(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty history, got %d records", len(records))
	}
}

func TestInMemoryPersistence_ListActive(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	p.SaveState(ctx, &StateMachineState{
		EntityID:     "active-1",
		EntityType:   "agent",
		CurrentState: "running",
	})
	p.SaveState(ctx, &StateMachineState{
		EntityID:     "active-2",
		EntityType:   "agent",
		CurrentState: "ready",
	})
	now := time.Now()
	p.SaveState(ctx, &StateMachineState{
		EntityID:     "completed-1",
		EntityType:   "agent",
		CurrentState: "stopped",
		CompletedAt:  &now,
	})

	active, err := p.ListActive(ctx, "agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active, got %d", len(active))
	}
}

func TestInMemoryPersistence_ListActiveByEntityType(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	p.SaveState(ctx, &StateMachineState{
		EntityID:     "agent-1",
		EntityType:   "agent",
		CurrentState: "running",
	})
	p.SaveState(ctx, &StateMachineState{
		EntityID:     "workflow-1",
		EntityType:   "workflow",
		CurrentState: "running",
	})

	agents, err := p.ListActive(ctx, "agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}

	workflows, err := p.ListActive(ctx, "workflow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(workflows))
	}
}

func TestInMemoryPersistence_DeleteState(t *testing.T) {
	p := NewInMemoryPersistence()
	ctx := context.Background()

	p.SaveState(ctx, &StateMachineState{
		EntityID:     "agent-del",
		EntityType:   "agent",
		CurrentState: "running",
	})

	if err := p.DeleteState(ctx, "agent-del"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := p.GetState(ctx, "agent-del")
	if err == nil {
		t.Error("should error after deletion")
	}
}

func TestStateMachineState_BeforeInsert(t *testing.T) {
	s := &StateMachineState{
		EntityID:     "test",
		EntityType:   "agent",
		CurrentState: "created",
		StateData:    map[string]interface{}{"key": "value"},
	}

	if err := s.BeforeInsert(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.ID == "" {
		t.Error("ID should be generated")
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if s.Version != 1 {
		t.Errorf("expected version 1, got %d", s.Version)
	}
	if len(s.StateDataJSON) == 0 {
		t.Error("StateDataJSON should be populated")
	}
}

func TestStateMachineState_BeforeInsertPreservesID(t *testing.T) {
	s := &StateMachineState{
		ID:           "custom-id",
		EntityID:     "test",
		EntityType:   "agent",
		CurrentState: "created",
		Version:      5,
	}

	s.BeforeInsert()
	if s.ID != "custom-id" {
		t.Errorf("expected custom-id, got %s", s.ID)
	}
	if s.Version != 5 {
		t.Errorf("expected version 5, got %d", s.Version)
	}
}

func TestStateMachineState_AfterScan(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"key": "value"})
	s := &StateMachineState{
		StateDataJSON: data,
	}

	if err := s.AfterScan(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.StateData["key"] != "value" {
		t.Error("StateData should be populated from JSON")
	}
}

func TestStateMachineState_IsComplete(t *testing.T) {
	s := &StateMachineState{}
	if s.IsComplete() {
		t.Error("should not be complete without CompletedAt")
	}

	now := time.Now()
	s.CompletedAt = &now
	if !s.IsComplete() {
		t.Error("should be complete with CompletedAt set")
	}
}

func TestTransitionHistoryRecord_Fields(t *testing.T) {
	now := time.Now()
	r := &TransitionHistoryRecord{
		ID:             "rec-1",
		StateMachineID: "sm-1",
		EntityID:       "agent-1",
		FromState:      "created",
		ToState:        "scheduled",
		Event:          "schedule",
		Success:        true,
		RetryCount:     0,
		DurationMs:     100,
		StartedAt:      now,
		CompletedAt:    now,
	}

	if r.FromState != "created" {
		t.Error("FromState not set correctly")
	}
	if !r.Success {
		t.Error("Success should be true")
	}
}

func TestDeepCopyMap(t *testing.T) {
	original := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	copy := deepCopyMap(original)
	if copy["key1"] != "value1" {
		t.Error("copy should have same values")
	}

	copy["key1"] = "modified"
	if original["key1"] == "modified" {
		t.Error("original should not be affected by copy modification")
	}

	nilCopy := deepCopyMap(nil)
	if nilCopy != nil {
		t.Error("nil map copy should be nil")
	}
}
