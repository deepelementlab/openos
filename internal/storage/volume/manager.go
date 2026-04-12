package volume

import (
	"context"
	"errors"
	"sync"
)

// Manager routes volume operations to a provisioner.
type Manager struct {
	mu          sync.RWMutex
	provisioner Provisioner
}

// NewManager creates a volume manager with the given provisioner.
func NewManager(p Provisioner) *Manager {
	return &Manager{provisioner: p}
}

// Create provisions a volume using the configured backend.
func (m *Manager) Create(ctx context.Context, spec *VolumeSpec) (*Volume, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.provisioner == nil {
		return nil, errors.New("volume: no provisioner configured")
	}
	return m.provisioner.Provision(ctx, spec)
}
