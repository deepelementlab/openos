// Package volume provides pluggable volume provisioning for agent workloads.
package volume

import "context"

// VolumeSpec describes a volume request.
type VolumeSpec struct {
	Name       string
	TenantID   string
	SizeBytes  int64
	AccessMode string // ReadWriteOnce, ReadOnlyMany
	Backend    string // local, nfs, future csi
}

// Volume is a provisioned volume handle.
type Volume struct {
	ID         string
	MountPath  string
	Backend    string
	DevicePath string
}

// Provisioner creates and deletes volumes.
type Provisioner interface {
	Provision(ctx context.Context, spec *VolumeSpec) (*Volume, error)
	Delete(ctx context.Context, volumeID string) error
}
