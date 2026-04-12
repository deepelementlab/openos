package middleware

import (
	"time"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)
func parseTime() time.Time {
	return time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
}

func TestZapAuditLogger_LogEvent_NoneLevel(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelNone)

	event := AuditEvent{
		ID:        "evt-1",
		Type:      EventTypeAuthentication,
		Timestamp: parseTime(),
		Success:   false,
	}
	// Should not error even at None level
	if err := al.LogEvent(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_LogAuthentication(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelMinimal)

	err := al.LogAuthentication(context.Background(), false, "user-1", "alice", "bad password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_LogAuthentication_SuccessAtMinimal(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelMinimal)

	// At Minimal level, success=true auth events are filtered out but should not error
	err := al.LogAuthentication(context.Background(), true, "user-1", "alice", "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_LogAuthorization(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelNormal)

	err := al.LogAuthorization(context.Background(), true, "user-1", "agents", "read", "role match")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_LogHTTPRequest(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelVerbose)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents?foo=bar", nil)
	ctx := context.Background()

	err := al.LogHTTPRequest(ctx, req, 200, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_SetGetLevel(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelMinimal)

	if al.GetLevel() != AuditLevelMinimal {
		t.Fatalf("expected Minimal, got %d", al.GetLevel())
	}

	al.SetLevel(AuditLevelVerbose)
	if al.GetLevel() != AuditLevelVerbose {
		t.Fatalf("expected Verbose, got %d", al.GetLevel())
	}
}

func TestZapAuditLogger_ShouldLogEvent(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		level     AuditLevel
		eventType AuditEventType
		success   bool
		expected  bool
	}{
		{AuditLevelNone, EventTypeAuthentication, false, false},
		{AuditLevelMinimal, EventTypeAuthentication, false, true},
		{AuditLevelMinimal, EventTypeAuthentication, true, false},
		{AuditLevelMinimal, EventTypeAuthorization, false, false},
		{AuditLevelNormal, EventTypeAuthentication, true, true},
		{AuditLevelNormal, EventTypeAuthorization, true, true},
		{AuditLevelNormal, EventTypeUserOperation, false, true},
		{AuditLevelVerbose, EventTypeSystemOperation, true, true},
		{AuditLevelVerbose, EventTypeSecurityViolation, false, true},
	}

	for _, tt := range tests {
		al := NewZapAuditLogger(logger, tt.level)
		result := al.shouldLogEvent(tt.eventType, tt.success)
		if result != tt.expected {
			t.Errorf("level=%d, type=%s, success=%v: expected %v, got %v",
				tt.level, tt.eventType, tt.success, tt.expected, result)
		}
	}
}

func TestZapAuditLogger_LogAuthWithContext(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelMinimal)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	ctx := context.WithValue(context.Background(), "http_request", req)
	ctx = context.WithValue(ctx, "session_id", "sess-123")

	err := al.LogAuthentication(ctx, false, "", "bob", "invalid credentials")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZapAuditLogger_LogAuthWithContextUsername(t *testing.T) {
	logger := zap.NewNop()
	al := NewZapAuditLogger(logger, AuditLevelNormal)

	ctx := context.WithValue(context.Background(), "username", "alice")
	ctx = context.WithValue(ctx, "tenant_id", "tenant-1")

	err := al.LogAuthorization(ctx, true, "user-1", "agents", "read", "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetIPAddress_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	ip := getIPAddress(req)
	if ip != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %s", ip)
	}
}

func TestGetIPAddress_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")

	ip := getIPAddress(req)
	if ip != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %s", ip)
	}
}

func TestGetIPAddress_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "172.16.0.1:12345"

	ip := getIPAddress(req)
	if ip != "172.16.0.1" {
		t.Fatalf("expected 172.16.0.1, got %s", ip)
	}
}

func TestIsSensitiveQueryParam(t *testing.T) {
	sensitive := []string{"password", "token", "secret", "key", "auth", "credential", "jwt", "api_key"}
	for _, key := range sensitive {
		if !isSensitiveQueryParam(key) {
			t.Fatalf("expected %q to be sensitive", key)
		}
		if !isSensitiveQueryParam("my_" + key + "_value") {
			t.Fatalf("expected %q containing %q to be sensitive", "my_"+key+"_value", key)
		}
	}

	nonSensitive := []string{"page", "limit", "sort", "filter"}
	for _, key := range nonSensitive {
		if isSensitiveQueryParam(key) {
			t.Fatalf("expected %q to NOT be sensitive", key)
		}
	}
}

func TestNullAuditLogger(t *testing.T) {
	al := &NullAuditLogger{}
	ctx := context.Background()

	if err := al.LogEvent(ctx, AuditEvent{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := al.LogAuthentication(ctx, true, "u", "n", "r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := al.LogAuthorization(ctx, true, "u", "r", "a", "reason"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := al.LogHTTPRequest(ctx, req, 200, "u"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	al.SetLevel(AuditLevelVerbose)
	if al.GetLevel() != AuditLevelNone {
		t.Fatalf("NullAuditLogger should always return None level")
	}
}

func TestNewAuditMiddleware(t *testing.T) {
	al := NewZapAuditLogger(zap.NewNop(), AuditLevelVerbose)
	mw := NewAuditMiddleware(al, AuditLevelVerbose)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("inner handler should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewAuditMiddleware_CapturesStatus(t *testing.T) {
	al := NewZapAuditLogger(zap.NewNop(), AuditLevelVerbose)
	mw := NewAuditMiddleware(al, AuditLevelVerbose)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
