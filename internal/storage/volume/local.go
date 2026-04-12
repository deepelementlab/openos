package volume

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalProvisioner stores data under a root directory (development / single-node).
type LocalProvisioner struct {
	Root string
}

// NewLocalProvisioner creates a local provisioner.
func NewLocalProvisioner(root string) *LocalProvisioner {
	return &LocalProvisioner{Root: root}
}

// Provision creates a directory for the volume.
func (p *LocalProvisioner) Provision(ctx context.Context, spec *VolumeSpec) (*Volume, error) {
	_ = ctx
	if spec.Name == "" {
		return nil, fmt.Errorf("volume: name required")
	}
	dir := filepath.Join(p.Root, spec.TenantID, spec.Name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Volume{
		ID:        spec.Name,
		MountPath: dir,
		Backend:   "local",
	}, nil
}

// Delete removes the volume directory.
func (p *LocalProvisioner) Delete(ctx context.Context, volumeID string) error {
	_ = ctx
	dir := filepath.Join(p.Root, volumeID)
	return os.RemoveAll(dir)
}
