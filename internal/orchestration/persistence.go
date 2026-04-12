package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// StateMachineState represents a persisted state machine instance.
type StateMachineState struct {
	ID             string                 `db:"id" json:"id"`
	EntityID       string                 `db:"entity_id" json:"entity_id"`
	EntityType     string                 `db:"entity_type" json:"entity_type"`
	CurrentState   string                 `db:"current_state" json:"current_state"`
	PreviousState  string                 `db:"previous_state" json:"previous_state"`
	StateData      map[string]interface{} `db:"-" json:"state_data"`
	StateDataJSON  []byte                 `db:"state_data" json:"-"`
	Version        int                    `db:"version" json:"version"`
	CreatedAt      time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time              `db:"updated_at" json:"updated_at"`
	CompletedAt    *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
}

// BeforeInsert prepares the record for insertion.
func (s *StateMachineState) BeforeInsert() error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = s.CreatedAt
	}
	if s.Version == 0 {
		s.Version = 1
	}

	// Serialize state data
	if s.StateData != nil {
		data, err := json.Marshal(s.StateData)
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %w", err)
		}
		s.StateDataJSON = data
	}

	return nil
}

// AfterScan handles post-scan initialization.
func (s *StateMachineState) AfterScan() error {
	if len(s.StateDataJSON) > 0 {
		if err := json.Unmarshal(s.StateDataJSON, &s.StateData); err != nil {
			return fmt.Errorf("failed to unmarshal state data: %w", err)
		}
	}
	return nil
}

// IsComplete checks if the state machine has completed.
func (s *StateMachineState) IsComplete() bool {
	return s.CompletedAt != nil
}

// TransitionHistoryRecord represents a persisted state transition.
type TransitionHistoryRecord struct {
	ID              string    `db:"id" json:"id"`
	StateMachineID  string    `db:"state_machine_id" json:"state_machine_id"`
	EntityID        string    `db:"entity_id" json:"entity_id"`
	FromState       string    `db:"from_state" json:"from_state"`
	ToState         string    `db:"to_state" json:"to_state"`
	Event           string    `db:"event" json:"event"`
	Success         bool      `db:"success" json:"success"`
	Error           string    `db:"error" json:"error,omitempty"`
	RetryCount      int       `db:"retry_count" json:"retry_count"`
	DurationMs      int64     `db:"duration_ms" json:"duration_ms"`
	StartedAt       time.Time `db:"started_at" json:"started_at"`
	CompletedAt     time.Time `db:"completed_at" json:"completed_at"`
}

// Persistence defines the interface for state machine persistence.
type Persistence interface {
	// SaveState saves the current state machine state.
	SaveState(ctx context.Context, state *StateMachineState) error

	// GetState retrieves a state machine by entity ID.
	GetState(ctx context.Context, entityID string) (*StateMachineState, error)

	// SaveTransition saves a transition history record.
	SaveTransition(ctx context.Context, record *TransitionHistoryRecord) error

	// GetTransitionHistory retrieves transition history for a state machine.
	GetTransitionHistory(ctx context.Context, stateMachineID string) ([]*TransitionHistoryRecord, error)

	// ListActive returns all active (non-completed) state machines.
	ListActive(ctx context.Context, entityType string) ([]*StateMachineState, error)

	// DeleteState removes a state machine state.
	DeleteState(ctx context.Context, entityID string) error
}

// SQLPersistence implements Persistence using a SQL database.
type SQLPersistence struct {
	db *sqlx.DB
}

// NewSQLPersistence creates a new SQL persistence layer.
func NewSQLPersistence(db *sqlx.DB) *SQLPersistence {
	return &SQLPersistence{db: db}
}

