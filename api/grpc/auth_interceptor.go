package grpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor handles authentication
type AuthInterceptor struct {
	logger    *zap.Logger
	jwtSecret string
	enabled   bool
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(logger *zap.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		logger:    logger,
		jwtSecret: "change-this-in-production",
		enabled:   true,
	}
}

// WithJWTSecret sets the JWT secret
func (a *AuthInterceptor) WithJWTSecret(secret string) *AuthInterceptor {
	a.jwtSecret = secret
	return a
}

// WithEnabled enables/disables authentication
func (a *AuthInterceptor) WithEnabled(enabled bool) *AuthInterceptor {
	a.enabled = enabled
	return a
}

// UnaryInterceptor returns the unary interceptor
func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !a.enabled {
			return handler(ctx, req)
		}

		// Skip auth for health checks
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract and validate token
		claims, err := a.extractAndValidateToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		// Add claims to context
		ctx = WithAuthClaims(ctx, claims)

		return handler(ctx, req)
	}
}

// StreamInterceptor returns the stream interceptor
func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !a.enabled {
			return handler(srv, ss)
		}

		// Skip auth for public methods
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// Extract and validate token
		claims, err := a.extractAndValidateToken(ss.Context())
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		// Wrap stream with auth context
		wrapped := &authWrappedStream{
			ServerStream: ss,
			ctx:          WithAuthClaims(ss.Context(), claims),
		}

		return handler(srv, wrapped)
	}
}

// isPublicMethod checks if the method is public (no auth required)
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
		"/aos.api.v1.MonitoringService/HealthCheck",
	}
	for _, m := range publicMethods {
		if method == m {
			return true
		}
	}
	return false
}

// extractAndValidateToken extracts and validates JWT token from context
func (a *AuthInterceptor) extractAndValidateToken(ctx context.Context) (*AuthClaims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	// Try to get token from authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	tokenString := authHeaders[0]
	// Remove "Bearer " prefix
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimPrefix(tokenString, "bearer ")

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &AuthClaims{
		UserID: claims["sub"].(string),
		Email:  getStringClaim(claims, "email"),
		Roles:  getStringSliceClaim(claims, "roles"),
	}, nil
}

// AuthClaims contains authentication claims
type AuthClaims struct {
	UserID   string
	Email    string
	Roles    []string
	TenantID string
}

type contextKey string

const authClaimsContextKey contextKey = "auth_claims"

// WithAuthClaims adds auth claims to context
func WithAuthClaims(ctx context.Context, claims *AuthClaims) context.Context {
	return context.WithValue(ctx, authClaimsContextKey, claims)
}

// AuthClaimsFromContext extracts auth claims from context
func AuthClaimsFromContext(ctx context.Context) (*AuthClaims, bool) {
	claims, ok := ctx.Value(authClaimsContextKey).(*AuthClaims)
	return claims, ok
}

// MustAuthClaimsFromContext extracts auth claims or panics
func MustAuthClaimsFromContext(ctx context.Context) *AuthClaims {
	claims, ok := AuthClaimsFromContext(ctx)
	if !ok {
		panic("auth claims not found in context")
	}
	return claims
}

// authWrappedStream wraps ServerStream with modified context
type authWrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *authWrappedStream) Context() context.Context {
	return w.ctx
}

// Helper functions
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getStringSliceClaim(claims jwt.MapClaims, key string) []string {
	if val, ok := claims[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}
