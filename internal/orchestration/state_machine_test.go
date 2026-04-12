package orchestration

import (
	"context"
	"testing"
	"time"
)

func TestNewStateMachineEngine(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	if engine == nil {
		t.Fatal("engine should not be nil")
	}
	defer engine.Stop()

	stats := engine.Stats()
	if stats["valid_states"] == nil {
		t.Error("stats should contain valid_states")
	}
}

func TestStateMachineEngine_StartStop(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{
		Persistence: NewInMemoryPersistence(),
	})

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}

	if err := engine.Stop(); err != nil {
		t.Fatalf("failed to stop engine: %v", err)
	}
}

func TestStateMachineEngine_StartWithoutPersistence(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start without persistence: %v", err)
	}
	defer engine.Stop()
}

func TestStateMachineEngine_CreateMachine(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	instance, err := engine.CreateMachine(ctx, "agent-1", "agent", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("failed to create machine: %v", err)
	}

	if instance.EntityID != "agent-1" {
		t.Errorf("expected entity_id agent-1, got %s", instance.EntityID)
	}
	if instance.CurrentState != "created" {
		t.Errorf("expected initial state created, got %s", instance.CurrentState)
	}
	if instance.Version != 1 {
		t.Errorf("expected version 1, got %d", instance.Version)
	}
	if instance.EntityType != "agent" {
		t.Errorf("expected entity_type agent, got %s", instance.EntityType)
	}
}

func TestStateMachineEngine_CreateMachineWithPersistence(t *testing.T) {
	persistence := NewInMemoryPersistence()
	engine := NewStateMachineEngine(StateMachineOptions{
		Persistence: persistence,
	})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	_, err := engine.CreateMachine(ctx, "agent-2", "agent", nil)
	if err != nil {
		t.Fatalf("failed to create machine: %v", err)
	}

	state, err := persistence.GetState(ctx, "agent-2")
	if err != nil {
		t.Fatalf("failed to get state from persistence: %v", err)
	}
	if state.CurrentState != "created" {
		t.Errorf("expected persisted state created, got %s", state.CurrentState)
	}
}

func TestStateMachineEngine_SendEvent(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-3", "agent", nil)

	result, err := engine.SendEvent(ctx, "agent-3", "schedule", nil)
	if err != nil {
		t.Fatalf("failed to send event: %v", err)
	}
	if !result.Success {
		t.Error("transition should succeed")
	}
	if result.From != "created" {
		t.Errorf("expected from created, got %s", result.From)
	}
	if result.To != "scheduled" {
		t.Errorf("expected to scheduled, got %s", result.To)
	}
}

func TestStateMachineEngine_SendEventTransitionSequence(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-4", "agent", nil)

	sequence := []struct {
		event      string
		expectedTo string
	}{
		{"schedule", "scheduled"},
		{"start", "starting"},
		{"ready", "ready"},
		{"stop", "stopping"},
		{"stopped", "stopped"},
	}

	for _, step := range sequence {
		result, err := engine.SendEvent(ctx, "agent-4", step.event, nil)
		if err != nil {
			t.Fatalf("failed on event %s: %v", step.event, err)
		}
		if result.To != step.expectedTo {
			t.Errorf("event %s: expected to %s, got %s", step.event, step.expectedTo, result.To)
		}
	}

	current, _ := engine.GetCurrentState("agent-4")
	if current != "stopped" {
		t.Errorf("expected final state stopped, got %s", current)
	}
}

func TestStateMachineEngine_SendEventDataMerge(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-data", "agent", map[string]interface{}{"initial": "val"})

	_, err := engine.SendEvent(ctx, "agent-data", "schedule", map[string]interface{}{"new_key": "new_val"})
	if err != nil {
		t.Fatalf("failed to send event: %v", err)
	}

	machine, _ := engine.GetMachine("agent-data")
	if machine.StateData["initial"] != "val" {
		t.Error("initial data should be preserved")
	}
	if machine.StateData["new_key"] != "new_val" {
		t.Error("new data should be merged")
	}
}

func TestStateMachineEngine_SendInvalidEvent(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-5", "agent", nil)

	_, err := engine.SendEvent(ctx, "agent-5", "ready", nil)
	if err == nil {
		t.Error("should not be able to transition from created to ready directly")
	}
}

func TestStateMachineEngine_SendEventNonexistentMachine(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	_, err := engine.SendEvent(ctx, "nonexistent", "schedule", nil)
	if err == nil {
		t.Error("should error for nonexistent machine")
	}
}

func TestStateMachineEngine_GetMachine(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-6", "agent", nil)

	instance, err := engine.GetMachine("agent-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.EntityID != "agent-6" {
		t.Errorf("expected agent-6, got %s", instance.EntityID)
	}

	_, err = engine.GetMachine("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent machine")
	}
}

