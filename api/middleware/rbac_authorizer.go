package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/agentos/aos/internal/security"
	"go.uber.org/zap"
)

// RBACAuthorizer implements RBAC-based authorization middleware
type RBACAuthorizer struct {
	logger           *zap.Logger
	policyEvaluator  security.PolicyEvaluator
	resourceResolver ResourceResolver
	auditLogger      AuditLogger
}

// ResourceResolver resolves HTTP requests to security resources and actions
type ResourceResolver interface {
	ResolveResource(r *http.Request) (resource, action string, err error)
}

// DefaultResourceResolver provides default resource-action mapping
type DefaultResourceResolver struct {
	rules []ResourceRule
}

// ResourceRule defines mapping from HTTP path/method to security resource and action
type ResourceRule struct {
	PathPattern string
	Methods     []string
	Resource    string
	Action      string
}

// NewRBACAuthorizer creates a new RBAC authorizer
func NewRBACAuthorizer(logger *zap.Logger, policyEvaluator security.PolicyEvaluator) *RBACAuthorizer {
	resolver := &DefaultResourceResolver{
		rules: []ResourceRule{
			// Agent operations
			{PathPattern: "/api/v1/agents", Methods: []string{"GET"}, Resource: "agents", Action: "list"},
			{PathPattern: "/api/v1/agents", Methods: []string{"POST"}, Resource: "agents", Action: "create"},
			{PathPattern: "/api/v1/agents/*", Methods: []string{"GET"}, Resource: "agents", Action: "read"},
			{PathPattern: "/api/v1/agents/*", Methods: []string{"PUT", "PATCH"}, Resource: "agents", Action: "update"},
			{PathPattern: "/api/v1/agents/*", Methods: []string{"DELETE"}, Resource: "agents", Action: "delete"},
			{PathPattern: "/api/v1/agents/*/start", Methods: []string{"POST"}, Resource: "agents", Action: "start"},
			{PathPattern: "/api/v1/agents/*/stop", Methods: []string{"POST"}, Resource: "agents", Action: "stop"},
			{PathPattern: "/api/v1/agents/*/restart", Methods: []string{"POST"}, Resource: "agents", Action: "restart"},
			
			// Monitoring operations (read-only for most users)
			{PathPattern: "/health", Methods: []string{"GET"}, Resource: "system", Action: "read"},
			{PathPattern: "/ready", Methods: []string{"GET"}, Resource: "system", Action: "read"},
			{PathPattern: "/live", Methods: []string{"GET"}, Resource: "system", Action: "read"},
			{PathPattern: "/metrics", Methods: []string{"GET"}, Resource: "metrics", Action: "read"},
		},
	}
	
	return &RBACAuthorizer{
		logger:           logger,
		policyEvaluator:  policyEvaluator,
		resourceResolver: resolver,
	}
}

// WithAuditLogger sets audit logger for authorization events
func (a *RBACAuthorizer) WithAuditLogger(auditLogger AuditLogger) *RBACAuthorizer {
	a.auditLogger = auditLogger
	return a
}

// logAuthorizationSuccess logs successful authorization event
func (a *RBACAuthorizer) logAuthorizationSuccess(ctx context.Context, userID string, roles []string, resource, action, reason string) {
	if a.auditLogger != nil {
		a.auditLogger.LogAuthorization(ctx, true, userID, resource, action, reason)
	}
}

// logAuthorizationFailure logs failed authorization event
func (a *RBACAuthorizer) logAuthorizationFailure(ctx context.Context, userID string, roles []string, resource, action, reason string) {
	if a.auditLogger != nil {
		a.auditLogger.LogAuthorization(ctx, false, userID, resource, action, reason)
	}
}

// WithResourceResolver sets custom resource resolver
func (a *RBACAuthorizer) WithResourceResolver(resolver ResourceResolver) *RBACAuthorizer {
	a.resourceResolver = resolver
	return a
}

