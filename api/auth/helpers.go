package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	ctxUserID   contextKey = "user_id"
	ctxUsername  contextKey = "username"
	ctxRoles    contextKey = "roles"
)

// SetUserContext stores authentication details in the request context.
func SetUserContext(ctx context.Context, userID, username string, roles []string) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxUsername, username)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	return ctx
}

// ExtractUserID returns the authenticated user ID from the context.
func ExtractUserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok
}

// ExtractUsername returns the authenticated username from the context.
func ExtractUsername(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUsername).(string)
	return v, ok
}

// ExtractRoles returns the authenticated user's roles from the context.
func ExtractRoles(ctx context.Context) []string {
	v, _ := ctx.Value(ctxRoles).([]string)
	return v
}

// RequireRole is an HTTP middleware that enforces role-based access.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := ExtractRoles(r.Context())
			for _, ur := range userRoles {
				if roleSet[ur] {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}
