package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agentos/aos/internal/tenant"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for agent storage
type Repository interface {
	Create(ctx context.Context, agent *Agent) error
	Get(ctx context.Context, id string) (*Agent, error)
	GetByTenant(ctx context.Context, tenantID string, id string) (*Agent, error)
	List(ctx context.Context, tenantID string, filter *ListFilter) ([]*Agent, int, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string) error
	DeleteByTenant(ctx context.Context, tenantID string, id string) error
	UpdateStatus(ctx context.Context, id string, status string, message string) error
	ListByNode(ctx context.Context, nodeID string) ([]*Agent, error)
}

// Agent represents an agent entity
type Agent struct {
	ID                string            `db:"id" json:"id"`
	TenantID          string            `db:"tenant_id" json:"tenant_id"`
	Name              string            `db:"name" json:"name"`
	Image             string            `db:"image" json:"image"`
	Runtime           string            `db:"runtime" json:"runtime"`
	Status            string            `db:"status" json:"status"`
	NodeID            sql.NullString    `db:"node_id" json:"node_id,omitempty"`
	CPURequest        sql.NullString    `db:"cpu_request" json:"cpu_request,omitempty"`
	MemoryRequest     sql.NullString    `db:"memory_request" json:"memory_request,omitempty"`
	StorageRequest    sql.NullString    `db:"storage_request" json:"storage_request,omitempty"`
	GPURequest        sql.NullString    `db:"gpu_request" json:"gpu_request,omitempty"`
	CPULimit          sql.NullString    `db:"cpu_limit" json:"cpu_limit,omitempty"`
	MemoryLimit       sql.NullString    `db:"memory_limit" json:"memory_limit,omitempty"`
	EnvironmentJSON   string            `db:"environment" json:"-"`
	LabelsJSON        string            `db:"labels" json:"-"`
	AnnotationsJSON   string            `db:"annotations" json:"-"`
	SandboxType       string            `db:"sandbox_type" json:"sandbox_type"`
	RunAsUser         sql.NullInt64     `db:"run_as_user" json:"run_as_user,omitempty"`
	RunAsGroup        sql.NullInt64     `db:"run_as_group" json:"run_as_group,omitempty"`
	ReadOnlyRootFS    bool              `db:"read_only_root_fs" json:"read_only_root_fs"`
	AllowPrivEsc      bool              `db:"allow_privilege_escalation" json:"allow_privilege_escalation"`
	SeccompProfile    sql.NullString    `db:"seccomp_profile" json:"seccomp_profile,omitempty"`
	CapabilitiesJSON  string            `db:"capabilities" json:"-"`
	AllowInbound      bool              `db:"allow_inbound" json:"allow_inbound"`
	AllowOutbound     bool              `db:"allow_outbound" json:"allow_outbound"`
	InboundPortsJSON  string            `db:"inbound_ports" json:"-"`
	OutboundHostsJSON string            `db:"outbound_hosts" json:"-"`
	Priority          int               `db:"priority" json:"priority"`
	RestartCount      int               `db:"restart_count" json:"restart_count"`
	LastError         sql.NullString    `db:"last_error" json:"last_error,omitempty"`
	MetadataJSON      string            `db:"metadata" json:"-"`
	CreatedAt         time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time         `db:"updated_at" json:"updated_at"`
	StartedAt         sql.NullTime      `db:"started_at" json:"started_at,omitempty"`
	StoppedAt         sql.NullTime      `db:"stopped_at" json:"stopped_at,omitempty"`
	DeletedAt         sql.NullTime      `db:"deleted_at" json:"deleted_at,omitempty"`
	CreatedBy         sql.NullString    `db:"created_by" json:"created_by,omitempty"`
	
	// Parsed JSON fields (not stored in DB)
	Environment   map[string]string `db:"-" json:"environment,omitempty"`
	Labels        map[string]string `db:"-" json:"labels,omitempty"`
	Annotations   map[string]string `db:"-" json:"annotations,omitempty"`
	Capabilities  []string          `db:"-" json:"capabilities,omitempty"`
	InboundPorts  []int             `db:"-" json:"inbound_ports,omitempty"`
	OutboundHosts []string          `db:"-" json:"outbound_hosts,omitempty"`
	Metadata      map[string]string `db:"-" json:"metadata,omitempty"`
}

