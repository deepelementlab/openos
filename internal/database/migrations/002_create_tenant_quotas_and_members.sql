-- Migration: 002_create_tenant_quotas_and_members
-- Description: Create tenant quota tracking and member management tables
-- Created at: 2024-01-01

BEGIN;

-- Create tenant resource usage tracking table
CREATE TABLE IF NOT EXISTS tenant_resource_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Current usage counters
    current_agents INT NOT NULL DEFAULT 0,
    current_cpu_cores INT NOT NULL DEFAULT 0,
    current_memory_gb INT NOT NULL DEFAULT 0,
    current_storage_gb INT NOT NULL DEFAULT 0,
    current_gpu INT NOT NULL DEFAULT 0,
    
    -- Period tracking
    period_start TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    period_end TIMESTAMP WITH TIME ZONE,
    
    -- Aggregated metrics
    total_api_calls BIGINT NOT NULL DEFAULT 0,
    total_requests BIGINT NOT NULL DEFAULT 0,
    avg_response_time_ms DOUBLE PRECISION DEFAULT 0.0,
    error_rate DOUBLE PRECISION DEFAULT 0.0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT tenant_resource_usage_tenant_period_unique UNIQUE (tenant_id, period_start),
    CONSTRAINT tenant_resource_usage_agents_positive CHECK (current_agents >= 0),
    CONSTRAINT tenant_resource_usage_cpu_positive CHECK (current_cpu_cores >= 0),
    CONSTRAINT tenant_resource_usage_memory_positive CHECK (current_memory_gb >= 0)
);

-- Create index for fast tenant usage lookups
CREATE INDEX idx_tenant_resource_usage_tenant_id ON tenant_resource_usage(tenant_id);
CREATE INDEX idx_tenant_resource_usage_period ON tenant_resource_usage(period_start DESC);

-- Create tenant members table
CREATE TABLE IF NOT EXISTS tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    
    invited_by VARCHAR(255),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT tenant_members_tenant_user_unique UNIQUE (tenant_id, user_id),
    CONSTRAINT tenant_members_email_not_empty CHECK (email <> '')
);

-- Create indexes for member lookups
CREATE INDEX idx_tenant_members_tenant_id ON tenant_members(tenant_id);
CREATE INDEX idx_tenant_members_user_id ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_role ON tenant_members(role);
CREATE INDEX idx_tenant_members_email ON tenant_members(email);

-- Create trigger for updating tenant_members updated_at
CREATE OR REPLACE FUNCTION update_tenant_members_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tenant_members_updated_at
    BEFORE UPDATE ON tenant_members
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_members_updated_at();

-- Create trigger for updating tenant_resource_usage updated_at
CREATE OR REPLACE FUNCTION update_tenant_resource_usage_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tenant_resource_usage_updated_at
    BEFORE UPDATE ON tenant_resource_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_resource_usage_updated_at();

-- Create view for tenant quota status
CREATE OR REPLACE VIEW tenant_quota_status AS
SELECT 
    t.id AS tenant_id,
    t.name AS tenant_name,
    t.status,
    t.plan,
    t.max_agents,
    t.max_cpu_cores,
    t.max_memory_gb,
    t.max_storage_gb,
    COALESCE(u.current_agents, 0) AS current_agents,
    COALESCE(u.current_cpu_cores, 0) AS current_cpu_cores,
    COALESCE(u.current_memory_gb, 0) AS current_memory_gb,
    COALESCE(u.current_storage_gb, 0) AS current_storage_gb,
    CASE 
        WHEN t.max_agents > 0 THEN (COALESCE(u.current_agents, 0)::FLOAT / t.max_agents * 100)
        ELSE 0
    END AS agents_usage_percent,
    CASE 
        WHEN t.max_cpu_cores > 0 THEN (COALESCE(u.current_cpu_cores, 0)::FLOAT / t.max_cpu_cores * 100)
        ELSE 0
    END AS cpu_usage_percent,
    CASE 
        WHEN t.max_memory_gb > 0 THEN (COALESCE(u.current_memory_gb, 0)::FLOAT / t.max_memory_gb * 100)
        ELSE 0
    END AS memory_usage_percent,
    CASE 
        WHEN t.max_storage_gb > 0 THEN (COALESCE(u.current_storage_gb, 0)::FLOAT / t.max_storage_gb * 100)
        ELSE 0
    END AS storage_usage_percent,
    GREATEST(
        CASE WHEN t.max_agents > 0 THEN (COALESCE(u.current_agents, 0)::FLOAT / t.max_agents * 100) ELSE 0 END,
        CASE WHEN t.max_cpu_cores > 0 THEN (COALESCE(u.current_cpu_cores, 0)::FLOAT / t.max_cpu_cores * 100) ELSE 0 END,
        CASE WHEN t.max_memory_gb > 0 THEN (COALESCE(u.current_memory_gb, 0)::FLOAT / t.max_memory_gb * 100) ELSE 0 END,
        CASE WHEN t.max_storage_gb > 0 THEN (COALESCE(u.current_storage_gb, 0)::FLOAT / t.max_storage_gb * 100) ELSE 0 END
    ) AS overall_usage_percent,
    u.total_api_calls,
    u.total_requests,
    u.avg_response_time_ms,
    u.error_rate,
    u.updated_at AS usage_updated_at
FROM tenants t
LEFT JOIN LATERAL (
    SELECT * FROM tenant_resource_usage 
    WHERE tenant_id = t.id 
    ORDER BY period_start DESC 
    LIMIT 1
) u ON true;

-- Add comments
COMMENT ON TABLE tenant_resource_usage IS 'Tracks resource usage per tenant for quota enforcement';
COMMENT ON TABLE tenant_members IS 'Manages tenant membership and roles';
COMMENT ON VIEW tenant_quota_status IS 'Computed view showing tenant quota utilization';

COMMIT;
