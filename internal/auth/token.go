package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims holds the application-specific JWT claims.
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenService generates and validates authentication tokens.
type TokenService interface {
	GenerateToken(userID, username string, roles []string) (string, error)
	ValidateToken(tokenStr string) (*TokenClaims, error)
	RefreshToken(tokenStr string) (string, error)
}

// JWTTokenService implements TokenService using JWT.
type JWTTokenService struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

// NewJWTTokenService creates a new JWT-based token service.
func NewJWTTokenService(secret string, issuer string, expiration time.Duration) *JWTTokenService {
	if expiration <= 0 {
		expiration = 24 * time.Hour
	}
	return &JWTTokenService{
		secret:     []byte(secret),
		issuer:     issuer,
		expiration: expiration,
	}
}

func (s *JWTTokenService) GenerateToken(userID, username string, roles []string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID is required")
	}

	now := time.Now()
	claims := TokenClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTTokenService) ValidateToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func (s *JWTTokenService) RefreshToken(tokenStr string) (string, error) {
	claims, err := s.ValidateToken(tokenStr)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}
	return s.GenerateToken(claims.UserID, claims.Username, claims.Roles)
}

var _ TokenService = (*JWTTokenService)(nil)