func TestStateMachineEngine_CanTransition(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-7", "agent", nil)

	if !engine.CanTransition("agent-7", "schedule") {
		t.Error("should be able to schedule from created")
	}
	if engine.CanTransition("agent-7", "ready") {
		t.Error("should not be able to go ready from created")
	}
	if engine.CanTransition("nonexistent", "schedule") {
		t.Error("nonexistent machine should not be able to transition")
	}
}

func TestStateMachineEngine_GetCurrentState(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-8", "agent", nil)

	state, err := engine.GetCurrentState("agent-8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != "created" {
		t.Errorf("expected created, got %s", state)
	}

	_, err = engine.GetCurrentState("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent machine")
	}
}

func TestStateMachineEngine_GetStateSummary(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-9", "agent", nil)

	summary, err := engine.GetStateSummary("agent-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.CurrentState != "created" {
		t.Errorf("expected created, got %s", summary.CurrentState)
	}
	if summary.IsTerminal {
		t.Error("created should not be terminal")
	}
}

func TestStateMachineEngine_GetStateSummaryWithPersistence(t *testing.T) {
	persistence := NewInMemoryPersistence()
	engine := NewStateMachineEngine(StateMachineOptions{
		Persistence: persistence,
	})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-10", "agent", nil)
	engine.SendEvent(ctx, "agent-10", "schedule", nil)

	summary, err := engine.GetStateSummary("agent-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.CurrentState != "scheduled" {
		t.Errorf("expected scheduled, got %s", summary.CurrentState)
	}
	if summary.TotalTransitions < 1 {
		t.Error("should have at least 1 transition")
	}
}

func TestStateMachineEngine_ListActive(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.CreateMachine(ctx, "agent-a1", "agent", nil)
	engine.CreateMachine(ctx, "agent-a2", "agent", nil)

	active := engine.ListActive()
	if len(active) != 2 {
		t.Errorf("expected 2 active machines, got %d", len(active))
	}

	engine.SendEvent(ctx, "agent-a1", "schedule", nil)
	engine.SendEvent(ctx, "agent-a1", "start", nil)
	engine.SendEvent(ctx, "agent-a1", "fail", nil)

	time.Sleep(10 * time.Millisecond)
	active = engine.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active machine after failure, got %d", len(active))
	}
}

func TestStateMachineEngine_RegisterCustomTransition(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	engine.AddCustomState(&State{ID: "paused", Name: "Paused", Terminal: false})
	engine.AddCustomState(&State{ID: "paused-stopped", Name: "PausedStopped", Terminal: true})

	err := engine.RegisterCustomTransition(&Transition{
		ID:    "ready-paused",
		From:  "ready",
		To:    "paused",
		Event: "pause",
	})
	if err != nil {
		t.Fatalf("failed to register custom transition: %v", err)
	}
}

func TestStateMachineEngine_AddCustomState(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})

	engine.AddCustomState(&State{ID: "suspended", Name: "Suspended", Terminal: false})

	stats := engine.Stats()
	if stats["valid_states"].(int) != 11 {
		t.Errorf("expected 11 states after adding custom, got %d", stats["valid_states"])
	}
}

func TestStateMachineEngine_Stats(t *testing.T) {
	engine := NewStateMachineEngine(StateMachineOptions{})
	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	stats := engine.Stats()
	if stats["total_machines"].(int) != 0 {
		t.Error("should start with 0 machines")
	}

	engine.CreateMachine(ctx, "s1", "agent", nil)
	engine.CreateMachine(ctx, "s2", "agent", nil)

	stats = engine.Stats()
	if stats["total_machines"].(int) != 2 {
		t.Errorf("expected 2 machines, got %d", stats["total_machines"])
	}
	if stats["active_machines"].(int) != 2 {
		t.Errorf("expected 2 active machines, got %d", stats["active_machines"])
	}
	if stats["terminal_machines"].(int) != 0 {
		t.Errorf("expected 0 terminal machines, got %d", stats["terminal_machines"])
	}
}

func TestStateMachineEngine_LoadFromPersistence(t *testing.T) {
	persistence := NewInMemoryPersistence()
	ctx := context.Background()

	state := &StateMachineState{
		EntityID:     "agent-loaded",
		EntityType:   "agent",
		CurrentState: "ready",
		StateData:    map[string]interface{}{"loaded": true},
	}
	persistence.SaveState(ctx, state)

	engine := NewStateMachineEngine(StateMachineOptions{
		Persistence: persistence,
	})
	engine.Start(ctx)
	defer engine.Stop()

	machine, err := engine.GetMachine("agent-loaded")
	if err != nil {
		t.Fatalf("should load machine from persistence: %v", err)
	}
	if machine.CurrentState != "ready" {
		t.Errorf("expected ready, got %s", machine.CurrentState)
	}
}
