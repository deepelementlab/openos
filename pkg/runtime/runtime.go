// Package runtime provides the core runtime implementation for Agent OS.
package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

// Manager manages multiple runtime instances
type Manager struct {
	logger     *zap.Logger
	runtimes   map[string]interfaces.Runtime
	factory    interfaces.RuntimeFactory
	config     *types.RuntimeConfig
	mu         sync.RWMutex
	shutdownCh chan struct{}
}

// NewManager creates a new runtime manager
func NewManager(logger *zap.Logger, factory interfaces.RuntimeFactory, config *types.RuntimeConfig) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if factory == nil {
		return nil, fmt.Errorf("factory cannot be nil")
	}

	return &Manager{
		logger:     logger.With(zap.String("component", "runtime-manager")),
		runtimes:   make(map[string]interfaces.Runtime),
		factory:    factory,
		config:     config,
		shutdownCh: make(chan struct{}),
	}, nil
}

// Initialize initializes the runtime manager
func (m *Manager) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing runtime manager")

	// Create default runtime
	defaultRuntime, err := m.factory.CreateRuntime(ctx, string(m.config.Type), m.config)
	if err != nil {
		return fmt.Errorf("failed to create default runtime: %w", err)
	}

	// Initialize the runtime
	if err := defaultRuntime.Initialize(ctx, m.config); err != nil {
		return fmt.Errorf("failed to initialize default runtime: %w", err)
	}

	m.mu.Lock()
	m.runtimes["default"] = defaultRuntime
	m.mu.Unlock()

	m.logger.Info("Runtime manager initialized successfully",
		zap.String("runtime_type", string(m.config.Type)))

	return nil
}

// GetRuntime returns a runtime instance by name
func (m *Manager) GetRuntime(name string) (interfaces.Runtime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runtime, exists := m.runtimes[name]
	if !exists {
		return nil, fmt.Errorf("runtime %s not found", name)
	}

	return runtime, nil
}

// CreateRuntime creates a new runtime instance
func (m *Manager) CreateRuntime(ctx context.Context, name string, runtimeType string, config *types.RuntimeConfig) (interfaces.Runtime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.runtimes[name]; exists {
		return nil, fmt.Errorf("runtime %s already exists", name)
	}

	runtime, err := m.factory.CreateRuntime(ctx, runtimeType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime %s: %w", name, err)
	}

	if err := runtime.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime %s: %w", name, err)
	}

	m.runtimes[name] = runtime
	m.logger.Info("Created new runtime", zap.String("name", name), zap.String("type", runtimeType))

	return runtime, nil
}

// RemoveRuntime removes a runtime instance
func (m *Manager) RemoveRuntime(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot remove default runtime")
	}

	runtime, exists := m.runtimes[name]
	if !exists {
		return fmt.Errorf("runtime %s not found", name)
	}

	// Cleanup the runtime
	if err := runtime.Cleanup(ctx); err != nil {
		m.logger.Warn("Failed to cleanup runtime during removal", 
			zap.String("name", name), zap.Error(err))
	}

	delete(m.runtimes, name)
	m.logger.Info("Removed runtime", zap.String("name", name))

	return nil
}

// ListRuntimes lists all runtime instances
func (m *Manager) ListRuntimes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.runtimes))
	for name := range m.runtimes {
		names = append(names, name)
	}

	return names
}

// HealthCheck performs health check on all runtimes
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)

	for name, runtime := range m.runtimes {
		if err := runtime.HealthCheck(ctx); err != nil {
			results[name] = err
			m.logger.Warn("Runtime health check failed", 
				zap.String("name", name), zap.Error(err))
		}
	}

	return results
}

// Shutdown gracefully shuts down the runtime manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down runtime manager")

	close(m.shutdownCh)

	m.mu.RLock()
	runtimes := make(map[string]interfaces.Runtime)
	for k, v := range m.runtimes {
		runtimes[k] = v
	}
	m.mu.RUnlock()

	// Cleanup all runtimes
	errors := make([]error, 0)
	for name, runtime := range runtimes {
		m.logger.Debug("Cleaning up runtime", zap.String("name", name))
		if err := runtime.Cleanup(ctx); err != nil {
			errors = append(errors, fmt.Errorf("runtime %s cleanup failed: %w", name, err))
			m.logger.Warn("Runtime cleanup failed", 
				zap.String("name", name), zap.Error(err))
		}
	}

	m.mu.Lock()
	m.runtimes = make(map[string]interfaces.Runtime)
	m.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with errors: %v", errors)
	}

	m.logger.Info("Runtime manager shutdown completed")
	return nil
}

// StartHealthMonitoring starts health monitoring for all runtimes
func (m *Manager) StartHealthMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Health monitoring stopped due to context cancellation")
			return
		case <-m.shutdownCh:
			m.logger.Info("Health monitoring stopped due to shutdown")
			return
		case <-ticker.C:
			m.performHealthCheck(ctx)
		}
	}
}

func (m *Manager) performHealthCheck(ctx context.Context) {
	m.logger.Debug("Performing periodic health check")
	
	results := m.HealthCheck(ctx)
	if len(results) > 0 {
		m.logger.Warn("Some runtimes failed health check", 
			zap.Int("failed_count", len(results)))
		
		// Try to recover failed runtimes
		for name := range results {
			m.logger.Info("Attempting to recover runtime", zap.String("name", name))
			// TODO: Implement recovery logic
		}
	}
}

// DefaultRuntime returns the default runtime
func (m *Manager) DefaultRuntime() interfaces.Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.runtimes["default"]
}

// SupportedRuntimes returns list of supported runtime types
func (m *Manager) SupportedRuntimes() []string {
	return m.factory.SupportedRuntimes()
}