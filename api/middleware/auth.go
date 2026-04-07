package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agentos/aos/api/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// AuthMiddleware implements authentication and authorization
type AuthMiddleware struct {
	logger       *zap.Logger
	jwtSecret    string
	redisClient  *redis.Client
	rbacEnabled  bool
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(logger *zap.Logger, jwtSecret string, redisClient *redis.Client, rbacEnabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		logger:      logger,
		jwtSecret:   jwtSecret,
		redisClient: redisClient,
		rbacEnabled: rbacEnabled,
	}
}

// Authenticate validates JWT token and extracts claims
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for public endpoints
		if m.isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Get token from header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.sendError(w, http.StatusUnauthorized, "AUTH_TOKEN_MISSING", "Authorization header is required")
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			m.sendError(w, http.StatusUnauthorized, "AUTH_TOKEN_INVALID_FORMAT", "Token format should be: Bearer <token>")
			return
		}

		tokenString := parts[1]
		claims, err := m.validateToken(tokenString)
		if err != nil {
			m.logger.Debug("Token validation failed", zap.Error(err))
			m.sendError(w, http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "Invalid or expired token")
			return
		}

		// Check if token is blacklisted
		if m.isTokenBlacklisted(r.Context(), tokenString) {
			m.sendError(w, http.StatusUnauthorized, "AUTH_TOKEN_REVOKED", "Token has been revoked")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		ctx = context.WithValue(ctx, "scopes", claims.Scopes)
		ctx = context.WithValue(ctx, "token", tokenString)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authorize implements RBAC authorization
func (m *AuthMiddleware) Authorize(requiredRoles []string, requiredScopes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.rbacEnabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get roles and scopes from context
			ctx := r.Context()
			userRoles, _ := ctx.Value("roles").([]string)
			userScopes, _ := ctx.Value("scopes").([]string)

			// Check roles
			if len(requiredRoles) > 0 {
				hasRole := false
				for _, requiredRole := range requiredRoles {
					for _, userRole := range userRoles {
						if userRole == requiredRole {
							hasRole = true
							break
						}
					}
					if hasRole {
						break
					}
				}
				if !hasRole {
					m.sendError(w, http.StatusForbidden, "AUTH_INSUFFICIENT_ROLES", "Insufficient roles")
					return
				}
			}

			// Check scopes
			if len(requiredScopes) > 0 {
				hasScope := false
				for _, requiredScope := range requiredScopes {
					for _, userScope := range userScopes {
						if userScope == requiredScope {
							hasScope = true
							break
						}
					}
					if hasScope {
						break
					}
				}
				if !hasScope {
					m.sendError(w, http.StatusForbidden, "AUTH_INSUFFICIENT_SCOPES", "Insufficient scopes")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateToken validates JWT token and returns claims
func (m *AuthMiddleware) ValidateToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString)
}

// GenerateToken generates a new JWT token
func (m *AuthMiddleware) GenerateToken(userID, username string, roles, scopes []string, expiresIn time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.jwtSecret))
}

// RevokeToken adds token to blacklist
func (m *AuthMiddleware) RevokeToken(ctx context.Context, tokenString string) error {
	claims, err := m.validateToken(tokenString)
	if err != nil {
		return err
	}

	// Calculate remaining TTL
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}

	// Add to Redis blacklist
	key := fmt.Sprintf("token_blacklist:%s", tokenString)
	return m.redisClient.Set(ctx, key, "revoked", ttl).Err()
}

// validateToken validates JWT token
func (m *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// isTokenBlacklisted checks if token is in Redis blacklist
func (m *AuthMiddleware) isTokenBlacklisted(ctx context.Context, tokenString string) bool {
	if m.redisClient == nil {
		return false
	}

	key := fmt.Sprintf("token_blacklist:%s", tokenString)
	val, err := m.redisClient.Get(ctx, key).Result()
	return err == nil && val == "revoked"
}

// isPublicEndpoint checks if endpoint doesn't require authentication
func (m *AuthMiddleware) isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/ready",
		"/live",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
	}

	for _, publicPath := range publicPaths {
		if path == publicPath || strings.HasPrefix(path, publicPath+"/") {
			return true
		}
	}

	return false
}

// sendError sends standardized error response
func (m *AuthMiddleware) sendError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.ErrorResponse(code, message)
	jsonData, _ := json.Marshal(response)
	w.Write(jsonData)
}