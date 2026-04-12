package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("client-1"), "request %d should be allowed", i+1)
	}
	assert.False(t, rl.Allow("client-1"), "6th request should be rate limited")
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(10, 2)

	assert.True(t, rl.Allow("client-1"))
	assert.True(t, rl.Allow("client-1"))
	assert.False(t, rl.Allow("client-1"))

	assert.True(t, rl.Allow("client-2"))
	assert.True(t, rl.Allow("client-2"))
	assert.False(t, rl.Allow("client-2"))
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	rl.Allow("old-client")

	time.Sleep(10 * time.Millisecond)
	rl.Cleanup(5 * time.Millisecond)

	rl.mu.Lock()
	_, exists := rl.buckets["old-client"]
	rl.mu.Unlock()
	assert.False(t, exists, "old client should be cleaned up")
}

func TestRateLimiter_Middleware_Allowed(t *testing.T) {
	rl := NewRateLimiter(100, 100)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Middleware_RateLimited(t *testing.T) {
	rl := NewRateLimiter(1, 2)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCircuitBreaker_Closed(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	assert.Equal(t, CircuitClosed, cb.State())
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())
	assert.False(t, cb.Allow())
}

func TestCircuitBreaker_HalfOpenAfterReset(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	time.Sleep(60 * time.Millisecond)
	assert.True(t, cb.Allow())
	assert.Equal(t, CircuitHalfOpen, cb.State())
}

func TestCircuitBreaker_ClosesAfterSuccesses(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())
}

func TestCircuitBreaker_Failures(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Second)
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, 2, cb.Failures())
}
