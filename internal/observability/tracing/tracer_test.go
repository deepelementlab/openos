package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func TestNewTracer(t *testing.T) {
	tr := NewTracer("test-service", zap.NewNop())
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
	if !tr.enabled {
		t.Fatal("expected tracer to be enabled by default")
	}
}

func TestTracer_SetEnabled(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	tr.SetEnabled(false)
	if tr.enabled {
		t.Fatal("expected tracer to be disabled")
	}
	tr.SetEnabled(true)
	if !tr.enabled {
		t.Fatal("expected tracer to be enabled")
	}
}

func TestTracer_StartSpan_Disabled(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	tr.SetEnabled(false)
	_, span := tr.StartSpan(context.Background(), "test-op")
	if span == nil {
		t.Fatal("expected non-nil span when disabled")
	}
	if span.IsRecording() {
		t.Fatal("expected noop span to not be recording")
	}
}

func TestTracer_StartSpan_Enabled(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	ctx, span := tr.StartSpan(context.Background(), "test-op")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	span.End()
}

func TestTracer_StartSpanWithAttributes_Disabled(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	tr.SetEnabled(false)
	ctx, span := tr.StartSpanWithAttributes(context.Background(), "op",
		attribute.String("key", "value"),
	)
	if span == nil {
		t.Fatal("expected non-nil span when disabled")
	}
	if span.IsRecording() {
		t.Fatal("expected noop span to not be recording")
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestTracer_StartSpanWithAttributes_Enabled(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	_, span := tr.StartSpanWithAttributes(context.Background(), "op",
		attribute.String("key", "val"),
	)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
}

func TestTracer_TraceFunc_Disabled_Success(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	tr.SetEnabled(false)
	called := false
	err := tr.TraceFunc(context.Background(), "op", func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected function to be called")
	}
}

func TestTracer_TraceFunc_Disabled_Error(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	tr.SetEnabled(false)
	expectedErr := errors.New("boom")
	err := tr.TraceFunc(context.Background(), "op", func(ctx context.Context) error {
		return expectedErr
	})
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestTracer_TraceFunc_Enabled_Success(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	err := tr.TraceFunc(context.Background(), "op", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracer_TraceFunc_Enabled_Error(t *testing.T) {
	tr := NewTracer("svc", zap.NewNop())
	expectedErr := errors.New("fail")
	err := tr.TraceFunc(context.Background(), "op", func(ctx context.Context) error {
		return expectedErr
	})
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestFromContext_NoSpan(t *testing.T) {
	sc := FromContext(context.Background())
	if sc != nil {
		t.Fatal("expected nil SpanContext from empty context")
	}
}

func TestToContext_NilSpanContext(t *testing.T) {
	ctx := context.Background()
	result := ToContext(ctx, nil)
	if result != ctx {
		t.Fatal("expected same context when spanContext is nil")
	}
}

func TestToContext_NonNilSpanContext(t *testing.T) {
	ctx := context.Background()
	sc := &SpanContext{TraceID: "abc", SpanID: "123", Sampled: true}
	result := ToContext(ctx, sc)
	if result == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestExtractTraceID_NoSpan(t *testing.T) {
	id := ExtractTraceID(context.Background())
	if id != "" {
		t.Fatalf("expected empty trace ID, got %q", id)
	}
}

func TestSpanContext_Struct(t *testing.T) {
	sc := SpanContext{TraceID: "trace123", SpanID: "span456", Sampled: true}
	if sc.TraceID != "trace123" {
		t.Fatalf("expected trace123, got %s", sc.TraceID)
	}
	if sc.SpanID != "span456" {
		t.Fatalf("expected span456, got %s", sc.SpanID)
	}
	if !sc.Sampled {
		t.Fatal("expected Sampled to be true")
	}
}

func TestAgentAttributes(t *testing.T) {
	kv := AgentAttributes("agent-42")
	if string(kv.Key) != "agent.id" {
		t.Fatalf("expected key 'agent.id', got %s", kv.Key)
	}
	if kv.Value.AsString() != "agent-42" {
		t.Fatalf("expected 'agent-42', got %s", kv.Value.AsString())
	}
}

func TestServiceAttributes(t *testing.T) {
	kv := ServiceAttributes("my-svc")
	if string(kv.Key) != "service.name" {
		t.Fatalf("expected key 'service.name', got %s", kv.Key)
	}
	if kv.Value.AsString() != "my-svc" {
		t.Fatalf("expected 'my-svc', got %s", kv.Value.AsString())
	}
}

func TestOperationAttributes(t *testing.T) {
	kv := OperationAttributes("create")
	if string(kv.Key) != "operation" {
		t.Fatalf("expected key 'operation', got %s", kv.Key)
	}
	if kv.Value.AsString() != "create" {
		t.Fatalf("expected 'create', got %s", kv.Value.AsString())
	}
}

func TestErrorAttributes(t *testing.T) {
	kv := ErrorAttributes(errors.New("oops"))
	if string(kv.Key) != "error.message" {
		t.Fatalf("expected key 'error.message', got %s", kv.Key)
	}
	if kv.Value.AsString() != "oops" {
		t.Fatalf("expected 'oops', got %s", kv.Value.AsString())
	}
}

func TestNoopSpan_Methods(t *testing.T) {
	s := &noopSpan{}
	s.End()
	s.AddEvent("test")
	if s.IsRecording() {
		t.Fatal("expected IsRecording to be false")
	}
	s.RecordError(errors.New("err"))
	if s.SpanContext().IsValid() {
		t.Fatal("expected invalid SpanContext")
	}
	s.SetStatus(0, "ok")
	s.SetName("span")
	s.SetAttributes(attribute.String("k", "v"))
	if s.TracerProvider() != nil {
		t.Fatal("expected nil TracerProvider")
	}
}