// Middleware creates authorization middleware
func (a *RBACAuthorizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authorization for public endpoints
		if a.isPublicEndpoint(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract user info from context
		ctx := r.Context()
		userID, _ := ctx.Value("user_id").(string)
		roles, _ := ctx.Value("roles").([]string)
		
		if userID == "" {
			a.logger.Error("user_id not found in context for authorization")
			a.writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
			return
		}

		// Resolve resource and action from request
		resource, action, err := a.resourceResolver.ResolveResource(r)
		if err != nil {
			a.logger.Warn("failed to resolve resource", zap.Error(err))
			a.writeError(w, http.StatusForbidden, "RESOURCE_RESOLVE_FAILED", "Cannot resolve resource for authorization")
			return
		}

		// Evaluate policy
		evalReq := security.EvalRequest{
			UserID:   userID,
			Roles:    roles,
			Resource: resource,
			Action:   action,
		}

		result, err := a.policyEvaluator.Evaluate(ctx, evalReq)
		if err != nil {
			a.logger.Error("policy evaluation failed", zap.Error(err))
			a.writeError(w, http.StatusInternalServerError, "POLICY_EVAL_FAILED", "Authorization system error")
			return
		}

		if !result.Allowed {
			// Log authorization failure
			a.logAuthorizationFailure(r.Context(), userID, roles, resource, action, result.Reason)
			
			a.logger.Debug("authorization denied",
				zap.String("user_id", userID),
				zap.Strings("roles", roles),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.String("reason", result.Reason))
			
			a.writeError(w, http.StatusForbidden, "FORBIDDEN", result.Reason)
			return
		}

		// Log authorization success
			a.logAuthorizationSuccess(r.Context(), userID, roles, resource, action, result.Reason)

		// Add authorization context
		ctx = context.WithValue(ctx, "authorized_resource", resource)
		ctx = context.WithValue(ctx, "authorized_action", action)
		ctx = context.WithValue(ctx, "authorization_result", result)

		// Continue with authorized request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission creates a middleware that requires specific permission
func (a *RBACAuthorizer) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID, _ := ctx.Value("user_id").(string)
			roles, _ := ctx.Value("roles").([]string)
			
			if userID == "" {
				a.writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
				return
			}

			evalReq := security.EvalRequest{
				UserID:   userID,
				Roles:    roles,
				Resource: resource,
				Action:   action,
			}

			result, err := a.policyEvaluator.Evaluate(ctx, evalReq)
			if err != nil {
				a.logger.Error("policy evaluation failed", zap.Error(err))
				a.writeError(w, http.StatusInternalServerError, "POLICY_EVAL_FAILED", "Authorization system error")
				return
			}

			if !result.Allowed {
				a.writeError(w, http.StatusForbidden, "FORBIDDEN", result.Reason)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResolveResource implements ResourceResolver interface
func (r *DefaultResourceResolver) ResolveResource(req *http.Request) (resource, action string, err error) {
	path := req.URL.Path
	method := req.Method
	
	for _, rule := range r.rules {
		if !r.pathMatches(rule.PathPattern, path) {
			continue
		}
		
		if !r.methodMatches(rule.Methods, method) {
			continue
		}
		
		return rule.Resource, rule.Action, nil
	}
	
	return "", "", &ResourceResolutionError{
		Path:   path,
		Method: method,
		Reason: "No matching resource rule found",
	}
}

// pathMatches checks if request path matches pattern
func (r *DefaultResourceResolver) pathMatches(pattern, path string) bool {
	if pattern == path {
		return true
	}
	
	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if strings.HasPrefix(path, prefix+"/") {
			remaining := strings.TrimPrefix(path, prefix+"/")
			if remaining == "" || !strings.Contains(remaining, "/") {
				return true
			}
		}
	}
	
	return false
}

// methodMatches checks if request method matches allowed methods
func (r *DefaultResourceResolver) methodMatches(allowed []string, method string) bool {
	for _, m := range allowed {
		if m == method {
			return true
		}
		if m == "*" {
			return true
		}
	}
	return false
}

// isPublicEndpoint checks if endpoint should bypass authorization
func (a *RBACAuthorizer) isPublicEndpoint(r *http.Request) bool {
	path := r.URL.Path
	
	publicPaths := []string{
		"/health",
		"/ready",
		"/live",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
	}
	
	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}
	
	return false
}

// writeError writes an error response
func (a *RBACAuthorizer) writeError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	errorResponse := map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    errCode,
			"message": msg,
		},
	}
	
	json.NewEncoder(w).Encode(errorResponse)
}

// ResourceResolutionError represents resource resolution failure
type ResourceResolutionError struct {
	Path   string
	Method string
	Reason string
}

func (e *ResourceResolutionError) Error() string {
	return e.Reason
}

// AuthorizationEvent represents an authorization audit event
type AuthorizationEvent struct {
	UserID   string
	Roles    []string
	Resource string
	Action   string
	Reason   string
	Success  bool
	Timestamp string
}