// SaveState saves the current state machine state.
func (p *SQLPersistence) SaveState(ctx context.Context, state *StateMachineState) error {
	if err := state.BeforeInsert(); err != nil {
		return err
	}

	query := `
		INSERT INTO state_machines (
			id, entity_id, entity_type, current_state, previous_state, state_data, version, created_at, updated_at, completed_at
		) VALUES (
			:id, :entity_id, :entity_type, :current_state, :previous_state, :state_data, :version, :created_at, :updated_at, :completed_at
		)
		ON CONFLICT (entity_id) DO UPDATE SET
			current_state = EXCLUDED.current_state,
			previous_state = EXCLUDED.previous_state,
			state_data = EXCLUDED.state_data,
			version = state_machines.version + 1,
			updated_at = EXCLUDED.updated_at,
			completed_at = EXCLUDED.completed_at
		WHERE state_machines.version = EXCLUDED.version
	`

	result, err := p.db.NamedExecContext(ctx, query, state)
	if err != nil {
		return fmt.Errorf("failed to save state machine: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("concurrent modification detected for entity %s", state.EntityID)
	}

	state.Version++
	return nil
}

// GetState retrieves a state machine by entity ID.
func (p *SQLPersistence) GetState(ctx context.Context, entityID string) (*StateMachineState, error) {
	var state StateMachineState
	query := `SELECT * FROM state_machines WHERE entity_id = $1`

	if err := p.db.GetContext(ctx, &state, query, entityID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state machine not found for entity %s", entityID)
		}
		return nil, fmt.Errorf("failed to get state machine: %w", err)
	}

	if err := state.AfterScan(); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveTransition saves a transition history record.
func (p *SQLPersistence) SaveTransition(ctx context.Context, record *TransitionHistoryRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	query := `
		INSERT INTO state_transitions (
			id, state_machine_id, entity_id, from_state, to_state, event, success, error, retry_count, duration_ms, started_at, completed_at
		) VALUES (
			:id, :state_machine_id, :entity_id, :from_state, :to_state, :event, :success, :error, :retry_count, :duration_ms, :started_at, :completed_at
		)
	`

	_, err := p.db.NamedExecContext(ctx, query, record)
	if err != nil {
		return fmt.Errorf("failed to save transition: %w", err)
	}

	return nil
}

// GetTransitionHistory retrieves transition history for a state machine.
func (p *SQLPersistence) GetTransitionHistory(ctx context.Context, stateMachineID string) ([]*TransitionHistoryRecord, error) {
	query := `SELECT * FROM state_transitions WHERE state_machine_id = $1 ORDER BY completed_at ASC`

	var records []*TransitionHistoryRecord
	if err := p.db.SelectContext(ctx, &records, query, stateMachineID); err != nil {
		return nil, fmt.Errorf("failed to get transition history: %w", err)
	}

	return records, nil
}

// ListActive returns all active (non-completed) state machines.
func (p *SQLPersistence) ListActive(ctx context.Context, entityType string) ([]*StateMachineState, error) {
	query := `SELECT * FROM state_machines WHERE entity_type = $1 AND completed_at IS NULL ORDER BY updated_at DESC`

	var states []*StateMachineState
	if err := p.db.SelectContext(ctx, &states, query, entityType); err != nil {
		return nil, fmt.Errorf("failed to list active state machines: %w", err)
	}

	for _, state := range states {
		if err := state.AfterScan(); err != nil {
			return nil, err
		}
	}

	return states, nil
}

// DeleteState removes a state machine state.
func (p *SQLPersistence) DeleteState(ctx context.Context, entityID string) error {
	query := `DELETE FROM state_machines WHERE entity_id = $1`

	_, err := p.db.ExecContext(ctx, query, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete state machine: %w", err)
	}

	return nil
}

// InMemoryPersistence implements Persistence for testing.
type InMemoryPersistence struct {
	states     map[string]*StateMachineState
	transitions map[string][]*TransitionHistoryRecord
	mu         sync.RWMutex
}

// NewInMemoryPersistence creates an in-memory persistence layer for testing.
func NewInMemoryPersistence() *InMemoryPersistence {
	return &InMemoryPersistence{
		states:      make(map[string]*StateMachineState),
		transitions: make(map[string][]*TransitionHistoryRecord),
	}
}

// SaveState saves the current state machine state.
func (p *InMemoryPersistence) SaveState(ctx context.Context, state *StateMachineState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := state.BeforeInsert(); err != nil {
		return err
	}

	if existing, exists := p.states[state.EntityID]; exists {
		if existing.Version != state.Version {
			return fmt.Errorf("concurrent modification detected for entity %s", state.EntityID)
		}
		state.Version++
		state.UpdatedAt = time.Now().UTC()
	}

	// Deep copy
	stateCopy := &StateMachineState{
		ID:            state.ID,
		EntityID:      state.EntityID,
		EntityType:    state.EntityType,
		CurrentState:  state.CurrentState,
		PreviousState: state.PreviousState,
		StateData:     deepCopyMap(state.StateData),
		Version:       state.Version,
		CreatedAt:     state.CreatedAt,
		UpdatedAt:     state.UpdatedAt,
		CompletedAt:   state.CompletedAt,
	}
	if state.StateData != nil {
		stateCopy.StateDataJSON, _ = json.Marshal(state.StateData)
	}

	p.states[state.EntityID] = stateCopy
	return nil
}

// GetState retrieves a state machine by entity ID.
func (p *InMemoryPersistence) GetState(ctx context.Context, entityID string) (*StateMachineState, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, exists := p.states[entityID]
	if !exists {
		return nil, fmt.Errorf("state machine not found for entity %s", entityID)
	}

	// Deep copy
	stateCopy := *state
	stateCopy.StateData = deepCopyMap(state.StateData)
	return &stateCopy, nil
}

// SaveTransition saves a transition history record.
func (p *InMemoryPersistence) SaveTransition(ctx context.Context, record *TransitionHistoryRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}

	recordCopy := *record
	p.transitions[record.StateMachineID] = append(p.transitions[record.StateMachineID], &recordCopy)
	return nil
}

