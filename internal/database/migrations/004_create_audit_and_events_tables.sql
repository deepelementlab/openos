-- Migration: 004_create_audit_and_events_tables
-- Description: Create audit logs and events tables for monitoring and compliance
-- Created at: 2024-01-01

BEGIN;

-- Create audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id VARCHAR(255),
    
    -- Action details
    action VARCHAR(100) NOT NULL,  -- create, update, delete, start, stop, etc.
    resource_type VARCHAR(100) NOT NULL,  -- agent, tenant, node, etc.
    resource_id VARCHAR(255),
    
    -- Change tracking
    old_values JSONB,
    new_values JSONB,
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    api_endpoint VARCHAR(500),
    http_method VARCHAR(10),
    
    -- Result
    result VARCHAR(50) NOT NULL DEFAULT 'success' CHECK (result IN ('success', 'failure', 'partial')),
    error_code VARCHAR(100),
    error_message TEXT,
    
    -- Timing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration_ms INT,  -- Request duration
    
    -- Partitioning support (for future)
    created_date DATE NOT NULL DEFAULT CURRENT_DATE
);

-- Create indexes for audit log queries
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_result ON audit_logs(result) WHERE result = 'failure';

-- Create composite index for common tenant+time queries
CREATE INDEX idx_audit_logs_tenant_time ON audit_logs(tenant_id, created_at DESC);

-- Create GIN indexes for JSON columns
CREATE INDEX idx_audit_logs_old_values ON audit_logs USING GIN(old_values);
CREATE INDEX idx_audit_logs_new_values ON audit_logs USING GIN(new_values);

-- Create events table for real-time event streaming
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,  -- agent.created, agent.started, tenant.suspended, etc.
    resource_type VARCHAR(100) NOT NULL,  -- agent, tenant, node, etc.
    resource_id VARCHAR(255),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    
    -- Event details
    severity VARCHAR(50) NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    message TEXT,
    metadata JSONB DEFAULT '{}',
    
    -- Event status for processing
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    processed_at TIMESTAMP WITH TIME ZONE,
    processor_id VARCHAR(255),  -- ID of the processor that handled this event
    
    -- Timing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- TTL support (for cleanup of old events)
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP + INTERVAL '30 days'
);

-- Create indexes for event queries
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_resource ON events(resource_type, resource_id);
CREATE INDEX idx_events_tenant_id ON events(tenant_id);
CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_severity ON events(severity);
CREATE INDEX idx_events_status ON events(status) WHERE status = 'pending';
CREATE INDEX idx_events_created_at ON events(created_at DESC);
CREATE INDEX idx_events_expires_at ON events(expires_at);

-- Create composite indexes for common queries
CREATE INDEX idx_events_tenant_type ON events(tenant_id, event_type);
CREATE INDEX idx_events_resource_created ON events(resource_type, resource_id, created_at DESC);

-- Create GIN index for metadata
CREATE INDEX idx_events_metadata ON events USING GIN(metadata);

-- Create alerts table for system alerting
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('info', 'warning', 'critical', 'emergency')),
    status VARCHAR(50) NOT NULL DEFAULT 'firing' CHECK (status IN ('firing', 'acknowledged', 'resolved', 'silenced')),
    
    -- Alert source
    source VARCHAR(100) NOT NULL,  -- prometheus, custom, etc.
    service VARCHAR(100),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Alert details
    labels JSONB DEFAULT '{}',
    annotations JSONB DEFAULT '{}',
    value DOUBLE PRECISION,  -- Metric value that triggered alert
    threshold VARCHAR(255),  -- Threshold condition (e.g., "> 80")
    
    -- Acknowledgment
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_comment TEXT,
    
    -- Resolution
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_comment TEXT,
    
    -- Timing
    fired_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE  -- When to auto-resolve
);

-- Create indexes for alerts
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_service ON alerts(service);
CREATE INDEX idx_alerts_tenant_id ON alerts(tenant_id);
CREATE INDEX idx_alerts_fired_at ON alerts(fired_at DESC);
CREATE INDEX idx_alerts_status_severity ON alerts(status, severity) WHERE status = 'firing';

-- Create GIN indexes for labels and annotations
CREATE INDEX idx_alerts_labels ON alerts USING GIN(labels);
CREATE INDEX idx_alerts_annotations ON alerts USING GIN(annotations);

-- Create function to auto-cleanup old events
CREATE OR REPLACE FUNCTION cleanup_expired_events()
RETURNS void AS $$
BEGIN
    DELETE FROM events WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Create function for updating alert updated_at
CREATE OR REPLACE FUNCTION update_alerts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_alerts_updated_at();

-- Create function to update agent metrics on status change
CREATE OR REPLACE FUNCTION log_agent_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO events (event_type, resource_type, resource_id, tenant_id, 
                          severity, message, metadata, status)
        VALUES (
            'agent.status_changed',
            'agent',
            NEW.id::TEXT,
            NEW.tenant_id,
            CASE 
                WHEN NEW.status = 'error' THEN 'error'
                WHEN NEW.status = 'running' THEN 'info'
                ELSE 'warning'
            END,
            format('Agent %s status changed from %s to %s', NEW.name, OLD.status, NEW.status),
            jsonb_build_object(
                'old_status', OLD.status,
                'new_status', NEW.status,
                'agent_name', NEW.name
            ),
            'completed'
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agent_status_change
    AFTER UPDATE OF status ON agents
    FOR EACH ROW
    EXECUTE FUNCTION log_agent_status_change();

-- Create views for common queries
CREATE OR REPLACE VIEW active_alerts AS
SELECT * FROM alerts 
WHERE status = 'firing' 
AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

CREATE OR REPLACE VIEW recent_events AS
SELECT * FROM events 
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours';

-- Add comments
COMMENT ON TABLE audit_logs IS 'Audit trail for all significant operations';
COMMENT ON TABLE events IS 'System events for real-time monitoring and streaming';
COMMENT ON TABLE alerts IS 'Active and historical alerts from monitoring system';
COMMENT ON VIEW active_alerts IS 'View showing only currently firing alerts';
COMMENT ON VIEW recent_events IS 'View showing events from last 24 hours';

COMMIT;
