package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostgresTenantRepository implements TenantRepository using PostgreSQL
type PostgresTenantRepository struct {
	db *sqlx.DB
}

// TenantRecord represents a tenant database record
type TenantRecord struct {
	ID               string         `db:"id"`
	Name             string         `db:"name"`
	Description      sql.NullString `db:"description"`
	Status           string         `db:"status"`
	Plan             string         `db:"plan"`
	MaxAgents        int            `db:"max_agents"`
	MaxCPUCores      int            `db:"max_cpu_cores"`
	MaxMemoryGB      int            `db:"max_memory_gb"`
	MaxStorageGB     int            `db:"max_storage_gb"`
	MaxGPU           int            `db:"max_gpu"`
	MaxRequestsPerMin int           `db:"max_requests_per_min"`
	LabelsJSON       string         `db:"labels"`
	MetadataJSON     string         `db:"metadata"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
	SuspendedAt      sql.NullTime   `db:"suspended_at"`
	SuspendedReason  sql.NullString `db:"suspended_reason"`
}

// TenantMemberRecord represents a tenant member database record
type TenantMemberRecord struct {
	ID        string         `db:"id"`
	TenantID  string         `db:"tenant_id"`
	UserID    string         `db:"user_id"`
	Email     string         `db:"email"`
	Name      sql.NullString `db:"name"`
	Role      string         `db:"role"`
	InvitedBy sql.NullString `db:"invited_by"`
	JoinedAt  time.Time      `db:"joined_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// NewPostgresTenantRepository creates a new PostgreSQL tenant repository
func NewPostgresTenantRepository(db *sqlx.DB) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

// Create creates a new tenant
func (r *PostgresTenantRepository) Create(ctx context.Context, t *Tenant) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO tenants (
			id, name, description, status, plan,
			max_agents, max_cpu_cores, max_memory_gb, max_storage_gb, max_gpu, max_requests_per_min,
			labels, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		t.ID,
		t.Name,
		sql.NullString{String: t.Description, Valid: t.Description != ""},
		string(t.Status),
		t.Plan,
		t.Quota.MaxAgents,
		t.Quota.MaxCPU,
		t.Quota.MaxMemoryGB,
		t.Quota.MaxStorageGB,
		t.Quota.MaxGPU,
		t.Quota.MaxRequestsPerMin,
		labelsJSON,
		metadataJSON,
		t.CreatedAt,
		t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// Get retrieves a tenant by ID
func (r *PostgresTenantRepository) Get(ctx context.Context, id string) (*Tenant, error) {
	query := `
		SELECT 
			id, name, description, status, plan,
			max_agents, max_cpu_cores, max_memory_gb, max_storage_gb, max_gpu, max_requests_per_min,
			labels, metadata, created_at, updated_at, suspended_at, suspended_reason
		FROM tenants
		WHERE id = $1
	`

	var record TenantRecord
	if err := r.db.GetContext(ctx, &record, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return r.recordToTenant(&record)
}

// Update updates an existing tenant
func (r *PostgresTenantRepository) Update(ctx context.Context, t *Tenant) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE tenants SET
			name = $2,
			description = $3,
			status = $4,
			plan = $5,
			max_agents = $6,
			max_cpu_cores = $7,
			max_memory_gb = $8,
			max_storage_gb = $9,
			max_gpu = $10,
			max_requests_per_min = $11,
			labels = $12,
			metadata = $13,
			updated_at = $14,
			suspended_at = $15,
			suspended_reason = $16
		WHERE id = $1
	`

	var suspendedAt sql.NullTime
	if t.Status == TenantSuspended && !t.UpdatedAt.IsZero() {
		suspendedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		t.ID,
		t.Name,
		sql.NullString{String: t.Description, Valid: t.Description != ""},
		string(t.Status),
		t.Plan,
		t.Quota.MaxAgents,
		t.Quota.MaxCPU,
		t.Quota.MaxMemoryGB,
		t.Quota.MaxStorageGB,
		t.Quota.MaxGPU,
		t.Quota.MaxRequestsPerMin,
		labelsJSON,
		metadataJSON,
		t.UpdatedAt,
		suspendedAt,
		sql.NullString{Valid: false}, // suspended_reason not stored in Tenant struct
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// Delete deletes a tenant
func (r *PostgresTenantRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// List retrieves all tenants
func (r *PostgresTenantRepository) List(ctx context.Context) ([]*Tenant, error) {
	query := `
		SELECT 
			id, name, description, status, plan,
			max_agents, max_cpu_cores, max_memory_gb, max_storage_gb, max_gpu, max_requests_per_min,
			labels, metadata, created_at, updated_at, suspended_at, suspended_reason
		FROM tenants
		ORDER BY created_at DESC
	`

	var records []TenantRecord
	if err := r.db.SelectContext(ctx, &records, query); err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	tenants := make([]*Tenant, len(records))
	for i, record := range records {
		t, err := r.recordToTenant(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record %d: %w", i, err)
		}
		tenants[i] = t
	}

	return tenants, nil
}

// AddMember adds a member to a tenant
func (r *PostgresTenantRepository) AddMember(ctx context.Context, m *TenantMember) error {
	query := `
		INSERT INTO tenant_members (
			tenant_id, user_id, email, role, joined_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		m.TenantID,
		m.UserID,
		m.Email,
		m.Role,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// RemoveMember removes a member from a tenant
func (r *PostgresTenantRepository) RemoveMember(ctx context.Context, tenantID, userID string) error {
	query := `DELETE FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// GetMembers retrieves all members of a tenant
func (r *PostgresTenantRepository) GetMembers(ctx context.Context, tenantID string) ([]*TenantMember, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, email, name, role, invited_by, joined_at, updated_at
		FROM tenant_members
		WHERE tenant_id = $1
		ORDER BY joined_at ASC
	`

	var records []TenantMemberRecord
	if err := r.db.SelectContext(ctx, &records, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	members := make([]*TenantMember, len(records))
	for i, record := range records {
		members[i] = r.recordToMember(&record)
	}

	return members, nil
}

// Helper methods

func (r *PostgresTenantRepository) recordToTenant(record *TenantRecord) (*Tenant, error) {
	// Parse labels
	labels := make(map[string]string)
	if record.LabelsJSON != "" {
		if err := json.Unmarshal([]byte(record.LabelsJSON), &labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
	}

	// Parse metadata
	metadata := make(map[string]string)
	if record.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(record.MetadataJSON), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	t := &Tenant{
		ID:     record.ID,
		Name:   record.Name,
		Status: TenantStatus(record.Status),
		Plan:   record.Plan,
		Quota: ResourceQuota{
			MaxAgents:        record.MaxAgents,
			MaxCPU:           record.MaxCPUCores,
			MaxMemoryGB:      record.MaxMemoryGB,
			MaxStorageGB:     record.MaxStorageGB,
			MaxGPU:           record.MaxGPU,
			MaxRequestsPerMin: record.MaxRequestsPerMin,
		},
		Labels:    labels,
		Metadata:  metadata,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}

	if record.Description.Valid {
		t.Description = record.Description.String
	}

	return t, nil
}

func (r *PostgresTenantRepository) recordToMember(record *TenantMemberRecord) *TenantMember {
	m := &TenantMember{
		TenantID: record.TenantID,
		UserID:   record.UserID,
		Email:    record.Email,
		Role:     record.Role,
	}

	if record.Name.Valid {
		m.Name = record.Name.String
	}

	if record.InvitedBy.Valid {
		m.InvitedBy = record.InvitedBy.String
	}

	return m
}