// GetTransitionHistory retrieves transition history for a state machine.
func (p *InMemoryPersistence) GetTransitionHistory(ctx context.Context, stateMachineID string) ([]*TransitionHistoryRecord, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	transitions := p.transitions[stateMachineID]
	result := make([]*TransitionHistoryRecord, len(transitions))
	for i, t := range transitions {
		copy := *t
		result[i] = &copy
	}
	return result, nil
}

// ListActive returns all active (non-completed) state machines.
func (p *InMemoryPersistence) ListActive(ctx context.Context, entityType string) ([]*StateMachineState, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*StateMachineState
	for _, state := range p.states {
		if state.EntityType == entityType && !state.IsComplete() {
			stateCopy := *state
			stateCopy.StateData = deepCopyMap(state.StateData)
			result = append(result, &stateCopy)
		}
	}

	return result, nil
}

// DeleteState removes a state machine state.
func (p *InMemoryPersistence) DeleteState(ctx context.Context, entityID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.states, entityID)
	return nil
}

// deepCopyMap creates a deep copy of a map.
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// EnsureTables creates the necessary database tables.
func EnsureTables(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS state_machines (
		id UUID PRIMARY KEY,
		entity_id VARCHAR(255) UNIQUE NOT NULL,
		entity_type VARCHAR(50) NOT NULL,
		current_state VARCHAR(50) NOT NULL,
		previous_state VARCHAR(50),
		state_data JSONB,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_state_machines_entity ON state_machines(entity_type, current_state);
	CREATE INDEX IF NOT EXISTS idx_state_machines_active ON state_machines(completed_at) WHERE completed_at IS NULL;

	CREATE TABLE IF NOT EXISTS state_transitions (
		id UUID PRIMARY KEY,
		state_machine_id UUID NOT NULL REFERENCES state_machines(id),
		entity_id VARCHAR(255) NOT NULL,
		from_state VARCHAR(50) NOT NULL,
		to_state VARCHAR(50) NOT NULL,
		event VARCHAR(50) NOT NULL,
		success BOOLEAN NOT NULL,
		error TEXT,
		retry_count INTEGER DEFAULT 0,
		duration_ms BIGINT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_state_transitions_sm ON state_transitions(state_machine_id);
	CREATE INDEX IF NOT EXISTS idx_state_transitions_time ON state_transitions(completed_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create state machine tables: %w", err)
	}

	return nil
}