// ListFilter defines filters for listing agents
type ListFilter struct {
	Status   string
	Runtime  string
	Labels   map[string]string
	Page     int
	PageSize int
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL agent repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create creates a new agent
func (r *PostgresRepository) Create(ctx context.Context, agent *Agent) error {
	// Marshal JSON fields
	envJSON, _ := json.Marshal(agent.Environment)
	labelsJSON, _ := json.Marshal(agent.Labels)
	annotationsJSON, _ := json.Marshal(agent.Annotations)
	capsJSON, _ := json.Marshal(agent.Capabilities)
	portsJSON, _ := json.Marshal(agent.InboundPorts)
	hostsJSON, _ := json.Marshal(agent.OutboundHosts)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	query := `
		INSERT INTO agents (
			id, tenant_id, name, image, runtime, status, node_id,
			cpu_request, memory_request, storage_request, gpu_request,
			cpu_limit, memory_limit,
			environment, labels, annotations,
			sandbox_type, run_as_user, run_as_group, read_only_root_fs,
			allow_privilege_escalation, seccomp_profile, capabilities,
			allow_inbound, allow_outbound, inbound_ports, outbound_hosts,
			priority, restart_count, last_error, metadata,
			created_at, updated_at, started_at, stopped_at, deleted_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13,
			$14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23,
			$24, $25, $26, $27,
			$28, $29, $30, $31,
			$32, $33, $34, $35, $36, $37
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		agent.ID, agent.TenantID, agent.Name, agent.Image, agent.Runtime, agent.Status, agent.NodeID,
		agent.CPURequest, agent.MemoryRequest, agent.StorageRequest, agent.GPURequest,
		agent.CPULimit, agent.MemoryLimit,
		envJSON, labelsJSON, annotationsJSON,
		agent.SandboxType, agent.RunAsUser, agent.RunAsGroup, agent.ReadOnlyRootFS,
		agent.AllowPrivEsc, agent.SeccompProfile, capsJSON,
		agent.AllowInbound, agent.AllowOutbound, portsJSON, hostsJSON,
		agent.Priority, agent.RestartCount, agent.LastError, metadataJSON,
		agent.CreatedAt, agent.UpdatedAt, agent.StartedAt, agent.StoppedAt, agent.DeletedAt, agent.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// Get retrieves an agent by ID
func (r *PostgresRepository) Get(ctx context.Context, id string) (*Agent, error) {
	query := `
		SELECT * FROM agents
		WHERE id = $1 AND deleted_at IS NULL
	`

	var agent Agent
	if err := r.db.GetContext(ctx, &agent, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	r.parseJSONFields(&agent)
	return &agent, nil
}

// GetByTenant retrieves an agent by ID within a tenant
func (r *PostgresRepository) GetByTenant(ctx context.Context, tenantID, id string) (*Agent, error) {
	query := `
		SELECT * FROM agents
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	var agent Agent
	if err := r.db.GetContext(ctx, &agent, query, id, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	r.parseJSONFields(&agent)
	return &agent, nil
}

// List lists agents for a tenant with filtering
func (r *PostgresRepository) List(ctx context.Context, tenantID string, filter *ListFilter) ([]*Agent, int, error) {
	// Build query
	whereClause := "tenant_id = $1 AND deleted_at IS NULL"
	args := []interface{}{tenantID}
	argCount := 1

	if filter.Status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}

	if filter.Runtime != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND runtime = $%d", argCount)
		args = append(args, filter.Runtime)
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agents WHERE %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	// Data query with pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := fmt.Sprintf(`
		SELECT * FROM agents
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	args = append(args, pageSize, (page-1)*pageSize)

	var agents []Agent
	if err := r.db.SelectContext(ctx, &agents, query, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to list agents: %w", err)
	}

	// Parse JSON fields
	result := make([]*Agent, len(agents))
	for i := range agents {
		r.parseJSONFields(&agents[i])
		result[i] = &agents[i]
	}

	return result, total, nil
}

// Update updates an agent
func (r *PostgresRepository) Update(ctx context.Context, agent *Agent) error {
	// Marshal JSON fields
	envJSON, _ := json.Marshal(agent.Environment)
	labelsJSON, _ := json.Marshal(agent.Labels)
	annotationsJSON, _ := json.Marshal(agent.Annotations)
	capsJSON, _ := json.Marshal(agent.Capabilities)
	portsJSON, _ := json.Marshal(agent.InboundPorts)
	hostsJSON, _ := json.Marshal(agent.OutboundHosts)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	query := `
		UPDATE agents SET
			name = $2,
			image = $3,
			runtime = $4,
			status = $5,
			cpu_request = $6,
			memory_request = $7,
			storage_request = $8,
			gpu_request = $9,
			cpu_limit = $10,
			memory_limit = $11,
			environment = $12,
			labels = $13,
			annotations = $14,
			sandbox_type = $15,
			run_as_user = $16,
			run_as_group = $17,
			read_only_root_fs = $18,
			allow_privilege_escalation = $19,
			seccomp_profile = $20,
			capabilities = $21,
			allow_inbound = $22,
			allow_outbound = $23,
			inbound_ports = $24,
			outbound_hosts = $25,
			priority = $26,
			metadata = $27,
			updated_at = $28
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		agent.ID,
		agent.Name,
		agent.Image,
		agent.Runtime,
		agent.Status,
		agent.CPURequest,
		agent.MemoryRequest,
		agent.StorageRequest,
		agent.GPURequest,
		agent.CPULimit,
		agent.MemoryLimit,
		envJSON,
		labelsJSON,
		annotationsJSON,
		agent.SandboxType,
		agent.RunAsUser,
		agent.RunAsGroup,
		agent.ReadOnlyRootFS,
		agent.AllowPrivEsc,
		agent.SeccompProfile,
		capsJSON,
		agent.AllowInbound,
		agent.AllowOutbound,
		portsJSON,
		hostsJSON,
		agent.Priority,
		metadataJSON,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// Delete soft-deletes an agent
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE agents SET
			deleted_at = $2,
			status = 'deleted',
			updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// DeleteByTenant soft-deletes an agent within a tenant
func (r *PostgresRepository) DeleteByTenant(ctx context.Context, tenantID, id string) error {
	query := `
		UPDATE agents SET
			deleted_at = $3,
			status = 'deleted',
			updated_at = $3
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, tenantID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// UpdateStatus updates agent status
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status string, message string) error {
	query := `
		UPDATE agents SET
			status = $2,
			last_error = $3,
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, id, status, sql.NullString{String: message, Valid: message != ""}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// ListByNode lists agents on a specific node
func (r *PostgresRepository) ListByNode(ctx context.Context, nodeID string) ([]*Agent, error) {
	query := `
		SELECT * FROM agents
		WHERE node_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var agents []Agent
	if err := r.db.SelectContext(ctx, &agents, query, nodeID); err != nil {
		return nil, fmt.Errorf("failed to list agents by node: %w", err)
	}

	result := make([]*Agent, len(agents))
	for i := range agents {
		r.parseJSONFields(&agents[i])
		result[i] = &agents[i]
	}

	return result, nil
}

// parseJSONFields parses JSON fields from the database record
func (r *PostgresRepository) parseJSONFields(agent *Agent) {
	if agent.EnvironmentJSON != "" {
		json.Unmarshal([]byte(agent.EnvironmentJSON), &agent.Environment)
	}
	if agent.LabelsJSON != "" {
		json.Unmarshal([]byte(agent.LabelsJSON), &agent.Labels)
	}
	if agent.AnnotationsJSON != "" {
		json.Unmarshal([]byte(agent.AnnotationsJSON), &agent.Annotations)
	}
	if agent.CapabilitiesJSON != "" {
		json.Unmarshal([]byte(agent.CapabilitiesJSON), &agent.Capabilities)
	}
	if agent.InboundPortsJSON != "" {
		json.Unmarshal([]byte(agent.InboundPortsJSON), &agent.InboundPorts)
	}
	if agent.OutboundHostsJSON != "" {
		json.Unmarshal([]byte(agent.OutboundHostsJSON), &agent.OutboundHosts)
	}
	if agent.MetadataJSON != "" {
		json.Unmarshal([]byte(agent.MetadataJSON), &agent.Metadata)
	}
}

// Ensure PostgresRepository implements Repository interface
var _ Repository = (*PostgresRepository)(nil)

// Helper to get tenant from context
type tenantContextKey string

const tenantIDKey tenantContextKey = "tenant_id"

// WithTenant adds tenant ID to context
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantFromContext extracts tenant ID from context
func TenantFromContext(ctx context.Context) (string, bool) {
	return tenant.TenantFromContext(ctx)
}
