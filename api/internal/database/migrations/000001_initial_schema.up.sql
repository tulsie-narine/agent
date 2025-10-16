-- +migrate Up
-- Initial schema for inventory agent system

-- Agents table
CREATE TABLE agents (
    device_id UUID PRIMARY KEY,
    org_id BIGINT DEFAULT 1,
    hostname TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'offline')),
    capabilities JSONB,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    auth_token_hash TEXT NOT NULL,
    agent_version TEXT,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for agents
CREATE INDEX idx_agents_org_id ON agents(org_id);
CREATE INDEX idx_agents_last_seen_at ON agents(last_seen_at);
CREATE INDEX idx_agents_status ON agents(status);

-- Telemetry table with partitioning
CREATE TABLE telemetry (
    device_id UUID NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL,
    metrics JSONB,
    tags JSONB,
    seq BIGINT NOT NULL DEFAULT 0,
    server_received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ingestion_id UUID NOT NULL DEFAULT gen_random_uuid(),
    PRIMARY KEY (device_id, collected_at, seq)
) PARTITION BY RANGE (collected_at);

-- Create initial partitions (current day ± 7 days)
DO $$
DECLARE
    start_date DATE := CURRENT_DATE - INTERVAL '7 days';
    end_date DATE := CURRENT_DATE + INTERVAL '7 days';
    partition_date DATE := start_date;
BEGIN
    WHILE partition_date <= end_date LOOP
        EXECUTE format('CREATE TABLE telemetry_y%sm%s PARTITION OF telemetry FOR VALUES FROM (%L) TO (%L)',
            EXTRACT(YEAR FROM partition_date), LPAD(EXTRACT(MONTH FROM partition_date)::TEXT, 2, '0'),
            partition_date, partition_date + INTERVAL '1 day');
        partition_date := partition_date + INTERVAL '1 day';
    END LOOP;
END $$;

-- Create indexes for telemetry
CREATE INDEX idx_telemetry_collected_at ON telemetry(collected_at DESC);
CREATE INDEX idx_telemetry_device_id ON telemetry(device_id);

-- Telemetry latest table
CREATE TABLE telemetry_latest (
    device_id UUID PRIMARY KEY REFERENCES agents(device_id) ON DELETE CASCADE,
    collected_at TIMESTAMPTZ NOT NULL,
    metrics JSONB,
    tags JSONB,
    seq BIGINT NOT NULL DEFAULT 0,
    server_received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Policies table
CREATE TABLE policies (
    policy_id BIGSERIAL PRIMARY KEY,
    device_id UUID REFERENCES agents(device_id) ON DELETE CASCADE,
    group_id BIGINT,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'group', 'device')),
    version INT NOT NULL DEFAULT 1,
    config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for policies
CREATE INDEX idx_policies_device_id ON policies(device_id);
CREATE INDEX idx_policies_group_id ON policies(group_id);
CREATE INDEX idx_policies_scope ON policies(scope);

-- Commands table
CREATE TABLE commands (
    command_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES agents(device_id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    parameters JSONB,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ttl_seconds INT NOT NULL DEFAULT 300,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'executing', 'completed', 'failed', 'expired')),
    result JSONB,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for commands
CREATE INDEX idx_commands_device_id ON commands(device_id);
CREATE INDEX idx_commands_status ON commands(status);

-- Audit log table
CREATE TABLE audit_log (
    log_id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details JSONB
);

-- Create index for audit log
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_commands_updated_at BEFORE UPDATE ON commands FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();