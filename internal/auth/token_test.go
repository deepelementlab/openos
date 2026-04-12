package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTTokenService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTTokenService("test-secret", "aos-test", 1*time.Hour)

	token, err := svc.GenerateToken("user-1", "testuser", []string{"admin"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Contains(t, claims.Roles, "admin")
}

func TestJWTTokenService_GenerateEmptyUserID(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 1*time.Hour)
	_, err := svc.GenerateToken("", "user", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestJWTTokenService_ValidateInvalidToken(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 1*time.Hour)
	_, err := svc.ValidateToken("invalid.token.string")
	assert.Error(t, err)
}

func TestJWTTokenService_ValidateWrongSecret(t *testing.T) {
	svc1 := NewJWTTokenService("secret-1", "aos", 1*time.Hour)
	svc2 := NewJWTTokenService("secret-2", "aos", 1*time.Hour)

	token, _ := svc1.GenerateToken("user-1", "test", nil)
	_, err := svc2.ValidateToken(token)
	assert.Error(t, err)
}

func TestJWTTokenService_RefreshToken(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 1*time.Hour)

	token, _ := svc.GenerateToken("user-1", "testuser", []string{"viewer"})
	newToken, err := svc.RefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)

	claims, err := svc.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestJWTTokenService_RefreshInvalidToken(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 1*time.Hour)
	_, err := svc.RefreshToken("invalid.token")
	assert.Error(t, err)
}

func TestJWTTokenService_DefaultExpiration(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 0)
	assert.Equal(t, 24*time.Hour, svc.expiration)
}

func TestJWTTokenService_MultipleRoles(t *testing.T) {
	svc := NewJWTTokenService("secret", "aos", 1*time.Hour)
	token, _ := svc.GenerateToken("user-1", "test", []string{"admin", "editor", "viewer"})
	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Len(t, claims.Roles, 3)
}
