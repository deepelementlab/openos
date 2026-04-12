package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockPolicyEvaluator implements security.PolicyEvaluator for testing
type mockPolicyEvaluator struct {
	result *security.EvalResult
	err    error
}

func (m *mockPolicyEvaluator) Evaluate(_ context.Context, _ security.EvalRequest) (*security.EvalResult, error) {
	return m.result, m.err
}

func (m *mockPolicyEvaluator) AddRule(_ security.PolicyRule) error  { return nil }
func (m *mockPolicyEvaluator) RemoveRule(_ string) error           { return nil }
func (m *mockPolicyEvaluator) ListRules() []security.PolicyRule    { return nil }

func TestRBACAuthorizer_NewRBACAuthorizer(t *testing.T) {
	pe := &mockPolicyEvaluator{result: &security.EvalResult{}}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	require.NotNil(t, a)
	assert.Equal(t, zap.NewNop(), a.logger)
	assert.NotNil(t, a.resourceResolver)
}

func TestRBACAuthorizer_PublicEndpoints(t *testing.T) {
	pe := &mockPolicyEvaluator{result: &security.EvalResult{}}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	publicPaths := []string{"/health", "/ready", "/live", "/metrics", "/api/v1/auth/login", "/api/v1/auth/refresh"}

	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			called := false
			handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.True(t, called, "next handler should be called for %s", path)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRBACAuthorizer_MissingUserID(t *testing.T) {
	pe := &mockPolicyEvaluator{result: &security.EvalResult{}}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "AUTH_REQUIRED", errObj["code"])
}

func TestRBACAuthorizer_Allowed(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: true, Reason: "role match"},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	called := false
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"admin"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRBACAuthorizer_Denied(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: false, Reason: "insufficient permissions"},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	called := false
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"viewer"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errObj["code"])
}

func TestRBACAuthorizer_PolicyEvalError(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{},
		err:    assert.AnError,
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"admin"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRBACAuthorizer_RequirePermission_Allowed(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: true, Reason: "ok"},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	called := false
	handler := a.RequirePermission("agents", "create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"admin"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRBACAuthorizer_RequirePermission_Denied(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: false, Reason: "no permission"},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	called := false
	handler := a.RequirePermission("agents", "delete")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"viewer"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/123", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRBACAuthorizer_RequirePermission_NoUserID(t *testing.T) {
	pe := &mockPolicyEvaluator{result: &security.EvalResult{}}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	handler := a.RequirePermission("agents", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDefaultResourceResolver_AgentList(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "agents", resource)
	assert.Equal(t, "list", action)
}

func TestDefaultResourceResolver_AgentCreate(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "agents", resource)
	assert.Equal(t, "create", action)
}

func TestDefaultResourceResolver_AgentRead(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent-123", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "agents", resource)
	assert.Equal(t, "read", action)
}

func TestDefaultResourceResolver_AgentUpdate(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/agent-123", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "agents", resource)
	assert.Equal(t, "update", action)
}

func TestDefaultResourceResolver_AgentDelete(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/agent-123", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "agents", resource)
	assert.Equal(t, "delete", action)
}

func TestDefaultResourceResolver_AgentWildcardNoSubpath(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	// /api/v1/agents/* should not match paths with sub-segments
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/agent-123/start", nil)
	_, _, err := resolver.ResolveResource(req)
	// Multi-segment wildcard not supported by current pathMatches implementation
	require.Error(t, err)
}

func TestDefaultResourceResolver_HealthSystem(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "system", resource)
	assert.Equal(t, "read", action)
}

func TestDefaultResourceResolver_Metrics(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resource, action, err := resolver.ResolveResource(req)
	require.NoError(t, err)
	assert.Equal(t, "metrics", resource)
	assert.Equal(t, "read", action)
}

func TestDefaultResourceResolver_NoMatch(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
	_, _, err := resolver.ResolveResource(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No matching resource rule")
}

func TestDefaultResourceResolver_MethodNotMatched(t *testing.T) {
	resolver := &DefaultResourceResolver{rules: defaultTestRules()}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/agents", nil)
	_, _, err := resolver.ResolveResource(req)
	require.Error(t, err)
}

func TestResourceResolutionError(t *testing.T) {
	err := &ResourceResolutionError{
		Path:   "/test",
		Method: "PATCH",
		Reason: "No matching resource rule found",
	}
	assert.Equal(t, "No matching resource rule found", err.Error())
}

func TestRBACAuthorizer_WithAuditLogger(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: true, Reason: "ok"},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	result := a.WithAuditLogger(nil)
	assert.NotNil(t, result)
}

func TestRBACAuthorizer_WithResourceResolver(t *testing.T) {
	pe := &mockPolicyEvaluator{result: &security.EvalResult{}}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	custom := &DefaultResourceResolver{rules: defaultTestRules()}
	result := a.WithResourceResolver(custom)
	assert.NotNil(t, result)
}

func TestRBACAuthorizer_ResourceResolveFailed(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{Allowed: true},
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"admin"})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/unknown", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRBACAuthorizer_RequirePermission_EvalError(t *testing.T) {
	pe := &mockPolicyEvaluator{
		result: &security.EvalResult{},
		err:    assert.AnError,
	}
	a := NewRBACAuthorizer(zap.NewNop(), pe)

	handler := a.RequirePermission("agents", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach")
	}))

	ctx := context.WithValue(context.Background(), "user_id", "user-1")
	ctx = context.WithValue(ctx, "roles", []string{"admin"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// defaultTestRules returns the default resource rules for testing
func defaultTestRules() []ResourceRule {
	return []ResourceRule{
		{PathPattern: "/api/v1/agents", Methods: []string{"GET"}, Resource: "agents", Action: "list"},
		{PathPattern: "/api/v1/agents", Methods: []string{"POST"}, Resource: "agents", Action: "create"},
		{PathPattern: "/api/v1/agents/*", Methods: []string{"GET"}, Resource: "agents", Action: "read"},
		{PathPattern: "/api/v1/agents/*", Methods: []string{"PUT", "PATCH"}, Resource: "agents", Action: "update"},
		{PathPattern: "/api/v1/agents/*", Methods: []string{"DELETE"}, Resource: "agents", Action: "delete"},
		{PathPattern: "/health", Methods: []string{"GET"}, Resource: "system", Action: "read"},
		{PathPattern: "/ready", Methods: []string{"GET"}, Resource: "system", Action: "read"},
		{PathPattern: "/live", Methods: []string{"GET"}, Resource: "system", Action: "read"},
		{PathPattern: "/metrics", Methods: []string{"GET"}, Resource: "metrics", Action: "read"},
	}
}
