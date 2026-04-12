package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Tracer provides distributed tracing capabilities.
type Tracer struct {
	tracer trace.Tracer
	logger *zap.Logger
	enabled bool
}

// NewTracer creates a new tracer.
func NewTracer(serviceName string, logger *zap.Logger) *Tracer {
	return &Tracer{
		tracer:  otel.Tracer(serviceName),
		logger:  logger,
		enabled: true,
	}
}

// StartSpan starts a new span.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, &noopSpan{}
	}

	ctx, span := t.tracer.Start(ctx, name, opts...)
	return ctx, span
}

// StartSpanWithAttributes starts a new span with attributes.
func (t *Tracer) StartSpanWithAttributes(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, &noopSpan{}
	}

	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	return ctx, span
}

// TraceFunc traces a function execution.
func (t *Tracer) TraceFunc(ctx context.Context, name string, fn func(context.Context) error) error {
	if !t.enabled {
		return fn(ctx)
	}

	ctx, span := t.StartSpan(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// SetEnabled enables or disables tracing.
func (t *Tracer) SetEnabled(enabled bool) {
	t.enabled = enabled
}

// noopSpan is a no-op span implementation.
type noopSpan struct{}

func (s *noopSpan) End(options ...trace.SpanEndOption) {}
func (s *noopSpan) AddEvent(name string, options ...trace.EventOption) {}
func (s *noopSpan) IsRecording() bool { return false }
func (s *noopSpan) RecordError(err error, options ...trace.EventOption) {}
func (s *noopSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }
func (s *noopSpan) SetStatus(code codes.Code, description string) {}
func (s *noopSpan) SetName(name string) {}
func (s *noopSpan) SetAttributes(kv ...attribute.KeyValue) {}
func (s *noopSpan) TracerProvider() trace.TracerProvider { return nil }

// SpanContext manages span context propagation.
type SpanContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// FromContext extracts span context from a context.
func FromContext(ctx context.Context) *SpanContext {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return nil
	}

	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return nil
	}

	return &SpanContext{
		TraceID: spanContext.TraceID().String(),
		SpanID:  spanContext.SpanID().String(),
		Sampled: spanContext.IsSampled(),
	}
}

// ToContext injects span context into a context.
func ToContext(ctx context.Context, spanContext *SpanContext) context.Context {
	if spanContext == nil {
		return ctx
	}

	// This is a simplified version
	// In production, you'd properly reconstruct the SpanContext
	return ctx
}

// TraceIDKey is the context key for trace ID.
type TraceIDKey struct{}

// SpanIDKey is the context key for span ID.
type SpanIDKey struct{}

// ExtractTraceID extracts trace ID from context.
func ExtractTraceID(ctx context.Context) string {
	if span := FromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

// Common span attributes.
func AgentAttributes(agentID string) attribute.KeyValue {
	return attribute.String("agent.id", agentID)
}

func ServiceAttributes(serviceName string) attribute.KeyValue {
	return attribute.String("service.name", serviceName)
}

func OperationAttributes(operation string) attribute.KeyValue {
	return attribute.String("operation", operation)
}

func ErrorAttributes(err error) attribute.KeyValue {
	return attribute.String("error.message", err.Error())
}