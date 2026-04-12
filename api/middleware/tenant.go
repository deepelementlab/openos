package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/agentos/aos/api/models"
	"github.com/agentos/aos/internal/tenant"
	"go.uber.org/zap"
)

type TenantMiddleware struct {
	logger *zap.Logger
}

func NewTenantMiddleware(logger *zap.Logger) *TenantMiddleware {
	return &TenantMiddleware{
		logger: logger,
	}
}

func (m *TenantMiddleware) ExtractTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isHealthEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		tenantID := m.extractFromHeader(r)
		if tenantID == "" {
			tenantID, _ = tenant.TenantFromContext(r.Context())
		}

		if tenantID == "" {
			m.sendError(w, http.StatusBadRequest, "TENANT_ID_MISSING", "X-Tenant-ID header or tenant_id in token is required")
			return
		}

		ctx := tenant.WithTenant(r.Context(), tenantID)
		m.logger.Debug("tenant resolved", zap.String("tenant_id", tenantID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *TenantMiddleware) extractFromHeader(r *http.Request) string {
	tenantID := r.Header.Get("X-Tenant-ID")
	return strings.TrimSpace(tenantID)
}

func (m *TenantMiddleware) isHealthEndpoint(path string) bool {
	healthPaths := []string{"/health", "/ready", "/live", "/metrics"}
	for _, p := range healthPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func (m *TenantMiddleware) sendError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := models.ErrorResponse(code, message)
	jsonData, _ := json.Marshal(response)
	w.Write(jsonData)
}

func RequireTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := tenant.TenantFromContext(r.Context())
		if !ok || tenantID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			jsonData, _ := json.Marshal(models.ErrorResponse("TENANT_ID_REQUIRED", "tenant_id is required in context"))
			w.Write(jsonData)
			return
		}
		ctx := context.WithValue(r.Context(), "tenant_id", tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
