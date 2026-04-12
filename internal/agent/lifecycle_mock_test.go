package agent

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
	"go.uber.org/zap"
)

// mockRuntime implements interfaces.Runtime for testing
type mockRuntime struct {
	createErr error
	startErr  error
	stopErr   error
	deleteErr error
	agents    map[string]*types.Agent
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{agents: make(map[string]*types.Agent)}
}

func (m *mockRuntime) Initialize(_ context.Context, _ *types.RuntimeConfig) error { return nil }
func (m *mockRuntime) GetRuntimeInfo() *types.RuntimeInfo {
	return &types.RuntimeInfo{Type: "mock", Version: "test"}
}
func (m *mockRuntime) CreateAgent(_ context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	a := &types.Agent{ID: spec.ID, Name: spec.Name, Image: spec.Image, Status: "created"}
	m.agents[spec.ID] = a
	return a, nil
}
func (m *mockRuntime) StartAgent(_ context.Context, agentID string) error {
	if m.startErr != nil {
		return m.startErr
	}
	if a, ok := m.agents[agentID]; ok {
		a.Status = "running"
	}
	return nil
}
func (m *mockRuntime) StopAgent(_ context.Context, agentID string, _ time.Duration) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	if a, ok := m.agents[agentID]; ok {
		a.Status = "stopped"
	}
	return nil
}
func (m *mockRuntime) DeleteAgent(_ context.Context, agentID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.agents, agentID)
	return nil
}
func (m *mockRuntime) GetAgent(_ context.Context, agentID string) (*types.Agent, error) {
	a, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return a, nil
}
func (m *mockRuntime) ListAgents(_ context.Context, _ *types.AgentFilter) ([]*types.Agent, error) {
	var list []*types.Agent
	for _, a := range m.agents {
		list = append(list, a)
	}
	return list, nil
}
func (m *mockRuntime) ExecuteCommand(_ context.Context, _ string, _ *types.Command) (*types.CommandResult, error) {
	return &types.CommandResult{}, nil
}
func (m *mockRuntime) GetAgentLogs(_ context.Context, _ string, _ *types.LogOptions) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockRuntime) GetAgentStats(_ context.Context, _ string) (*types.ResourceStats, error) {
	return &types.ResourceStats{}, nil
}
func (m *mockRuntime) UpdateAgent(_ context.Context, _ string, _ *types.AgentSpec) error { return nil }
func (m *mockRuntime) ResizeAgentTerminal(_ context.Context, _ string, _, _ uint) error   { return nil }
func (m *mockRuntime) AttachAgent(_ context.Context, _ string, _ *types.AttachOptions) (io.ReadWriteCloser, error) {
	return nil, nil
}
func (m *mockRuntime) HealthCheck(_ context.Context) error { return nil }
func (m *mockRuntime) Cleanup(_ context.Context) error     { return nil }

// Ensure mockRuntime satisfies the interface at compile time
var _ interfaces.Runtime = (*mockRuntime)(nil)

func TestLifecycleManager_CreateAgent_Success(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	req := CreateAgentRequest{
		ID:      "agent-1",
		Name:    "test-agent",
		Image:   "nginx:latest",
		Runtime: "containerd",
	}

	agent, err := lm.CreateAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != "agent-1" {
		t.Errorf("expected agent-1, got %s", agent.ID)
	}
	if agent.Name != "test-agent" {
		t.Errorf("expected test-agent, got %s", agent.Name)
	}
	if agent.Status != repository.AgentStatusPending {
		t.Errorf("expected pending after successful create, got %s", agent.Status)
	}

	// Verify persisted
	found, _ := repo.Get(context.Background(), "agent-1")
	if found == nil {
		t.Error("agent should be persisted in repo")
	}
}

func TestLifecycleManager_CreateAgent_ValidationError(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	tests := []struct {
		name string
		req  CreateAgentRequest
	}{
		{name: "empty name and image", req: CreateAgentRequest{ID: "x"}},
		{name: "empty name", req: CreateAgentRequest{ID: "x", Image: "nginx"}},
		{name: "empty image", req: CreateAgentRequest{ID: "x", Name: "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := lm.CreateAgent(context.Background(), tt.req)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestLifecycleManager_CreateAgent_RuntimeError(t *testing.T) {
	rt := newMockRuntime()
	rt.createErr = fmt.Errorf("runtime unavailable")
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	req := CreateAgentRequest{ID: "a1", Name: "test", Image: "nginx"}
	_, err := lm.CreateAgent(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}

	// Agent should be persisted with error status
	found, _ := repo.Get(context.Background(), "a1")
	if found == nil {
		t.Fatal("agent should be persisted")
	}
	if found.Status != repository.AgentStatusError {
		t.Errorf("expected error status, got %s", found.Status)
	}
}

func TestLifecycleManager_StartAgent_Success(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusPending,
	})

	err := lm.StartAgent(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.Get(ctx, "a1")
	if found.Status != repository.AgentStatusRunning {
		t.Errorf("expected running, got %s", found.Status)
	}
}

func TestLifecycleManager_StartAgent_NotFound(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	err := lm.StartAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestLifecycleManager_StartAgent_RuntimeError(t *testing.T) {
	rt := newMockRuntime()
	rt.startErr = fmt.Errorf("start failed")
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusPending,
	})

	err := lm.StartAgent(ctx, "a1")
	if err == nil {
		t.Fatal("expected error")
	}

	found, _ := repo.Get(ctx, "a1")
	if found.Status != repository.AgentStatusError {
		t.Errorf("expected error status, got %s", found.Status)
	}
}

func TestLifecycleManager_StopAgent_Success(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusRunning,
	})

	err := lm.StopAgent(ctx, "a1", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.Get(ctx, "a1")
	if found.Status != repository.AgentStatusStopped {
		t.Errorf("expected stopped, got %s", found.Status)
	}
}

