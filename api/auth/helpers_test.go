package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUserContext(t *testing.T) {
	ctx := context.Background()
	roles := []string{"admin", "user"}

	result := SetUserContext(ctx, "user-1", "alice", roles)

	// Verify all values are stored
	uid, ok := ExtractUserID(result)
	assert.True(t, ok)
	assert.Equal(t, "user-1", uid)

	username, ok := ExtractUsername(result)
	assert.True(t, ok)
	assert.Equal(t, "alice", username)

	gotRoles := ExtractRoles(result)
	assert.Equal(t, roles, gotRoles)
}

func TestExtractUserID_Missing(t *testing.T) {
	ctx := context.Background()
	uid, ok := ExtractUserID(ctx)
	assert.False(t, ok)
	assert.Empty(t, uid)
}

func TestExtractUserID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxUserID, 12345)
	uid, ok := ExtractUserID(ctx)
	assert.False(t, ok)
	assert.Empty(t, uid)
}

func TestExtractUsername_Missing(t *testing.T) {
	ctx := context.Background()
	username, ok := ExtractUsername(ctx)
	assert.False(t, ok)
	assert.Empty(t, username)
}

func TestExtractUsername_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxUsername, 12345)
	username, ok := ExtractUsername(ctx)
	assert.False(t, ok)
	assert.Empty(t, username)
}

func TestExtractRoles_Missing(t *testing.T) {
	ctx := context.Background()
	roles := ExtractRoles(ctx)
	assert.Nil(t, roles)
}

func TestExtractRoles_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxRoles, "not-a-slice")
	roles := ExtractRoles(ctx)
	assert.Nil(t, roles)
}

func TestExtractRoles_Empty(t *testing.T) {
	ctx := SetUserContext(context.Background(), "u1", "alice", []string{})
	roles := ExtractRoles(ctx)
	assert.Empty(t, roles)
}

func TestRequireRole_Allowed(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx := SetUserContext(context.Background(), "u1", "alice", []string{"admin", "user"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Denied(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx := SetUserContext(context.Background(), "u1", "alice", []string{"viewer"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireRole_NoRoles(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx := SetUserContext(context.Background(), "u1", "alice", []string{})
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireRole_MultipleRequired(t *testing.T) {
	tests := []struct {
		name       string
		userRoles  []string
		reqRoles   []string
		wantStatus int
	}{
		{
			name:       "has first required role",
			userRoles:  []string{"admin"},
			reqRoles:   []string{"admin", "superadmin"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "has second required role",
			userRoles:  []string{"superadmin"},
			reqRoles:   []string{"admin", "superadmin"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "has neither required role",
			userRoles:  []string{"viewer"},
			reqRoles:   []string{"admin", "superadmin"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireRole(tt.reqRoles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			ctx := SetUserContext(context.Background(), "u1", "alice", tt.userRoles)
			req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestSetUserContext_EmptyValues(t *testing.T) {
	ctx := SetUserContext(context.Background(), "", "", nil)

	uid, ok := ExtractUserID(ctx)
	assert.True(t, ok)
	assert.Empty(t, uid)

	username, ok := ExtractUsername(ctx)
	assert.True(t, ok)
	assert.Empty(t, username)

	roles := ExtractRoles(ctx)
	assert.Nil(t, roles)
}

func TestSetUserContext_OverridesExisting(t *testing.T) {
	ctx := SetUserContext(context.Background(), "user-1", "alice", []string{"admin"})
	ctx = SetUserContext(ctx, "user-2", "bob", []string{"viewer"})

	uid, _ := ExtractUserID(ctx)
	assert.Equal(t, "user-2", uid)

	username, _ := ExtractUsername(ctx)
	assert.Equal(t, "bob", username)

	roles := ExtractRoles(ctx)
	assert.Equal(t, []string{"viewer"}, roles)
}

func TestRequireRole_IntegrationWithHTTP(t *testing.T) {
	mux := http.NewServeMux()

	// Protected endpoint
	mux.Handle("/protected", RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("secret"))
	})))

	tests := []struct {
		name       string
		roles      []string
		wantStatus int
		wantBody   string
	}{
		{"admin gets access", []string{"admin"}, http.StatusOK, "secret"},
		{"viewer denied", []string{"viewer"}, http.StatusForbidden, "Forbidden\n"},
		{"no roles denied", []string{}, http.StatusForbidden, "Forbidden\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SetUserContext(context.Background(), "u1", "test", tt.roles)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantBody, rec.Body.String())
		})
	}
}

func TestContextKeyValues(t *testing.T) {
	require.Equal(t, contextKey("user_id"), ctxUserID)
	require.Equal(t, contextKey("username"), ctxUsername)
	require.Equal(t, contextKey("roles"), ctxRoles)
}
