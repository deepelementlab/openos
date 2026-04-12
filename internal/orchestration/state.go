package orchestration

import (
	"fmt"
	"time"
)

// State represents a state in the state machine.
type State struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Terminal    bool                   `json:"terminal"` // If true, this is a final state
	Metadata    map[string]interface{} `json:"metadata"`
}

// StateType represents predefined state types.
type StateType string

const (
	// Agent lifecycle states
	StateCreated    StateType = "created"
	StateScheduled  StateType = "scheduled"
	StateStarting   StateType = "starting"
	StateReady      StateType = "ready"
	StateStopping   StateType = "stopping"
	StateStopped    StateType = "stopped"
	StateFailed     StateType = "failed"
	StateRecovering StateType = "recovering"
	StateDeleting   StateType = "deleting"
	StateDeleted    StateType = "deleted"

	// Workflow states
	StatePending    StateType = "pending"
	StateRunning    StateType = "running"
	StateCompleted  StateType = "completed"
	StateCancelled  StateType = "cancelled"
)

// StandardStates provides standard Agent lifecycle states.
func StandardStates() []*State {
	return []*State{
		{
			ID:          string(StateCreated),
			Name:        "Created",
			Description: "Agent has been created but not yet scheduled",
			Terminal:    false,
		},
		{
			ID:          string(StateScheduled),
			Name:        "Scheduled",
			Description: "Agent has been scheduled on a node",
			Terminal:    false,
		},
		{
			ID:          string(StateStarting),
			Name:        "Starting",
			Description: "Agent is starting on the node",
			Terminal:    false,
		},
		{
			ID:          string(StateReady),
			Name:        "Ready",
			Description: "Agent is ready and running",
			Terminal:    false,
		},
		{
			ID:          string(StateStopping),
			Name:        "Stopping",
			Description: "Agent is stopping",
			Terminal:    false,
		},
		{
			ID:          string(StateStopped),
			Name:        "Stopped",
			Description: "Agent has stopped",
			Terminal:    true,
		},
		{
			ID:          string(StateFailed),
			Name:        "Failed",
			Description: "Agent has failed",
			Terminal:    true,
		},
		{
			ID:          string(StateRecovering),
			Name:        "Recovering",
			Description: "Agent is recovering from failure",
			Terminal:    false,
		},
		{
			ID:          string(StateDeleting),
			Name:        "Deleting",
			Description: "Agent is being deleted",
			Terminal:    false,
		},
		{
			ID:          string(StateDeleted),
			Name:        "Deleted",
			Description: "Agent has been deleted",
			Terminal:    true,
		},
	}
}

// StateInstance represents a specific state instance in a state machine execution.
type StateInstance struct {
	StateID     string                 `json:"state_id"`
	EntityID    string                 `json:"entity_id"`    // e.g., Agent ID
	EntityType  string                 `json:"entity_type"`  // e.g., "agent"
	EnteredAt   time.Time              `json:"entered_at"`
	ExitedAt    *time.Time             `json:"exited_at,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// IsTerminal checks if the state is terminal.
func (si *StateInstance) IsTerminal(states map[string]*State) bool {
	state, exists := states[si.StateID]
	if !exists {
		return false
	}
	return state.Terminal
}

// Duration returns how long the state has been active.
func (si *StateInstance) Duration() time.Duration {
	if si.ExitedAt != nil {
		return si.ExitedAt.Sub(si.EnteredAt)
	}
	return time.Since(si.EnteredAt)
}

// StateValidator validates state transitions.
type StateValidator struct {
	validStates map[string]*State
}

// NewStateValidator creates a new state validator with standard states.
func NewStateValidator() *StateValidator {
	states := make(map[string]*State)
	for _, state := range StandardStates() {
		states[state.ID] = state
	}

	return &StateValidator{
		validStates: states,
	}
}

// ValidateState checks if a state ID is valid.
func (v *StateValidator) ValidateState(stateID string) error {
	if _, exists := v.validStates[stateID]; !exists {
		return fmt.Errorf("invalid state: %s", stateID)
	}
	return nil
}

// IsTerminal checks if a state is terminal.
func (v *StateValidator) IsTerminal(stateID string) bool {
	if state, exists := v.validStates[stateID]; exists {
		return state.Terminal
	}
	return false
}

// GetState retrieves a state definition.
func (v *StateValidator) GetState(stateID string) (*State, error) {
	state, exists := v.validStates[stateID]
	if !exists {
		return nil, fmt.Errorf("state not found: %s", stateID)
	}
	return state, nil
}

// AddState adds a custom state.
func (v *StateValidator) AddState(state *State) {
	v.validStates[state.ID] = state
}

// GetAllStates returns all valid states.
func (v *StateValidator) GetAllStates() []*State {
	states := make([]*State, 0, len(v.validStates))
	for _, state := range v.validStates {
		states = append(states, state)
	}
	return states
}

// StateSummary provides a summary of current state.
type StateSummary struct {
	CurrentState     string         `json:"current_state"`
	PreviousState    string         `json:"previous_state,omitempty"`
	StateEnteredAt   time.Time      `json:"state_entered_at"`
	StateDuration    time.Duration  `json:"state_duration"`
	IsTerminal       bool           `json:"is_terminal"`
	TotalTransitions int            `json:"total_transitions"`
	StateHistory     []string       `json:"state_history,omitempty"`
}
