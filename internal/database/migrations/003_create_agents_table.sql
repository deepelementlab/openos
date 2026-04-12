-- Migration: 003_create_agents_table
-- Description: Create agents table with tenant isolation
-- Created at: 2024-01-01

BEGIN;

-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Basic info
    name VARCHAR(255) NOT NULL,
    image VARCHAR(500) NOT NULL,
    runtime VARCHAR(100) NOT NULL DEFAULT 'containerd',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'creating', 'running', 'stopping', 'stopped', 'error', 'deleting', 'deleted')),
    
    -- Resource requirements
    cpu_request VARCHAR(50),
    memory_request VARCHAR(50),
    storage_request VARCHAR(50),
    gpu_request VARCHAR(50),
    cpu_limit VARCHAR(50),
    memory_limit VARCHAR(50),
    
    -- Environment and configuration
    environment JSONB DEFAULT '{}',
    labels JSONB DEFAULT '{}',
    annotations JSONB DEFAULT '{}',
    
    -- Security context
    run_as_user BIGINT,
    run_as_group BIGINT,
    read_only_root_fs BOOLEAN DEFAULT FALSE,
    allow_privilege_escalation BOOLEAN DEFAULT FALSE,
    sandbox_type VARCHAR(50) DEFAULT 'containerd',
    seccomp_profile VARCHAR(255),
    capabilities_add JSONB DEFAULT '[]',
    capabilities_drop JSONB DEFAULT '[]',
    
    -- Network policy
    allow_inbound BOOLEAN DEFAULT FALSE,
    allow_outbound BOOLEAN DEFAULT TRUE,
    inbound_ports JSONB DEFAULT '[]',
    outbound_hosts JSONB DEFAULT '[]',
    network_ingress_class VARCHAR(100),
    
    -- Scheduling
    node_id UUID,
    priority INT DEFAULT 0,
    
    -- Metadata
    restart_count INT DEFAULT 0,
    last_error TEXT,
    metadata JSONB DEFAULT '{}',
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    stopped_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    
    -- Constraints
    CONSTRAINT agents_name_not_empty CHECK (name <> ''),
    CONSTRAINT agents_image_not_empty CHECK (image <> ''),
    CONSTRAINT agents_restart_count_positive CHECK (restart_count >= 0)
);

-- Create indexes for common queries
CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_runtime ON agents(runtime);
CREATE INDEX idx_agents_node_id ON agents(node_id);
CREATE INDEX idx_agents_created_at ON agents(created_at DESC);
CREATE INDEX idx_agents_name ON agents(name);

-- Create composite index for tenant-scoped status queries
CREATE INDEX idx_agents_tenant_status ON agents(tenant_id, status);

-- Create GIN indexes for JSON fields
CREATE INDEX idx_agents_labels ON agents USING GIN(labels);
CREATE INDEX idx_agents_annotations ON agents USING GIN(annotations);
CREATE INDEX idx_agents_environment ON agents USING GIN(environment);

-- Create partial index for active agents (not deleted)
CREATE INDEX idx_agents_active ON agents(tenant_id, status) 
    WHERE deleted_at IS NULL;

-- Create trigger for updating updated_at
CREATE OR REPLACE FUNCTION update_agents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_agents_updated_at();

-- Create function to update tenant resource usage when agent changes
CREATE OR REPLACE FUNCTION update_tenant_resource_usage_from_agent()
RETURNS TRIGGER AS $$
BEGIN
    -- Update current agent count for the tenant
    IF TG_OP = 'INSERT' AND NEW.deleted_at IS NULL THEN
        INSERT INTO tenant_resource_usage (tenant_id, period_start, current_agents)
        VALUES (NEW.tenant_id, DATE_TRUNC('hour', CURRENT_TIMESTAMP), 1)
        ON CONFLICT (tenant_id, period_start) 
        DO UPDATE SET 
            current_agents = tenant_resource_usage.current_agents + 1,
            updated_at = CURRENT_TIMESTAMP;
    ELSIF TG_OP = 'UPDATE' THEN
        -- If agent is being deleted
        IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
            UPDATE tenant_resource_usage 
            SET current_agents = GREATEST(current_agents - 1, 0),
                updated_at = CURRENT_TIMESTAMP
            WHERE tenant_id = NEW.tenant_id 
            AND period_start = DATE_TRUNC('hour', CURRENT_TIMESTAMP);
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agents_update_tenant_usage
    AFTER INSERT OR UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_resource_usage_from_agent();

-- Create view for agent summary with tenant info
CREATE OR REPLACE VIEW agent_summary AS
SELECT 
    a.id,
    a.tenant_id,
    t.name AS tenant_name,
    a.name,
    a.image,
    a.runtime,
    a.status,
    a.node_id,
    a.priority,
    a.restart_count,
    a.created_at,
    a.updated_at,
    a.started_at,
    a.stopped_at,
    CASE 
        WHEN a.started_at IS NOT NULL AND a.stopped_at IS NULL 
        THEN EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - a.started_at))::BIGINT
        WHEN a.started_at IS NOT NULL AND a.stopped_at IS NOT NULL
        THEN EXTRACT(EPOCH FROM (a.stopped_at - a.started_at))::BIGINT
        ELSE 0
    END AS uptime_seconds
FROM agents a
JOIN tenants t ON a.tenant_id = t.id
WHERE a.deleted_at IS NULL;

-- Add comments
COMMENT ON TABLE agents IS 'Stores agent instances with tenant isolation';
COMMENT ON COLUMN agents.tenant_id IS 'Foreign key to tenants table for isolation';
COMMENT ON COLUMN agents.status IS 'Agent lifecycle status';
COMMENT ON COLUMN agents.sandbox_type IS 'Security sandbox type: containerd, gvisor, or kata';
COMMENT ON VIEW agent_summary IS 'Convenience view showing agent details with tenant name';

COMMIT;
