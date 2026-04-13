// Package falco integrates Falco runtime security events into AOS reactions.
package falco

import (
	"context"
	"encoding/json"
	"fmt"
)

// Event is a minimal Falco JSON event (fields vary by rule).
type Event struct {
	Rule      string            `json:"rule"`
	Priority  string            `json:"priority"`
	Output    string            `json:"output"`
	Labels    map[string]string `json:"labels"`
	AgentID   string            `json:"agent_id,omitempty"`
	TenantID  string            `json:"tenant_id,omitempty"`
}

// Handler processes Falco events (e.g. isolate agent).
type Handler interface {
	Handle(ctx context.Context, e Event) error
}

// ParseEvent decodes a webhook/gRPC payload.
func ParseEvent(data []byte) (Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return Event{}, fmt.Errorf("falco: %w", err)
	}
	return e, nil
}

// DefaultIsolateHandler is a stub that logs intent; wire to runtime isolate API.
type DefaultIsolateHandler struct{}

// Handle implements Handler.
func (DefaultIsolateHandler) Handle(ctx context.Context, e Event) error {
	if e.Priority == "CRITICAL" || e.Priority == "ERROR" {
		return fmt.Errorf("falco: critical event on agent %s: %s", e.AgentID, e.Rule)
	}
	return nil
}
