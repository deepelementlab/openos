package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/data/repository"
	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
	"go.uber.org/zap"
)

// LifecycleManager orchestrates Agent lifecycle operations, bridging
// the API-level repository model with the runtime-level container model.
type LifecycleManager struct {
	runtime interfaces.Runtime
	repo    repository.AgentRepository
	logger  *zap.Logger
}

// NewLifecycleManager creates a new lifecycle manager.
func NewLifecycleManager(rt interfaces.Runtime, repo repository.AgentRepository, logger *zap.Logger) *LifecycleManager {
	return &LifecycleManager{
		runtime: rt,
		repo:    repo,
		logger:  logger,
	}
}

// CreateAgent creates an agent in both the repository and the runtime.
func (m *LifecycleManager) CreateAgent(ctx context.Context, req CreateAgentRequest) (*repository.Agent, error) {
	if req.Name == "" || req.Image == "" {
		return nil, fmt.Errorf("name and image are required")
	}

	now := time.Now().UTC()
	repoAgent := &repository.Agent{
		ID:        req.ID,
		Name:      req.Name,
		Image:     req.Image,
		Runtime:   req.Runtime,
		Resources: req.Resources,
		Status:    repository.AgentStatusCreating,
		Metadata:  map[string]interface{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.repo.Create(ctx, repoAgent); err != nil {
		return nil, fmt.Errorf("persist agent: %w", err)
	}

	spec := &types.AgentSpec{
		ID:    req.ID,
		Name:  req.Name,
		Image: req.Image,
	}

	if _, err := m.runtime.CreateAgent(ctx, spec); err != nil {
		repoAgent.Status = repository.AgentStatusError
		repoAgent.UpdatedAt = time.Now().UTC()
		_ = m.repo.Update(ctx, repoAgent)
		return nil, fmt.Errorf("runtime create: %w", err)
	}

	repoAgent.Status = repository.AgentStatusPending
	repoAgent.UpdatedAt = time.Now().UTC()
	_ = m.repo.Update(ctx, repoAgent)

	m.logger.Info("agent created", zap.String("id", req.ID), zap.String("name", req.Name))
	return repoAgent, nil
}

// StartAgent starts a previously created agent.
func (m *LifecycleManager) StartAgent(ctx context.Context, agentID string) error {
	agent, err := m.repo.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	if err := m.runtime.StartAgent(ctx, agentID); err != nil {
		agent.Status = repository.AgentStatusError
		agent.UpdatedAt = time.Now().UTC()
		_ = m.repo.Update(ctx, agent)
		return fmt.Errorf("runtime start: %w", err)
	}

	agent.Status = repository.AgentStatusRunning
	agent.UpdatedAt = time.Now().UTC()
	_ = m.repo.Update(ctx, agent)

	m.logger.Info("agent started", zap.String("id", agentID))
	return nil
}

// StopAgent stops a running agent.
func (m *LifecycleManager) StopAgent(ctx context.Context, agentID string, timeout time.Duration) error {
	agent, err := m.repo.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	agent.Status = repository.AgentStatusStopping
	agent.UpdatedAt = time.Now().UTC()
	_ = m.repo.Update(ctx, agent)

	if err := m.runtime.StopAgent(ctx, agentID, timeout); err != nil {
		agent.Status = repository.AgentStatusError
		agent.UpdatedAt = time.Now().UTC()
		_ = m.repo.Update(ctx, agent)
		return fmt.Errorf("runtime stop: %w", err)
	}

	agent.Status = repository.AgentStatusStopped
	agent.UpdatedAt = time.Now().UTC()
	_ = m.repo.Update(ctx, agent)

	m.logger.Info("agent stopped", zap.String("id", agentID))
	return nil
}

// DeleteAgent deletes an agent from both runtime and repository.
func (m *LifecycleManager) DeleteAgent(ctx context.Context, agentID string) error {
	if err := m.runtime.DeleteAgent(ctx, agentID); err != nil {
		m.logger.Warn("runtime delete failed, continuing with repo cleanup", zap.Error(err))
	}

	if err := m.repo.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("repo delete: %w", err)
	}

	m.logger.Info("agent deleted", zap.String("id", agentID))
	return nil
}

// GetAgent retrieves an agent with current runtime state.
func (m *LifecycleManager) GetAgent(ctx context.Context, agentID string) (*repository.Agent, error) {
	return m.repo.Get(ctx, agentID)
}

// CreateAgentRequest holds parameters for creating an agent.
type CreateAgentRequest struct {
	ID        string
	Name      string
	Image     string
	Runtime   string
	Resources map[string]string
}