func TestLifecycleManager_StopAgent_NotFound(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	err := lm.StopAgent(context.Background(), "nonexistent", 10*time.Second)
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestLifecycleManager_StopAgent_RuntimeError(t *testing.T) {
	rt := newMockRuntime()
	rt.stopErr = fmt.Errorf("stop failed")
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusRunning,
	})

	err := lm.StopAgent(ctx, "a1", 30*time.Second)
	if err == nil {
		t.Fatal("expected error")
	}

	found, _ := repo.Get(ctx, "a1")
	if found.Status != repository.AgentStatusError {
		t.Errorf("expected error, got %s", found.Status)
	}
}

func TestLifecycleManager_DeleteAgent_Success(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusStopped,
	})

	err := lm.DeleteAgent(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.Get(ctx, "a1")
	if err == nil {
		t.Error("agent should be deleted from repo")
	}
}

func TestLifecycleManager_DeleteAgent_RuntimeError_ContinueRepoCleanup(t *testing.T) {
	rt := newMockRuntime()
	rt.deleteErr = fmt.Errorf("runtime delete failed")
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusStopped,
	})

	// Should succeed despite runtime error (cleanup continues)
	err := lm.DeleteAgent(ctx, "a1")
	if err != nil {
		t.Fatalf("should succeed with repo cleanup: %v", err)
	}
	_, err = repo.Get(ctx, "a1")
	if err == nil {
		t.Error("agent should be deleted from repo even with runtime error")
	}
}

func TestLifecycleManager_DeleteAgent_RepoError(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	err := lm.DeleteAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for deleting nonexistent agent from repo")
	}
}

func TestLifecycleManager_GetAgent_Success(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	ctx := context.Background()
	repo.Create(ctx, &repository.Agent{
		ID: "a1", Name: "test", Image: "nginx", Status: repository.AgentStatusRunning,
	})

	agent, err := lm.GetAgent(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != "a1" {
		t.Errorf("expected a1, got %s", agent.ID)
	}
}

func TestLifecycleManager_GetAgent_NotFound(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	_, err := lm.GetAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestLifecycleManager_FullLifecycle(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())
	ctx := context.Background()

	// Create
	agent, err := lm.CreateAgent(ctx, CreateAgentRequest{
		ID: "lc-1", Name: "lifecycle-test", Image: "python:3.11",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if agent.Status != repository.AgentStatusPending {
		t.Errorf("expected pending, got %s", agent.Status)
	}

	// Start
	err = lm.StartAgent(ctx, "lc-1")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	agent, _ = lm.GetAgent(ctx, "lc-1")
	if agent.Status != repository.AgentStatusRunning {
		t.Errorf("expected running, got %s", agent.Status)
	}

	// Stop
	err = lm.StopAgent(ctx, "lc-1", 10*time.Second)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	agent, _ = lm.GetAgent(ctx, "lc-1")
	if agent.Status != repository.AgentStatusStopped {
		t.Errorf("expected stopped, got %s", agent.Status)
	}

	// Delete
	err = lm.DeleteAgent(ctx, "lc-1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = lm.GetAgent(ctx, "lc-1")
	if err == nil {
		t.Error("agent should be deleted")
	}
}

func TestLifecycleManager_CreateAgent_WithResources(t *testing.T) {
	rt := newMockRuntime()
	repo := repository.NewInMemoryAgentRepository()
	lm := NewLifecycleManager(rt, repo, zap.NewNop())

	req := CreateAgentRequest{
		ID:      "agent-res",
		Name:    "resource-agent",
		Image:   "python:3.11",
		Runtime: "gvisor",
		Resources: map[string]string{
			"cpu":    "2000m",
			"memory": "4Gi",
			"gpu":    "1",
		},
	}

	agent, err := lm.CreateAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Resources["cpu"] != "2000m" {
		t.Errorf("expected cpu=2000m, got %s", agent.Resources["cpu"])
	}
	if agent.Resources["gpu"] != "1" {
		t.Errorf("expected gpu=1, got %s", agent.Resources["gpu"])
	}
	if agent.Runtime != "gvisor" {
		t.Errorf("expected gvisor runtime, got %s", agent.Runtime)
	}
}
