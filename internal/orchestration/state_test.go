package orchestration

import (
	"testing"
	"time"
)

func TestStateTypeConstants(t *testing.T) {
	types := map[StateType]string{
		StateCreated:    "created",
		StateScheduled:  "scheduled",
		StateStarting:   "starting",
		StateReady:      "ready",
		StateStopping:   "stopping",
		StateStopped:    "stopped",
		StateFailed:     "failed",
		StateRecovering: "recovering",
		StateDeleting:   "deleting",
		StateDeleted:    "deleted",
		StatePending:    "pending",
		StateRunning:    "running",
		StateCompleted:  "completed",
		StateCancelled:  "cancelled",
	}
	for st, expected := range types {
		if string(st) != expected {
			t.Errorf("expected %s, got %s", expected, string(st))
		}
	}
}

func TestStandardStates(t *testing.T) {
	states := StandardStates()
	if len(states) != 10 {
		t.Fatalf("expected 10 standard states, got %d", len(states))
	}

	seen := make(map[string]bool)
	for _, s := range states {
		if s.ID == "" {
			t.Error("state ID should not be empty")
		}
		if s.Name == "" {
			t.Errorf("state %s should have a name", s.ID)
		}
		if seen[s.ID] {
			t.Errorf("duplicate state ID: %s", s.ID)
		}
		seen[s.ID] = true
	}

	terminalStates := []string{"stopped", "failed", "deleted"}
	for _, id := range terminalStates {
		for _, s := range states {
			if s.ID == id && !s.Terminal {
				t.Errorf("state %s should be terminal", id)
			}
		}
	}

	nonTerminal := []string{"created", "scheduled", "starting", "ready", "stopping", "recovering", "deleting"}
	for _, id := range nonTerminal {
		for _, s := range states {
			if s.ID == id && s.Terminal {
				t.Errorf("state %s should not be terminal", id)
			}
		}
	}
}

func TestStateInstance_IsTerminal(t *testing.T) {
	states := make(map[string]*State)
	for _, s := range StandardStates() {
		states[s.ID] = s
	}

	si := &StateInstance{StateID: "stopped"}
	if !si.IsTerminal(states) {
		t.Error("stopped should be terminal")
	}

	si = &StateInstance{StateID: "ready"}
	if si.IsTerminal(states) {
		t.Error("ready should not be terminal")
	}

	si = &StateInstance{StateID: "nonexistent"}
	if si.IsTerminal(states) {
		t.Error("nonexistent state should not be terminal")
	}
}

func TestStateInstance_Duration(t *testing.T) {
	now := time.Now()
	si := &StateInstance{EnteredAt: now}

	d := si.Duration()
	if d < 0 {
		t.Error("duration should be non-negative")
	}

	exitTime := now.Add(5 * time.Second)
	si.ExitedAt = &exitTime
	d = si.Duration()
	if d != 5*time.Second {
		t.Errorf("expected 5s duration, got %v", d)
	}
}

func TestStateValidator_ValidateState(t *testing.T) {
	v := NewStateValidator()

	if err := v.ValidateState("created"); err != nil {
		t.Errorf("created should be valid: %v", err)
	}
	if err := v.ValidateState("ready"); err != nil {
		t.Errorf("ready should be valid: %v", err)
	}
	if err := v.ValidateState("nonexistent"); err == nil {
		t.Error("nonexistent should be invalid")
	}
}

func TestStateValidator_IsTerminal(t *testing.T) {
	v := NewStateValidator()

	if !v.IsTerminal("stopped") {
		t.Error("stopped should be terminal")
	}
	if !v.IsTerminal("failed") {
		t.Error("failed should be terminal")
	}
	if !v.IsTerminal("deleted") {
		t.Error("deleted should be terminal")
	}
	if v.IsTerminal("created") {
		t.Error("created should not be terminal")
	}
	if v.IsTerminal("nonexistent") {
		t.Error("nonexistent should not be terminal")
	}
}

func TestStateValidator_GetState(t *testing.T) {
	v := NewStateValidator()

	s, err := v.GetState("created")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != "created" {
		t.Errorf("expected ID created, got %s", s.ID)
	}

	_, err = v.GetState("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent state")
	}
}

func TestStateValidator_AddState(t *testing.T) {
	v := NewStateValidator()

	custom := &State{ID: "custom", Name: "Custom", Terminal: false}
	v.AddState(custom)

	if err := v.ValidateState("custom"); err != nil {
		t.Errorf("custom state should be valid: %v", err)
	}

	s, err := v.GetState("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "Custom" {
		t.Errorf("expected name Custom, got %s", s.Name)
	}
}

func TestStateValidator_GetAllStates(t *testing.T) {
	v := NewStateValidator()
	all := v.GetAllStates()

	if len(all) != 10 {
		t.Errorf("expected 10 standard states, got %d", len(all))
	}

	custom := &State{ID: "custom1", Name: "Custom1"}
	v.AddState(custom)
	all = v.GetAllStates()
	if len(all) != 11 {
		t.Errorf("expected 11 states after adding one, got %d", len(all))
	}
}

func TestStateSummary(t *testing.T) {
	now := time.Now()
	s := &StateSummary{
		CurrentState:     "ready",
		PreviousState:    "starting",
		StateEnteredAt:   now,
		StateDuration:    5 * time.Second,
		IsTerminal:       false,
		TotalTransitions: 3,
		StateHistory:     []string{"created", "scheduled", "starting"},
	}

	if s.CurrentState != "ready" {
		t.Errorf("expected ready, got %s", s.CurrentState)
	}
	if s.IsTerminal {
		t.Error("should not be terminal")
	}
	if s.TotalTransitions != 3 {
		t.Errorf("expected 3 transitions, got %d", s.TotalTransitions)
	}
}

func TestState_Metadata(t *testing.T) {
	s := &State{
		ID:          "test",
		Name:        "Test",
		Description: "Test state",
		Terminal:    false,
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	if s.Metadata["key"] != "value" {
		t.Error("metadata not set correctly")
	}
}
