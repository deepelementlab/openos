package server

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     int
	capacity int
}

type tokenBucket struct {
	tokens    int
	lastCheck time.Time
}

func NewRateLimiter(rate, capacity int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &tokenBucket{
			tokens:    rl.capacity - 1,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(bucket.lastCheck)
	refill := int(elapsed.Seconds()) * rl.rate
	bucket.tokens += refill
	if bucket.tokens > rl.capacity {
		bucket.tokens = rl.capacity
	}
	bucket.lastCheck = now

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		if now.Sub(bucket.lastCheck) > maxAge {
			delete(rl.buckets, key)
		}
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if !rl.Allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

type CircuitBreaker struct {
	mu          sync.Mutex
	state       CircuitState
	failures    int
	successes   int
	maxFailures int
	resetAfter  time.Duration
	lastFailure time.Time
	halfOpenMax int
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func NewCircuitBreaker(maxFailures int, resetAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       CircuitClosed,
		maxFailures: maxFailures,
		resetAfter:  resetAfter,
		halfOpenMax: 3,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.resetAfter {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.successes < cb.halfOpenMax
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	if cb.state == CircuitHalfOpen && cb.successes >= cb.halfOpenMax {
		cb.state = CircuitClosed
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()
	if cb.state == CircuitHalfOpen || cb.failures >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}
