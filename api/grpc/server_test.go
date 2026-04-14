package grpc

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewServer(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultServerConfig()
	config.Port = 0 // Use any available port

	server, err := NewServer(config, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer server.Stop()

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	if server.grpcServer == nil {
		t.Fatal("expected gRPC server to be initialized")
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewRecoveryInterceptor(logger)

	// Test unary interceptor
	unary := interceptor.UnaryInterceptor()

	// Test with panicking handler
	panicHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	resp, err := unary(context.Background(), nil, nil, panicHandler)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	if resp != nil {
		t.Fatal("expected nil response from panic")
	}

	// Test stream interceptor
	stream := interceptor.StreamInterceptor()
	
	panicStreamHandler := func(srv interface{}, stream grpc.ServerStream) error {
		panic("test panic")
	}

	err = stream(nil, nil, nil, panicStreamHandler)
	if err != nil {
		t.Fatalf("stream interceptor should swallow panic errors, got: %v", err)
	}
}

func TestLoggingInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewLoggingInterceptor(logger)

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "response" {
		t.Fatalf("expected 'response', got %v", resp)
	}
}

func TestAuthInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewAuthInterceptor(logger).WithEnabled(false)

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error when auth disabled: %v", err)
	}
	if resp != "response" {
		t.Fatalf("expected 'response', got %v", resp)
	}
}

func TestAuthInterceptorEnabled(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewAuthInterceptor(logger).WithEnabled(true)

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	// Test without auth header (should fail)
	_, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Test"}, handler)
	if err == nil {
		t.Fatal("expected error without auth header")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated error, got: %v", err)
	}
}

func TestTenantInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewTenantInterceptor(logger).WithEnabled(false)

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error when tenant disabled: %v", err)
	}
	if resp != "response" {
		t.Fatalf("expected 'response', got %v", resp)
	}
}

func TestMetricsInterceptor(t *testing.T) {
	interceptor := NewMetricsInterceptor()

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/aos.api.v1.AgentService/GetAgent"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "response" {
		t.Fatalf("expected 'response', got %v", resp)
	}
}

func TestQuotaInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewQuotaInterceptor(logger).WithEnabled(false)

	unary := interceptor.UnaryInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := unary(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/aos.api.v1.AgentService/CreateAgent"}, handler)
	if err != nil {
		t.Fatalf("unexpected error when quota disabled: %v", err)
	}
	if resp != "response" {
		t.Fatalf("expected 'response', got %v", resp)
	}
}

func TestInterceptorChain(t *testing.T) {
	logger := zap.NewNop()
	chain := NewInterceptorChain(logger)

	recovery := NewRecoveryInterceptor(logger)
	logging := NewLoggingInterceptor(logger)

	chain.Add(recovery)
	chain.Add(logging)

	unaryInterceptors := chain.UnaryInterceptors()
	if len(unaryInterceptors) != 2 {
		t.Fatalf("expected 2 unary interceptors, got %d", len(unaryInterceptors))
	}

	streamInterceptors := chain.StreamInterceptors()
	if len(streamInterceptors) != 2 {
		t.Fatalf("expected 2 stream interceptors, got %d", len(streamInterceptors))
	}
}

func TestIsPublicMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"/grpc.health.v1.Health/Check", true},
		{"/grpc.health.v1.Health/Watch", true},
		{"/aos.api.v1.MonitoringService/HealthCheck", true},
		{"/aos.api.v1.AgentService/GetAgent", false},
		{"/aos.api.v1.AgentService/CreateAgent", false},
	}

	for _, test := range tests {
		result := isPublicMethod(test.method)
		if result != test.expected {
			t.Errorf("isPublicMethod(%q) = %v, expected %v", test.method, result, test.expected)
		}
	}
}
