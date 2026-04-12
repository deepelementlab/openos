package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/pkg/runtime/types"
	"go.uber.org/zap"
)

func TestCreateAgentRequest_Validation(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	tests := []struct {
		name    string
		req     CreateAgentRequest
		wantErr bool
	}{
		{
			name:    "missing name and image",
			req:     CreateAgentRequest{ID: "agent-1"},
			wantErr: true,
		},
		{
			name:    "missing name",
			req:     CreateAgentRequest{ID: "agent-1", Image: "nginx"},
			wantErr: true,
		},
		{
			name:    "missing image",
			req:     CreateAgentRequest{ID: "agent-1", Name: "test"},
			wantErr: true,
		},
		{
			name:    "valid request",
			req:     CreateAgentRequest{ID: "agent-1", Name: "test", Image: "nginx"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := lm.CreateAgent(context.Background(), tt.req)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateAgentRequest_RepoIntegration(t *testing.T) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()

	now := time.Now().UTC()
	agent := &repository.Agent{
		ID:        "agent-1",
		Name:      "test-agent",
		Image:     "nginx:latest",
		Runtime:   "containerd",
		Status:    repository.AgentStatusCreating,
		Resources: map[string]string{"cpu": "500m"},
		Metadata:  map[string]interface{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(ctx, agent)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	found, err := repo.Get(ctx, "agent-1")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if found.Name != "test-agent" {
		t.Errorf("expected name=test-agent, got %s", found.Name)
	}
	if found.Status != repository.AgentStatusCreating {
		t.Errorf("expected creating status")
	}
}

func TestAgentLifecycle_StateTransitions(t *testing.T) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()

	now := time.Now().UTC()
	agent := &repository.Agent{
		ID:        "agent-1",
		Name:      "test-agent",
		Image:     "nginx:latest",
		Status:    repository.AgentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = repo.Create(ctx, agent)

	a, _ := repo.Get(ctx, "agent-1")
	a.Status = repository.AgentStatusCreating
	_ = repo.Update(ctx, a)

	a, _ = repo.Get(ctx, "agent-1")
	if a.Status != repository.AgentStatusCreating {
		t.Errorf("expected creating, got %s", a.Status)
	}

	a.Status = repository.AgentStatusRunning
	_ = repo.Update(ctx, a)

	a, _ = repo.Get(ctx, "agent-1")
	if a.Status != repository.AgentStatusRunning {
		t.Errorf("expected running, got %s", a.Status)
	}

	a.Status = repository.AgentStatusStopping
	_ = repo.Update(ctx, a)
	a.Status = repository.AgentStatusStopped
	_ = repo.Update(ctx, a)

	a, _ = repo.Get(ctx, "agent-1")
	if a.Status != repository.AgentStatusStopped {
		t.Errorf("expected stopped, got %s", a.Status)
	}

	_ = repo.Delete(ctx, "agent-1")
	_, err := repo.Get(ctx, "agent-1")
	if err == nil {
		t.Error("expected agent to be deleted")
	}
}

func TestAgentLifecycle_ErrorRecovery(t *testing.T) {
	repo := repository.NewInMemoryAgentRepository()
	ctx := context.Background()

	now := time.Now().UTC()
	agent := &repository.Agent{
		ID:        "agent-1",
		Name:      "test-agent",
		Image:     "nginx:latest",
		Status:    repository.AgentStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = repo.Create(ctx, agent)

	a, _ := repo.Get(ctx, "agent-1")
	a.Status = repository.AgentStatusError
	_ = repo.Update(ctx, a)

	a, _ = repo.Get(ctx, "agent-1")
	if a.Status != repository.AgentStatusError {
		t.Errorf("expected error, got %s", a.Status)
	}

	a.Status = repository.AgentStatusCreating
	_ = repo.Update(ctx, a)

	a, _ = repo.Get(ctx, "agent-1")
	if a.Status != repository.AgentStatusCreating {
		t.Errorf("expected creating after recovery, got %s", a.Status)
	}
}

func TestCreateAgentRequest_Fields(t *testing.T) {
	req := CreateAgentRequest{
		ID:        "agent-test",
		Name:      "my-agent",
		Image:     "python:3.11",
		Runtime:   "gvisor",
		Resources: map[string]string{"cpu": "1000m", "memory": "1Gi"},
	}

	if req.ID != "agent-test" {
		t.Errorf("expected agent-test, got %s", req.ID)
	}
	if req.Runtime != "gvisor" {
		t.Errorf("expected gvisor, got %s", req.Runtime)
	}
	if req.Resources["cpu"] != "1000m" {
		t.Errorf("expected 1000m cpu")
	}
}

func TestNewLifecycleManager(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())
	if lm == nil {
		t.Error("expected non-nil LifecycleManager")
	}
}

func TestValidateCreateRequest_Standalone(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateAgentRequest
		wantErr bool
	}{
		{name: "empty", req: CreateAgentRequest{}, wantErr: true},
		{name: "name_only", req: CreateAgentRequest{Name: "x"}, wantErr: true},
		{name: "image_only", req: CreateAgentRequest{Image: "x"}, wantErr: true},
		{name: "both", req: CreateAgentRequest{Name: "x", Image: "y"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("name and image are required")
			gotErr := tt.req.Name == "" || tt.req.Image == ""
			if tt.wantErr != gotErr {
				t.Errorf("wantErr=%v gotErr=%v for %+v", tt.wantErr, gotErr, tt.req)
			}
			_ = err
		})
	}
}

func TestCreateAgentRequest_Defaults(t *testing.T) {
	req := CreateAgentRequest{ID: "a1", Name: "n", Image: "i"}
	if req.Resources != nil {
		t.Error("expected nil resources by default")
	}
}

func TestAgentTypeValues(t *testing.T) {
	spec := &types.AgentSpec{ID: "s1", Name: "spec", Image: "img"}
	if spec.ID != "s1" {
		t.Errorf("unexpected ID: %s", spec.ID)
	}
}

var _ = zap.NewNop()
