package tenant

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTenantValidate_Valid(t *testing.T) {
	tenant := &Tenant{
		ID:     "t-1",
		Name:   "Acme Corp",
		Status: TenantActive,
	}
	assert.NoError(t, tenant.Validate())
}

func TestTenantValidate_MissingID(t *testing.T) {
	tenant := &Tenant{
		Name:   "Acme Corp",
		Status: TenantActive,
	}
	err := tenant.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestTenantValidate_MissingName(t *testing.T) {
	tenant := &Tenant{
		ID:     "t-1",
		Status: TenantActive,
	}
	err := tenant.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant name is required")
}

func TestTenantValidate_InvalidStatus(t *testing.T) {
	tenant := &Tenant{
		ID:     "t-1",
		Name:   "Acme Corp",
		Status: TenantStatus("unknown"),
	}
	err := tenant.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant status")
}

func TestTenantIsActive(t *testing.T) {
	assert.True(t, (&Tenant{Status: TenantActive}).IsActive())
	assert.False(t, (&Tenant{Status: TenantSuspended}).IsActive())
}

func TestTenantStatusConstants(t *testing.T) {
	assert.Equal(t, TenantStatus("active"), TenantActive)
	assert.Equal(t, TenantStatus("suspended"), TenantSuspended)
}

func TestResourceQuota_Fields(t *testing.T) {
	q := ResourceQuota{MaxAgents: 10, MaxCPU: 4, MaxMemoryGB: 16}
	assert.Equal(t, 10, q.MaxAgents)
	assert.Equal(t, 4, q.MaxCPU)
	assert.Equal(t, 16, q.MaxMemoryGB)
}

func TestTenantMemberValidate_Valid(t *testing.T) {
	m := &TenantMember{TenantID: "t-1", UserID: "u-1", Role: "admin"}
	assert.NoError(t, m.Validate())
}

func TestTenantMemberValidate_MissingTenantID(t *testing.T) {
	m := &TenantMember{UserID: "u-1", Role: "admin"}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestTenantMemberValidate_MissingUserID(t *testing.T) {
	m := &TenantMember{TenantID: "t-1", Role: "admin"}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestTenantMemberValidate_MissingRole(t *testing.T) {
	m := &TenantMember{TenantID: "t-1", UserID: "u-1"}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role is required")
}

func TestTenant_LabelsMap(t *testing.T) {
	tn := &Tenant{
		ID:     "t-1",
		Name:   "Test",
		Status: TenantActive,
		Labels: map[string]string{"env": "prod", "region": "us-east"},
	}
	assert.Equal(t, "prod", tn.Labels["env"])
	assert.Equal(t, "us-east", tn.Labels["region"])
}

func TestTenant_Timestamps(t *testing.T) {
	now := time.Now()
	tn := &Tenant{
		ID:        "t-1",
		Name:      "Test",
		Status:    TenantActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	assert.Equal(t, now, tn.CreatedAt)
	assert.Equal(t, now, tn.UpdatedAt)
}

func TestNewTenant(t *testing.T) {
	tn := NewTenant("t-1", "Acme Corp", "enterprise")
	assert.Equal(t, "t-1", tn.ID)
	assert.Equal(t, "Acme Corp", tn.Name)
	assert.Equal(t, TenantActive, tn.Status)
	assert.Equal(t, "enterprise", tn.Plan)
	assert.NotNil(t, tn.Labels)
	assert.WithinDuration(t, time.Now(), tn.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), tn.UpdatedAt, time.Second)
}
