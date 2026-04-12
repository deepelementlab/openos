package orchestration

import (
	"context"
	"fmt"
	"time"
)

// Transition represents a transition between states.
type Transition struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Event       string                 `json:"event"`
	Guard       TransitionGuard        `json:"-"`         // Optional guard condition
	Actions     []TransitionAction     `json:"-"`         // Actions to execute during transition
	Timeout     time.Duration          `json:"timeout"`   // Transition timeout
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TransitionGuard is a function that determines if a transition is allowed.
type TransitionGuard func(ctx context.Context, from, to string, data interface{}) error

// TransitionAction is a function executed during a transition.
type TransitionAction func(ctx context.Context, from, to string, data interface{}) error

// RetryPolicy defines retry behavior for failed transitions.
type RetryPolicy struct {
	MaxRetries  int           `json:"max_retries"`
	BackoffBase time.Duration `json:"backoff_base"`
	BackoffMax  time.Duration `json:"backoff_max"`
}

// TransitionResult represents the result of a transition.
type TransitionResult struct {
	Success     bool                   `json:"success"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Duration returns the transition duration.
func (r *TransitionResult) Duration() time.Duration {
	if r.CompletedAt.IsZero() {
		return time.Since(r.StartedAt)
	}
	return r.CompletedAt.Sub(r.StartedAt)
}

// TransitionRules defines all valid transitions for a state machine.
type TransitionRules struct {
	transitions map[string][]*Transition // from_state -> transitions
	validator   *StateValidator
}

// NewTransitionRules creates new transition rules.
func NewTransitionRules(validator *StateValidator) *TransitionRules {
	tr := &TransitionRules{
		transitions: make(map[string][]*Transition),
		validator:   validator,
	}

	// Register standard transitions
	tr.registerStandardTransitions()

	return tr
}

// registerStandardTransitions registers the standard Agent lifecycle transitions.
func (tr *TransitionRules) registerStandardTransitions() {
	// [*] -> Created (initial state)
	tr.Register(&Transition{
		ID:    "init-created",
		From:  "*",
		To:    string(StateCreated),
		Event: "create",
	})

	// Created -> Scheduled
	tr.Register(&Transition{
		ID:    "created-scheduled",
		From:  string(StateCreated),
		To:    string(StateScheduled),
		Event: "schedule",
	})

	// Scheduled -> Starting
	tr.Register(&Transition{
		ID:    "scheduled-starting",
		From:  string(StateScheduled),
		To:    string(StateStarting),
		Event: "start",
	})

	// Starting -> Ready
	tr.Register(&Transition{
		ID:    "starting-ready",
		From:  string(StateStarting),
		To:    string(StateReady),
		Event: "ready",
	})

	// Starting -> Failed
	tr.Register(&Transition{
		ID:    "starting-failed",
		From:  string(StateStarting),
		To:    string(StateFailed),
		Event: "fail",
	})

	// Ready -> Stopping
	tr.Register(&Transition{
		ID:    "ready-stopping",
		From:  string(StateReady),
		To:    string(StateStopping),
		Event: "stop",
	})

	// Stopping -> Stopped
	tr.Register(&Transition{
		ID:    "stopping-stopped",
		From:  string(StateStopping),
		To:    string(StateStopped),
		Event: "stopped",
	})

	// Ready -> Failed
	tr.Register(&Transition{
		ID:    "ready-failed",
		From:  string(StateReady),
		To:    string(StateFailed),
		Event: "fail",
	})

	// Failed -> Recovering
	tr.Register(&Transition{
		ID:      "failed-recovering",
		From:    string(StateFailed),
		To:      string(StateRecovering),
		Event:   "recover",
		RetryPolicy: &RetryPolicy{
			MaxRetries:  3,
			BackoffBase: time.Second,
			BackoffMax:  time.Minute,
		},
	})

	// Recovering -> Starting (retry)
	tr.Register(&Transition{
		ID:    "recovering-starting",
		From:  string(StateRecovering),
		To:    string(StateStarting),
		Event: "retry",
	})

	// Recovering -> Failed (recovery failed)
	tr.Register(&Transition{
		ID:    "recovering-failed",
		From:  string(StateRecovering),
		To:    string(StateFailed),
		Event: "recovery_failed",
	})

	// Ready -> Deleting
	tr.Register(&Transition{
		ID:    "ready-deleting",
		From:  string(StateReady),
		To:    string(StateDeleting),
		Event: "delete",
	})

	// Stopped -> Deleting
	tr.Register(&Transition{
		ID:    "stopped-deleting",
		From:  string(StateStopped),
		To:    string(StateDeleting),
		Event: "delete",
	})

	// Failed -> Deleting
	tr.Register(&Transition{
		ID:    "failed-deleting",
		From:  string(StateFailed),
		To:    string(StateDeleting),
		Event: "delete",
	})

	// Deleting -> Deleted
	tr.Register(&Transition{
		ID:    "deleting-deleted",
		From:  string(StateDeleting),
		To:    string(StateDeleted),
		Event: "deleted",
	})
}

// Register adds a transition rule.
func (tr *TransitionRules) Register(transition *Transition) error {
	// Validate states
	if transition.From != "*" {
		if err := tr.validator.ValidateState(transition.From); err != nil {
			return fmt.Errorf("invalid from state: %w", err)
		}
	}
	if err := tr.validator.ValidateState(transition.To); err != nil {
		return fmt.Errorf("invalid to state: %w", err)
	}

	// Check for duplicate
	for _, t := range tr.transitions[transition.From] {
		if t.Event == transition.Event && t.To == transition.To {
			return fmt.Errorf("transition already exists: %s --%s--> %s", transition.From, transition.Event, transition.To)
		}
	}

	tr.transitions[transition.From] = append(tr.transitions[transition.From], transition)
	return nil
}

// GetTransitions returns all transitions from a state.
func (tr *TransitionRules) GetTransitions(from string) []*Transition {
	result := make([]*Transition, 0)

	// Get transitions from specific state
	if transitions, exists := tr.transitions[from]; exists {
		result = append(result, transitions...)
	}

	// Get transitions from wildcard "*" (initial state)
	if from != "*" {
		if transitions, exists := tr.transitions["*"]; exists {
			result = append(result, transitions...)
		}
	}

	return result
}

// FindTransition finds a valid transition from current state for an event.
func (tr *TransitionRules) FindTransition(from, event string) (*Transition, error) {
	transitions := tr.GetTransitions(from)

	for _, t := range transitions {
		if t.Event == event {
			return t, nil
		}
	}

	return nil, fmt.Errorf("no transition found for event %s from state %s", event, from)
}

// IsValid checks if a transition is valid.
func (tr *TransitionRules) IsValid(from, to, event string) bool {
	transitions := tr.GetTransitions(from)

	for _, t := range transitions {
		if t.To == to && t.Event == event {
			return true
		}
	}

	return false
}

// GetAllTransitions returns all registered transitions.
func (tr *TransitionRules) GetAllTransitions() []*Transition {
	all := make([]*Transition, 0)
	seen := make(map[string]bool)

	for _, transitions := range tr.transitions {
		for _, t := range transitions {
			key := fmt.Sprintf("%s-%s-%s", t.From, t.Event, t.To)
			if !seen[key] {
				seen[key] = true
				all = append(all, t)
			}
		}
	}

	return all
}

// TransitionExecutor executes state transitions.
type TransitionExecutor struct {
	rules *TransitionRules
}

// NewTransitionExecutor creates a new transition executor.
func NewTransitionExecutor(rules *TransitionRules) *TransitionExecutor {
	return &TransitionExecutor{
		rules: rules,
	}
}

// Execute executes a transition.
func (te *TransitionExecutor) Execute(ctx context.Context, from, event string, data interface{}) (*TransitionResult, error) {
	// Find transition
	transition, err := te.rules.FindTransition(from, event)
	if err != nil {
		return nil, err
	}

	result := &TransitionResult{
		From:      from,
		To:        transition.To,
		StartedAt: time.Now().UTC(),
		Success:   true,
	}

	// Execute guard if present
	if transition.Guard != nil {
		if err := transition.Guard(ctx, from, transition.To, data); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("guard failed: %v", err)
			result.CompletedAt = time.Now().UTC()
			return result, nil
		}
	}

	// Execute actions with retry if needed
	maxRetries := 0
	if transition.RetryPolicy != nil {
		maxRetries = transition.RetryPolicy.MaxRetries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = te.executeActions(ctx, transition, data)
		if err == nil {
			break
		}

		result.RetryCount = attempt + 1

		if attempt < maxRetries && transition.RetryPolicy != nil {
			backoff := transition.RetryPolicy.BackoffBase * time.Duration(attempt+1)
			if backoff > transition.RetryPolicy.BackoffMax {
				backoff = transition.RetryPolicy.BackoffMax
			}
			time.Sleep(backoff)
		} else {
			result.Success = false
			result.Error = err.Error()
			result.CompletedAt = time.Now().UTC()
			return result, nil
		}
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

// executeActions executes all transition actions.
func (te *TransitionExecutor) executeActions(ctx context.Context, transition *Transition, data interface{}) error {
	for _, action := range transition.Actions {
		if err := action(ctx, transition.From, transition.To, data); err != nil {
			return fmt.Errorf("action failed: %w", err)
		}
	}
	return nil
}

// CanTransition checks if a transition is possible.
func (te *TransitionExecutor) CanTransition(from, event string) bool {
	_, err := te.rules.FindTransition(from, event)
	return err == nil
}
