package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/google/uuid"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func SuccessAPIResponse(data interface{}) APIResponse {
	return APIResponse{
		Success:   true,
		Data:      data,
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func ErrorAPIResponse(code, message string) APIResponse {
	return APIResponse{
		Success:   false,
		Error:     &APIError{Code: code, Message: message},
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func ErrorAPIResponseWithDetails(code, message, details string) APIResponse {
	return APIResponse{
		Success:   false,
		Error:     &APIError{Code: code, Message: message, Details: details},
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func WriteAPIResponse(w http.ResponseWriter, statusCode int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

type CreateAgentRequest struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Runtime     string            `json:"runtime,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func (r *CreateAgentRequest) Validate() []string {
	var errs []string
	if strings.TrimSpace(r.Name) == "" {
		errs = append(errs, "name is required")
	}
	if strings.TrimSpace(r.Image) == "" {
		errs = append(errs, "image is required")
	}
	if len(r.Name) > 255 {
		errs = append(errs, "name must be less than 255 characters")
	}
	return errs
}

type UpdateAgentRequest struct {
	Name      string                 `json:"name,omitempty"`
	Image     string                 `json:"image,omitempty"`
	Runtime   string                 `json:"runtime,omitempty"`
	Status    repository.AgentStatus `json:"status,omitempty"`
	Resources map[string]string      `json:"resources,omitempty"`
}

func (r *UpdateAgentRequest) Validate() []string {
	var errs []string
	if r.Name != "" && len(r.Name) > 255 {
		errs = append(errs, "name must be less than 255 characters")
	}
	if r.Status != "" {
		valid := false
		for _, s := range []repository.AgentStatus{
			repository.AgentStatusPending,
			repository.AgentStatusCreating,
			repository.AgentStatusRunning,
			repository.AgentStatusStopping,
			repository.AgentStatusStopped,
			repository.AgentStatusError,
		} {
			if r.Status == s {
				valid = true
				break
			}
		}
		if !valid {
			errs = append(errs, "invalid status value")
		}
	}
	return errs
}

type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func (p *PaginationParams) Validate() []string {
	var errs []string
	if p.Page < 0 {
		errs = append(errs, "page must be non-negative")
	}
	if p.PageSize < 0 {
		errs = append(errs, "page_size must be non-negative")
	}
	if p.PageSize > 100 {
		errs = append(errs, "page_size must not exceed 100")
	}
	return errs
}

func (p *PaginationParams) GetPage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p *PaginationParams) GetPageSize() int {
	if p.PageSize < 1 {
		return 20
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}
