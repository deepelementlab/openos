package server

import (
	"testing"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/stretchr/testify/assert"
)

func TestCreateAgentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateAgentRequest
		wantErr int
	}{
		{"valid", CreateAgentRequest{Name: "test", Image: "nginx"}, 0},
		{"missing name", CreateAgentRequest{Image: "nginx"}, 1},
		{"missing image", CreateAgentRequest{Name: "test"}, 1},
		{"missing both", CreateAgentRequest{}, 2},
		{"name too long", CreateAgentRequest{Name: string(make([]byte, 256)), Image: "nginx"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			assert.Len(t, errs, tt.wantErr)
		})
	}
}

func TestUpdateAgentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateAgentRequest
		wantErr int
	}{
		{"empty valid", UpdateAgentRequest{}, 0},
		{"valid name", UpdateAgentRequest{Name: "test"}, 0},
		{"valid status", UpdateAgentRequest{Status: repository.AgentStatusRunning}, 0},
		{"name too long", UpdateAgentRequest{Name: string(make([]byte, 256))}, 1},
		{"invalid status", UpdateAgentRequest{Status: "invalid_status"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			assert.Len(t, errs, tt.wantErr)
		})
	}
}

func TestPaginationParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  PaginationParams
		wantErr int
	}{
		{"valid", PaginationParams{Page: 1, PageSize: 20}, 0},
		{"negative page", PaginationParams{Page: -1, PageSize: 20}, 1},
		{"negative pagesize", PaginationParams{Page: 1, PageSize: -1}, 1},
		{"too large pagesize", PaginationParams{Page: 1, PageSize: 200}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.params.Validate()
			assert.Len(t, errs, tt.wantErr)
		})
	}
}

func TestPaginationParams_GetPage(t *testing.T) {
	assert.Equal(t, 1, (&PaginationParams{Page: 0}).GetPage())
	assert.Equal(t, 1, (&PaginationParams{Page: -1}).GetPage())
	assert.Equal(t, 5, (&PaginationParams{Page: 5}).GetPage())
}

func TestPaginationParams_GetPageSize(t *testing.T) {
	assert.Equal(t, 20, (&PaginationParams{PageSize: 0}).GetPageSize())
	assert.Equal(t, 20, (&PaginationParams{PageSize: -1}).GetPageSize())
	assert.Equal(t, 50, (&PaginationParams{PageSize: 50}).GetPageSize())
	assert.Equal(t, 100, (&PaginationParams{PageSize: 200}).GetPageSize())
}

func TestSuccessAPIResponse(t *testing.T) {
	resp := SuccessAPIResponse(map[string]string{"id": "test"})
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Error)
	assert.NotEmpty(t, resp.RequestID)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestErrorAPIResponse(t *testing.T) {
	resp := ErrorAPIResponse("NOT_FOUND", "resource not found")
	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	assert.Equal(t, "resource not found", resp.Error.Message)
}

func TestErrorAPIResponseWithDetails(t *testing.T) {
	resp := ErrorAPIResponseWithDetails("VALIDATION", "invalid input", "name is required")
	assert.False(t, resp.Success)
	assert.Equal(t, "name is required", resp.Error.Details)
}

func TestAPIResponse_FieldTypes(t *testing.T) {
	resp := APIResponse{
		Success:   true,
		Data:      "test",
		RequestID: "req-123",
		Timestamp: "2025-01-01T00:00:00Z",
	}
	assert.True(t, resp.Success)
	assert.Equal(t, "test", resp.Data)
	assert.Equal(t, "req-123", resp.RequestID)
}
