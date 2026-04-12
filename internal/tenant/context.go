package tenant

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const tenantContextKey contextKey = "tenant_id"

func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

func TenantFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantContextKey).(string)
	return tenantID, ok
}

func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok || tenantID == "" {
			http.Error(w, `{"error":"tenant_id is required"}`, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TenantMiddleware(tenantHeader string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := strings.TrimSpace(r.Header.Get(tenantHeader))
			if tenantID == "" {
				tenantID, _ = TenantFromContext(r.Context())
			}
			if tenantID == "" {
				http.Error(w, `{"error":"tenant_id is required"}`, http.StatusBadRequest)
				return
			}
			ctx := WithTenant(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ExtractTenantID(ctx context.Context) (string, error) {
	tenantID, ok := TenantFromContext(ctx)
	if !ok || tenantID == "" {
		return "", fmt.Errorf("tenant_id not found in context")
	}
	return tenantID, nil
}
