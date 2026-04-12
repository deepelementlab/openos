// Package audit provides structured audit events for tenant operations.
package audit

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Event is a minimal audit record aligned with architecture event fields.
type Event struct {
	EventID   string
	TenantID  string
	Actor     string
	Action    string
	Resource  string
	Timestamp time.Time
	TraceID   string
}

// Logger writes audit events (stdout / DB / outbox integration TBD).
type Logger struct {
	z *zap.Logger
}

// NewLogger creates an audit logger.
func NewLogger(z *zap.Logger) *Logger {
	return &Logger{z: z}
}

// Log records an audit event.
func (l *Logger) Log(ctx context.Context, e Event) {
	_ = ctx
	if l.z == nil {
		return
	}
	l.z.Info("audit",
		zap.String("event_id", e.EventID),
		zap.String("tenant_id", e.TenantID),
		zap.String("actor", e.Actor),
		zap.String("action", e.Action),
		zap.String("resource", e.Resource),
		zap.String("trace_id", e.TraceID),
		zap.Time("ts", e.Timestamp),
	)
}
