package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestMiddleware() *AuthMiddleware {
	return NewAuthMiddleware(zap.NewNop(), "test-secret-key", nil, false)
}

func TestNewAuthMiddleware(t *testing.T) {
	m := NewAuthMiddleware(zap.NewNop(), "secret", nil, true)
	require.NotNil(t, m)
	assert.Equal(t, "secret", m.jwtSecret)
	assert.True(t, m.rbacEnabled)
	assert.Nil(t, m.redisClient)
}

func TestAuthMiddleware_PublicEndpoints(t *testing.T) {
	m := newTestMiddleware()

	tests := []struct {
		name string
		path string
	}{
		{name: "health", path: "/health"},
		{name: "ready", path: "/ready"},
		{name: "live", path: "/live"},
		{name: "metrics", path: "/metrics"},
		{name: "login", path: "/api/v1/auth/login"},
		{name: "register", path: "/api/v1/auth/register"},
		{name: "refresh", path: "/api/v1/auth/refresh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.True(t, called, "next handler should be called for public endpoint %s", tt.path)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	m := newTestMiddleware()
	called := false
	handler := m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	m := newTestMiddleware()

	called := false
	handler := m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	m := newTestMiddleware()

	token, err := m.GenerateToken("user-1", "alice", []string{"admin"}, []string{}, 1*time.Hour)
	require.NoError(t, err)

	called := false
	handler := m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_ValidateToken(t *testing.T) {
	m := newTestMiddleware()

	token, err := m.GenerateToken("user-1", "alice", []string{"admin"}, []string{}, 1*time.Hour)
	require.NoError(t, err)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, []string{"admin"}, claims.Roles)
}

func TestAuthMiddleware_ValidateToken_Expired(t *testing.T) {
	m := newTestMiddleware()

	token, err := m.GenerateToken("user-1", "alice", []string{"admin"}, []string{}, 1*time.Nanosecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateToken(token)
	require.Error(t, err)
}

func TestAuthMiddleware_ValidateToken_WrongSecret(t *testing.T) {
	m := newTestMiddleware()
	m2 := NewAuthMiddleware(zap.NewNop(), "different-secret", nil, false)

	token, err := m.GenerateToken("user-1", "alice", []string{"admin"}, []string{}, 1*time.Hour)
	require.NoError(t, err)

	_, err = m2.ValidateToken(token)
	require.Error(t, err)
}

func TestAuthMiddleware_GenerateToken(t *testing.T) {
	m := newTestMiddleware()

	token, err := m.GenerateToken("uid-42", "bob", []string{"user"}, []string{"read"}, 2*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "uid-42", claims.UserID)
	assert.Equal(t, "bob", claims.Username)
	assert.Equal(t, []string{"user"}, claims.Roles)
	assert.Equal(t, []string{"read"}, claims.Scopes)
}

func TestAuthMiddleware_Authorize_RBACDisabled(t *testing.T) {
	m := newTestMiddleware()

	called := false
	handler := m.Authorize([]string{"admin"}, []string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_Authorize_RBACEnabled_SufficientRoles(t *testing.T) {
	m := NewAuthMiddleware(zap.NewNop(), "secret", nil, true)

	called := false
	handler := m.Authorize([]string{"admin"}, []string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), "roles", []string{"admin", "user"})
	ctx = context.WithValue(ctx, "scopes", []string{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_Authorize_RBACEnabled_InsufficientRoles(t *testing.T) {
	m := NewAuthMiddleware(zap.NewNop(), "secret", nil, true)

	called := false
	handler := m.Authorize([]string{"admin"}, []string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), "roles", []string{"viewer"})
	ctx = context.WithValue(ctx, "scopes", []string{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuthMiddleware_Authorize_ScopeCheck(t *testing.T) {
	m := NewAuthMiddleware(zap.NewNop(), "secret", nil, true)

	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		wantStatus     int
	}{
		{name: "has required scope", userScopes: []string{"read", "write"}, requiredScopes: []string{"write"}, wantStatus: http.StatusOK},
		{name: "missing required scope", userScopes: []string{"read"}, requiredScopes: []string{"write"}, wantStatus: http.StatusForbidden},
		{name: "no scopes", userScopes: []string{}, requiredScopes: []string{"read"}, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := m.Authorize([]string{}, tt.requiredScopes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			ctx := context.WithValue(context.Background(), "roles", []string{})
			ctx = context.WithValue(ctx, "scopes", tt.userScopes)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestAuthMiddleware_isPublicEndpoint(t *testing.T) {
	m := newTestMiddleware()

	tests := []struct {
		path   string
		expect bool
	}{
		{"/health", true},
		{"/ready", true},
		{"/live", true},
		{"/metrics", true},
		{"/api/v1/auth/login", true},
		{"/api/v1/agents", false},
		{"/unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expect, m.isPublicEndpoint(tt.path))
		})
	}
}

func TestAuthMiddleware_isTokenBlacklisted_NilRedis(t *testing.T) {
	m := newTestMiddleware()
	assert.False(t, m.isTokenBlacklisted(context.Background(), "some-token"))
}
