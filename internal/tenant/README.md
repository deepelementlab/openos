# Multi-Tenancy Module

This module implements multi-tenancy support for Agent OS, including tenant management, resource quotas, and isolation.

## Features

- **Tenant Management**: Create, update, delete, and query tenants
- **Resource Quotas**: Configurable limits per tenant (agents, CPU, memory, storage)
- **Member Management**: Role-based access control within tenants
- **Usage Tracking**: Real-time and historical usage statistics
- **Data Isolation**: Row-level security in PostgreSQL

## Architecture

### Data Model

```
┌─────────────────────────────────────────┐
│               tenants                   │
├─────────────────────────────────────────┤
│ id (PK)                                 │
│ name                                    │
│ status (active/suspended/pending)       │
│ plan (free/basic/pro/enterprise)        │
│ max_agents, max_cpu, max_memory...      │
│ labels (JSONB)                          │
│ metadata (JSONB)                        │
│ created_at, updated_at                  │
└─────────────────────────────────────────┘
                   │
                   │ has many
                   ▼
┌─────────────────────────────────────────┐
│           tenant_members                │
├─────────────────────────────────────────┤
│ id (PK)                                 │
│ tenant_id (FK)                          │
│ user_id                                 │
│ email                                   │
│ role (owner/admin/member/viewer)        │
│ joined_at                               │
└─────────────────────────────────────────┘
                   │
                   │ tracks
                   ▼
┌─────────────────────────────────────────┐
│        tenant_resource_usage            │
├─────────────────────────────────────────┤
│ id (PK)                                 │
│ tenant_id (FK)                          │
│ current_agents, current_cpu...          │
│ period_start                            │
│ total_api_calls                         │
└─────────────────────────────────────────┘
```

## Components

### Repository

- `repository.go` - In-memory implementation for testing
- `postgres_repository.go` - PostgreSQL implementation for production

### Quota Management

- `quota.go` - Quota manager interface and implementation
- Enforces limits on:
  - Number of agents
  - CPU cores
  - Memory (GB)
  - Storage (GB)
  - GPU count
  - API rate limits

### Middleware

- `context.go` - Tenant context extraction from HTTP/gRPC
- `middleware.go` - HTTP middleware for tenant validation

## Configuration

```yaml
tenant:
  enabled: true
  isolated_namespaces: false
  strict_mode: true
  default_quota:
    max_agents: 5
    max_cpu_cores: 4
    max_memory_gb: 8
    max_storage_gb: 50
    max_gpu: 0
    max_requests_per_min: 100
  quota_enforcement:
    enabled: true
    check_interval: 60
    hard_limits: true
```

## Usage

### Creating a Tenant

```go
repo := tenant.NewPostgresTenantRepository(db)

t := &tenant.Tenant{
    ID:   uuid.New().String(),
    Name: "Acme Corp",
    Plan: "pro",
    Quota: tenant.ResourceQuota{
        MaxAgents: 50,
        MaxCPU:    32,
    },
}

err := repo.Create(ctx, t)
```

### Checking Quota

```go
qm := tenant.NewDefaultQuotaManager(logger, repo, cache)

// Check if tenant can create more agents
err := qm.CheckAgentQuota(ctx, tenantID)
if err != nil {
    // Quota exceeded
}

// Increment usage after successful creation
err = qm.IncrementAgentUsage(ctx, tenantID)
```

### Adding Members

```go
member := &tenant.TenantMember{
    TenantID: "tenant-123",
    UserID:   "user@example.com",
    Email:    "user@example.com",
    Role:     "admin",
}

err := repo.AddMember(ctx, member)
```

## Database Migrations

See `internal/database/migrations/`:

- `001_create_tenants_table.sql` - Core tenant table
- `002_create_tenant_quotas_and_members.sql` - Quotas and members
- `003_create_agents_table.sql` - Agent table with tenant FK
- `004_create_audit_and_events_tables.sql` - Audit logging

## Security

1. **Row-Level Security**: All queries filter by tenant_id
2. **Role-Based Access**: Members have roles (owner/admin/member/viewer)
3. **Audit Logging**: All tenant operations are logged
4. **Resource Isolation**: Quotas prevent resource exhaustion

## Testing

Run tests with:

```bash
go test ./internal/tenant/...
```

## Integration

The tenant module integrates with:

- **gRPC Interceptor**: Extracts tenant from context
- **Quota Interceptor**: Enforces quotas on API calls
- **Agent Repository**: Multi-tenant agent queries
- **Audit System**: Logs all tenant operations
