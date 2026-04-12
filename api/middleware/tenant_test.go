package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestTenantMiddleware() *TenantMiddleware {
	return NewTenantMiddleware(zap.NewNop())
}

func TestNewTenantMiddleware(t *testing.T) {
	m := NewTenantMiddleware(zap.NewNop())
	require.NotNil(t, m)
}

func TestTenantMiddleware_ExtractTenant_FromHeader(t *testing.T) {
	m := newTestTenantMiddleware()
	called := false
	handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := tenant.TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "tenant-abc", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Tenant-ID", "tenant-abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantMiddleware_ExtractTenant_FromContext(t *testing.T) {
	m := newTestTenantMiddleware()
	called := false
	handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := tenant.TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "ctx-tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	ctx := tenant.WithTenant(context.Background(), "ctx-tenant")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantMiddleware_ExtractTenant_Missing(t *testing.T) {
	m := newTestTenantMiddleware()
	called := false
	handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.Equal(t, "TENANT_ID_MISSING", resp.Error.Code)
}

func TestTenantMiddleware_HealthEndpoints(t *testing.T) {
	m := newTestTenantMiddleware()

	healthPaths := []string{"/health", "/ready", "/live", "/metrics"}
	for _, path := range healthPaths {
		t.Run(path, func(t *testing.T) {
			called := false
			handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.True(t, called)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestTenantMiddleware_EmptyTenantID(t *testing.T) {
	m := newTestTenantMiddleware()
	called := false
	handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Tenant-ID", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRequireTenantContext_Present(t *testing.T) {
	called := false
	handler := RequireTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "tenant-1", r.Context().Value("tenant_id"))
		w.WriteHeader(http.StatusOK)
	}))

	ctx := tenant.WithTenant(context.Background(), "tenant-1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireTenantContext_Missing(t *testing.T) {
	called := false
	handler := RequireTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "TENANT_ID_REQUIRED", resp.Error.Code)
}

func TestRequireTenantContext_Empty(t *testing.T) {
	called := false
	handler := RequireTenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := tenant.WithTenant(context.Background(), "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTenantMiddleware_HeaderOverridesContext(t *testing.T) {
	m := newTestTenantMiddleware()
	called := false
	handler := m.ExtractTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, _ := tenant.TenantFromContext(r.Context())
		assert.Equal(t, "header-tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	ctx := tenant.WithTenant(context.Background(), "ctx-tenant")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}
