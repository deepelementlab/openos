package tenant

import (
	"fmt"
	"time"
)

type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
)

type ResourceQuota struct {
	MaxAgents          int
	MaxCPU             int
	MaxMemoryGB        int
	MaxStorageGB       int
	MaxGPU             int
	MaxRequestsPerMin  int
}

// ResourceUsage tracks current consumption for quota enforcement.
type ResourceUsage struct {
	AgentsCount   int
	CPUCoresUsed  int
	MemoryGBUsed  int
	StorageGBUsed int
	GPUUsed       int
}

type Tenant struct {
	ID          string
	Name        string
	Description string
	Status      TenantStatus
	Plan        string
	Quota       ResourceQuota
	Labels      map[string]string
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *Tenant) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if t.Name == "" {
		return fmt.Errorf("tenant name is required")
	}
	if t.Status != TenantActive && t.Status != TenantSuspended {
		return fmt.Errorf("invalid tenant status: %s", t.Status)
	}
	return nil
}

func (t *Tenant) IsActive() bool {
	return t.Status == TenantActive
}

type TenantMember struct {
	TenantID  string
	UserID    string
	Email     string
	Name      string
	Role      string
	InvitedBy string
}

func (m *TenantMember) Validate() error {
	if m.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if m.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if m.Role == "" {
		return fmt.Errorf("role is required")
	}
	return nil
}
