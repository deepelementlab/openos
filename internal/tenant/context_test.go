package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTenantAndTenantFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-123")

	tenantID, ok := TenantFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "tenant-123", tenantID)
}

func TestTenantFromContext_Missing(t *testing.T) {
	tenantID, ok := TenantFromContext(context.Background())
	assert.False(t, ok)
	assert.Empty(t, tenantID)
}

func TestTenantFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), tenantContextKey, 12345)
	tenantID, ok := TenantFromContext(ctx)
	assert.False(t, ok)
	assert.Empty(t, tenantID)
}

func TestTenantFromContext_EmptyString(t *testing.T) {
	ctx := WithTenant(context.Background(), "")
	tenantID, ok := TenantFromContext(ctx)
	assert.True(t, ok)
	assert.Empty(t, tenantID)
}

func TestExtractTenantID_Success(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-456")
	tenantID, err := ExtractTenantID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tenant-456", tenantID)
}

func TestExtractTenantID_Missing(t *testing.T) {
	_, err := ExtractTenantID(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id not found")
}

func TestExtractTenantID_Empty(t *testing.T) {
	ctx := WithTenant(context.Background(), "")
	_, err := ExtractTenantID(ctx)
	assert.Error(t, err)
}

func TestRequireTenant_Present(t *testing.T) {
	called := false
	handler := RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	ctx := WithTenant(context.Background(), "tenant-1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireTenant_Missing(t *testing.T) {
	called := false
	handler := RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRequireTenant_Empty(t *testing.T) {
	called := false
	handler := RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := WithTenant(context.Background(), "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTenantMiddleware_FromHeader(t *testing.T) {
	called := false
	handler := TenantMiddleware("X-Tenant-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "header-tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantMiddleware_FromContext(t *testing.T) {
	called := false
	handler := TenantMiddleware("X-Tenant-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "ctx-tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	ctx := WithTenant(context.Background(), "ctx-tenant")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantMiddleware_HeaderOverridesContext(t *testing.T) {
	called := false
	handler := TenantMiddleware("X-Tenant-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "header-wins", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	ctx := WithTenant(context.Background(), "ctx-tenant")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil).WithContext(ctx)
	req.Header.Set("X-Tenant-ID", "header-wins")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
}

func TestTenantMiddleware_Missing(t *testing.T) {
	called := false
	handler := TenantMiddleware("X-Tenant-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTenantMiddleware_TrimsWhitespace(t *testing.T) {
	called := false
	handler := TenantMiddleware("X-Tenant-ID")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		tenantID, ok := TenantFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "tenant-1", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Tenant-ID", "  tenant-1  ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}
