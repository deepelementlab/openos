-- Migration: 001_create_tenants_table
-- Description: Create tenants table for multi-tenancy support
-- Created at: 2024-01-01

BEGIN;

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'pending', 'deactivated')),
    plan VARCHAR(100) NOT NULL DEFAULT 'free',
    
    -- Resource quota columns (flattened from JSON for querying)
    max_agents INT NOT NULL DEFAULT 5,
    max_cpu_cores INT NOT NULL DEFAULT 4,
    max_memory_gb INT NOT NULL DEFAULT 8,
    max_storage_gb INT NOT NULL DEFAULT 50,
    max_gpu INT NOT NULL DEFAULT 0,
    max_requests_per_min INT NOT NULL DEFAULT 100,
    
    -- JSON fields for extensibility
    labels JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    suspended_at TIMESTAMP WITH TIME ZONE,
    suspended_reason TEXT,
    
    -- Constraints
    CONSTRAINT tenants_name_unique UNIQUE (name),
    CONSTRAINT tenants_name_not_empty CHECK (name <> ''),
    CONSTRAINT tenants_max_agents_positive CHECK (max_agents >= 0),
    CONSTRAINT tenants_max_cpu_positive CHECK (max_cpu_cores >= 0),
    CONSTRAINT tenants_max_memory_positive CHECK (max_memory_gb >= 0)
);

-- Create index on tenant status for filtering
CREATE INDEX idx_tenants_status ON tenants(status);

-- Create index on tenant plan for filtering by subscription tier
CREATE INDEX idx_tenants_plan ON tenants(plan);

-- Create index on created_at for sorting
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC);

-- Create GIN index on labels for fast JSON queries
CREATE INDEX idx_tenants_labels ON tenants USING GIN(labels);

-- Create GIN index on metadata for fast JSON queries
CREATE INDEX idx_tenants_metadata ON tenants USING GIN(metadata);

-- Create trigger for updating updated_at timestamp
CREATE OR REPLACE FUNCTION update_tenants_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_tenants_updated_at();

-- Add comment for documentation
COMMENT ON TABLE tenants IS 'Stores tenant information for multi-tenancy support';
COMMENT ON COLUMN tenants.id IS 'Unique tenant identifier';
COMMENT ON COLUMN tenants.status IS 'Tenant status: active, suspended, pending, or deactivated';
COMMENT ON COLUMN tenants.plan IS 'Subscription plan tier (free, basic, pro, enterprise)';

COMMIT;
