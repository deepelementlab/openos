package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/internal/messaging"
	"go.uber.org/zap"
)

// StateMachineEngine is the core state machine orchestration engine.
type StateMachineEngine struct {
	validator   *StateValidator
	rules       *TransitionRules
	executor    *TransitionExecutor
	persistence Persistence
	eventBus    messaging.EventBus
	logger      *zap.Logger
	mu          sync.RWMutex
	machines    map[string]*StateMachineInstance // entity_id -> instance
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// StateMachineInstance represents a running state machine instance.
type StateMachineInstance struct {
	ID          string
	EntityID    string
	EntityType  string
	CurrentState string
	PreviousState string
	StateData   map[string]interface{}
	StartTime   time.Time
	Version     int
}

// StateMachineOptions provides options for creating a state machine engine.
type StateMachineOptions struct {
	Persistence Persistence
	EventBus    messaging.EventBus
	Logger      *zap.Logger
}

// NewStateMachineEngine creates a new state machine engine.
func NewStateMachineEngine(opts StateMachineOptions) *StateMachineEngine {
	validator := NewStateValidator()
	rules := NewTransitionRules(validator)
	executor := NewTransitionExecutor(rules)

	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &StateMachineEngine{
		validator:   validator,
		rules:       rules,
		executor:    executor,
		persistence: opts.Persistence,
		eventBus:    opts.EventBus,
		logger:      logger,
		machines:    make(map[string]*StateMachineInstance),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the state machine engine.
func (e *StateMachineEngine) Start(ctx context.Context) error {
	e.logger.Info("Starting state machine engine")

	// Load active state machines from persistence
	if e.persistence != nil {
		states, err := e.persistence.ListActive(ctx, "agent")
		if err != nil {
			e.logger.Warn("Failed to load active state machines", zap.Error(err))
		} else {
			for _, state := range states {
				instance := &StateMachineInstance{
					ID:            state.ID,
					EntityID:      state.EntityID,
					EntityType:    state.EntityType,
					CurrentState:  state.CurrentState,
					PreviousState: state.PreviousState,
					StateData:     state.StateData,
					Version:       state.Version,
				}
				e.machines[state.EntityID] = instance
				e.logger.Debug("Loaded state machine",
					zap.String("entity_id", state.EntityID),
					zap.String("current_state", state.CurrentState),
				)
			}
			e.logger.Info("Loaded active state machines", zap.Int("count", len(states)))
		}
	}

	return nil
}

// Stop stops the state machine engine.
func (e *StateMachineEngine) Stop() error {
	e.logger.Info("Stopping state machine engine")
	close(e.stopCh)
	e.wg.Wait()
	e.logger.Info("State machine engine stopped")
	return nil
}

// CreateMachine creates a new state machine instance.
func (e *StateMachineEngine) CreateMachine(ctx context.Context, entityID, entityType string, initialData map[string]interface{}) (*StateMachineInstance, error) {
	// Validate that initial state is valid
	initialState := string(StateCreated)
	if err := e.validator.ValidateState(initialState); err != nil {
		return nil, fmt.Errorf("invalid initial state: %w", err)
	}

	instance := &StateMachineInstance{
		ID:           fmt.Sprintf("sm-%s", entityID),
		EntityID:     entityID,
		EntityType:   entityType,
		CurrentState: initialState,
		StateData:    initialData,
		StartTime:    time.Now().UTC(),
		Version:      1,
	}

	// Persist the state machine
	if e.persistence != nil {
		state := &StateMachineState{
			ID:           instance.ID,
			EntityID:     entityID,
			EntityType:   entityType,
			CurrentState: initialState,
			StateData:    initialData,
			Version:      1,
		}

		if err := e.persistence.SaveState(ctx, state); err != nil {
			return nil, fmt.Errorf("failed to persist state machine: %w", err)
		}
	}

	// Store in memory
	e.mu.Lock()
	e.machines[entityID] = instance
	e.mu.Unlock()

	// Publish event
	if e.eventBus != nil {
		event, _ := messaging.NewEvent(messaging.EventAgentCreated, map[string]interface{}{
			"entity_id": entityID,
			"entity_type": entityType,
			"state": initialState,
		})
		event.SetAgentID(entityID)
		if err := e.eventBus.Publish(ctx, event); err != nil {
			e.logger.Warn("Failed to publish state machine created event", zap.Error(err))
		}
	}

	e.logger.Info("Created state machine",
		zap.String("entity_id", entityID),
		zap.String("entity_type", entityType),
	)

	return instance, nil
}

// GetMachine retrieves a state machine instance by entity ID.
func (e *StateMachineEngine) GetMachine(entityID string) (*StateMachineInstance, error) {
	e.mu.RLock()
	instance, exists := e.machines[entityID]
	e.mu.RUnlock()

	if exists {
		return instance, nil
	}

	// Try to load from persistence
	if e.persistence != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state, err := e.persistence.GetState(ctx, entityID)
		if err != nil {
			return nil, err
		}

		instance = &StateMachineInstance{
			ID:            state.ID,
			EntityID:      state.EntityID,
			EntityType:    state.EntityType,
			CurrentState:  state.CurrentState,
			PreviousState: state.PreviousState,
			StateData:     state.StateData,
			Version:       state.Version,
		}

		e.mu.Lock()
		e.machines[entityID] = instance
		e.mu.Unlock()

		return instance, nil
	}

	return nil, fmt.Errorf("state machine not found for entity %s", entityID)
}

// SendEvent sends an event to a state machine to trigger a transition.
func (e *StateMachineEngine) SendEvent(ctx context.Context, entityID, event string, data interface{}) (*TransitionResult, error) {
	// Get the state machine
	instance, err := e.GetMachine(entityID)
	if err != nil {
		return nil, err
	}

	// Execute the transition
	result, err := e.executor.Execute(ctx, instance.CurrentState, event, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transition: %w", err)
	}

	// Save the transition record
	if e.persistence != nil {
		record := &TransitionHistoryRecord{
			StateMachineID: instance.ID,
			EntityID:       entityID,
			FromState:      result.From,
			ToState:        result.To,
			Event:          event,
			Success:        result.Success,
			Error:          result.Error,
			RetryCount:     result.RetryCount,
			DurationMs:     int64(result.Duration().Milliseconds()),
			StartedAt:      result.StartedAt,
			CompletedAt:    result.CompletedAt,
		}

		if err := e.persistence.SaveTransition(ctx, record); err != nil {
			e.logger.Warn("Failed to save transition", zap.Error(err))
		}
	}

	// Update instance if successful
	if result.Success {
		instance.PreviousState = instance.CurrentState
		instance.CurrentState = result.To
		instance.Version++

		if dataMap, ok := data.(map[string]interface{}); ok {
			// Merge new data into state data
			for k, v := range dataMap {
				instance.StateData[k] = v
			}
		}

		// Persist state change
		if e.persistence != nil {
			state := &StateMachineState{
				ID:            instance.ID,
				EntityID:      instance.EntityID,
				EntityType:    instance.EntityType,
				CurrentState:  instance.CurrentState,
				PreviousState: instance.PreviousState,
				StateData:     instance.StateData,
				Version:       instance.Version,
			}

			// Check if terminal state
			if e.validator.IsTerminal(instance.CurrentState) {
				now := time.Now().UTC()
				state.CompletedAt = &now
			}

			if err := e.persistence.SaveState(ctx, state); err != nil {
				e.logger.Error("Failed to persist state change", zap.Error(err))
			}
		}

		// Publish state change event
		e.publishStateChangeEvent(ctx, instance, result)
	}

	e.logger.Debug("State transition executed",
		zap.String("entity_id", entityID),
		zap.String("event", event),
		zap.String("from", result.From),
		zap.String("to", result.To),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// publishStateChangeEvent publishes a state change event to the event bus.
func (e *StateMachineEngine) publishStateChangeEvent(ctx context.Context, instance *StateMachineInstance, result *TransitionResult) {
	if e.eventBus == nil {
		return
	}

	// Map state to event type
	var eventType string
	switch result.To {
	case string(StateCreated):
		eventType = messaging.EventAgentCreated
	case string(StateScheduled):
		eventType = messaging.EventAgentScheduled
	case string(StateStarting):
		eventType = messaging.EventAgentStarting
	case string(StateReady):
		eventType = messaging.EventAgentReady
	case string(StateStopping):
		eventType = messaging.EventAgentStopping
	case string(StateStopped):
		eventType = messaging.EventAgentStopped
	case string(StateFailed):
		eventType = messaging.EventAgentFailed
	case string(StateRecovering):
		eventType = messaging.EventAgentRecovered
	default:
		eventType = fmt.Sprintf("agent.state.%s", result.To)
	}

	event, _ := messaging.NewEvent(eventType, map[string]interface{}{
		"entity_id":      instance.EntityID,
		"entity_type":    instance.EntityType,
		"previous_state": result.From,
		"current_state":  result.To,
		"transition_success": result.Success,
		"transition_duration_ms": result.Duration().Milliseconds(),
	})
	event.SetAgentID(instance.EntityID)

	if err := e.eventBus.Publish(ctx, event); err != nil {
		e.logger.Warn("Failed to publish state change event", zap.Error(err))
	}
}

// CanTransition checks if a transition is possible.
func (e *StateMachineEngine) CanTransition(entityID, event string) bool {
	instance, err := e.GetMachine(entityID)
	if err != nil {
		return false
	}

	return e.executor.CanTransition(instance.CurrentState, event)
}

// GetCurrentState returns the current state of a state machine.
func (e *StateMachineEngine) GetCurrentState(entityID string) (string, error) {
	instance, err := e.GetMachine(entityID)
	if err != nil {
		return "", err
	}

	return instance.CurrentState, nil
}

// GetStateSummary returns a summary of the state machine state.
func (e *StateMachineEngine) GetStateSummary(entityID string) (*StateSummary, error) {
	instance, err := e.GetMachine(entityID)
	if err != nil {
		return nil, err
	}

	var history []string
	if e.persistence != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		records, err := e.persistence.GetTransitionHistory(ctx, instance.ID)
		if err == nil {
			for _, r := range records {
				history = append(history, r.FromState, r.ToState)
			}
		}
	}

	return &StateSummary{
		CurrentState:     instance.CurrentState,
		PreviousState:    instance.PreviousState,
		StateEnteredAt:   instance.StartTime,
		StateDuration:    time.Since(instance.StartTime),
		IsTerminal:       e.validator.IsTerminal(instance.CurrentState),
		TotalTransitions: len(history),
		StateHistory:     history,
	}, nil
}

// ListActive returns all active state machines.
func (e *StateMachineEngine) ListActive() []*StateMachineInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*StateMachineInstance, 0, len(e.machines))
	for _, instance := range e.machines {
		if !e.validator.IsTerminal(instance.CurrentState) {
			result = append(result, instance)
		}
	}

	return result
}

// RegisterCustomTransition registers a custom transition rule.
func (e *StateMachineEngine) RegisterCustomTransition(transition *Transition) error {
	return e.rules.Register(transition)
}

// AddCustomState adds a custom state.
func (e *StateMachineEngine) AddCustomState(state *State) {
	e.validator.AddState(state)
}

// Stats returns engine statistics.
func (e *StateMachineEngine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := 0
	terminal := 0
	for _, instance := range e.machines {
		if e.validator.IsTerminal(instance.CurrentState) {
			terminal++
		} else {
			active++
		}
	}

	return map[string]interface{}{
		"total_machines":     len(e.machines),
		"active_machines":    active,
		"terminal_machines":  terminal,
		"valid_states":       len(e.validator.GetAllStates()),
		"transition_rules":   len(e.rules.GetAllTransitions()),
	}
}
