# Database Design

## Overview

The system uses PostgreSQL 16+ as the primary database with native partitioning for efficient handling of large-scale telemetry data. The schema is designed for high performance, scalability, and maintainability.

## Database Configuration

### Connection Settings
```sql
-- Recommended PostgreSQL configuration
shared_preload_libraries = 'pg_stat_statements'
max_connections = 200
work_mem = '64MB'
maintenance_work_mem = '256MB'
wal_level = replica
max_wal_senders = 10
```

### Extensions
```sql
-- Required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pg_buffercache";
```

## Schema Design

### Core Tables

#### devices
Stores registered Windows agents and their metadata.

```sql
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hostname VARCHAR(255) NOT NULL,
    os_version VARCHAR(100),
    os_build VARCHAR(50),
    architecture VARCHAR(20),
    domain VARCHAR(255),
    api_key_hash VARCHAR(255) UNIQUE,
    last_seen TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'unknown' CHECK (status IN ('online', 'offline', 'unknown')),
    capabilities JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_devices_hostname ON devices(hostname);
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_last_seen ON devices(last_seen);
CREATE INDEX idx_devices_domain ON devices(domain);
CREATE INDEX idx_devices_capabilities ON devices USING GIN(capabilities);
```

#### telemetry
Partitioned table for time-series telemetry data.

```sql
-- Main telemetry table (partitioned)
CREATE TABLE telemetry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    value_numeric NUMERIC,
    value_text TEXT,
    value_boolean BOOLEAN,
    unit VARCHAR(20),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

-- Partition creation function
CREATE OR REPLACE FUNCTION create_telemetry_partition(start_date DATE, end_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
BEGIN
    partition_name := 'telemetry_' || TO_CHAR(start_date, 'YYYY_MM_DD');

    EXECUTE FORMAT(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF telemetry
         FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );

    -- Create indexes on partition
    EXECUTE FORMAT(
        'CREATE INDEX IF NOT EXISTS idx_%I_device_timestamp ON %I(device_id, timestamp)',
        partition_name, partition_name
    );

    EXECUTE FORMAT(
        'CREATE INDEX IF NOT EXISTS idx_%I_metric_type ON %I(metric_type)',
        partition_name, partition_name
    );

    EXECUTE FORMAT(
        'CREATE INDEX IF NOT EXISTS idx_%I_timestamp ON %I(timestamp)',
        partition_name, partition_name
    );
END;
$$ LANGUAGE plpgsql;

-- Daily partition management
SELECT create_telemetry_partition(CURRENT_DATE, CURRENT_DATE + INTERVAL '1 day');
```

#### policies
Stores policy definitions and targeting rules.

```sql
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL,
    content JSONB NOT NULL,
    target_filters JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_policies_name_version ON policies(name, version);
CREATE INDEX idx_policies_enabled ON policies(enabled);
CREATE INDEX idx_policies_target_filters ON policies USING GIN(target_filters);
```

#### commands
Stores command execution requests and results.

```sql
CREATE TABLE commands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    command_type VARCHAR(50) NOT NULL,
    command_data JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'expired')),
    result JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_commands_device_id ON commands(device_id);
CREATE INDEX idx_commands_status ON commands(status);
CREATE INDEX idx_commands_created_at ON commands(created_at);
CREATE INDEX idx_commands_expires_at ON commands(expires_at);
```

#### users
Stores user accounts for web console access.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'viewer' CHECK (role IN ('admin', 'operator', 'viewer')),
    last_login TIMESTAMP WITH TIME ZONE,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

#### audit_log
Tracks all system activities for compliance and debugging.

```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id UUID REFERENCES users(id),
    device_id UUID REFERENCES devices(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB,
    ip_address INET,
    user_agent TEXT
) PARTITION BY RANGE (timestamp);

-- Indexes
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_log_device_id ON audit_log(device_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_resource_type ON audit_log(resource_type);
```

## Partitioning Strategy

### Telemetry Data Partitioning
- **Partition Key**: `timestamp`
- **Partition Interval**: Daily
- **Retention**: 90 days (configurable)
- **Benefits**:
  - Improved query performance for time-based filters
  - Easier maintenance and backup operations
  - Automatic partition pruning

### Partition Management
```sql
-- Automatic partition creation (run daily)
CREATE OR REPLACE FUNCTION manage_telemetry_partitions()
RETURNS VOID AS $$
DECLARE
    next_partition_date DATE;
BEGIN
    -- Create next 7 days of partitions
    FOR i IN 0..6 LOOP
        next_partition_date := CURRENT_DATE + i;
        PERFORM create_telemetry_partition(next_partition_date, next_partition_date + 1);
    END LOOP;

    -- Drop old partitions (older than 90 days)
    FOR partition_rec IN
        SELECT tablename
        FROM pg_tables
        WHERE tablename LIKE 'telemetry_%'
        AND tablename != 'telemetry'
        AND TO_DATE(SUBSTRING(tablename FROM 11), 'YYYY_MM_DD') < CURRENT_DATE - INTERVAL '90 days'
    LOOP
        EXECUTE FORMAT('DROP TABLE IF EXISTS %I', partition_rec.tablename);
    END LOOP;
END;
$$ LANGUAGE plpgsql;
```

## Indexing Strategy

### Performance Indexes
```sql
-- Composite indexes for common query patterns
CREATE INDEX idx_telemetry_device_metric_time ON telemetry(device_id, metric_type, timestamp DESC);
CREATE INDEX idx_commands_device_status_created ON commands(device_id, status, created_at DESC);
CREATE INDEX idx_devices_status_last_seen ON devices(status, last_seen DESC);

-- Partial indexes for active data
CREATE INDEX idx_commands_pending ON commands(device_id, created_at) WHERE status = 'pending';
CREATE INDEX idx_policies_active ON policies(id) WHERE enabled = true;

-- JSONB indexes for flexible queries
CREATE INDEX idx_telemetry_metadata ON telemetry USING GIN(metadata);
CREATE INDEX idx_devices_capabilities ON devices USING GIN(capabilities);
```

### Index Maintenance
```sql
-- Reindexing script (run weekly)
REINDEX TABLE CONCURRENTLY devices;
REINDEX TABLE CONCURRENTLY policies;
REINDEX TABLE CONCURRENTLY commands;
REINDEX TABLE CONCURRENTLY users;

-- Analyze tables for query planner
ANALYZE devices, telemetry, policies, commands, users, audit_log;
```

## Data Retention Policies

### Telemetry Data
- **Hot Storage**: Last 30 days (frequent access)
- **Warm Storage**: 31-90 days (less frequent access)
- **Cold Storage**: 91+ days (archived or deleted)

### Audit Logs
- **Retention**: 7 years (compliance requirement)
- **Compression**: Monthly compressed partitions
- **Archiving**: Automated archival to long-term storage

### Command History
- **Retention**: 1 year
- **Cleanup**: Automated deletion of expired commands

## Backup and Recovery

### Backup Strategy
```bash
# Daily full backup
pg_dump -h localhost -U inventory -d inventory_db -F c -b -v > "inventory_$(date +%Y%m%d).backup"

# Hourly incremental backups using WAL
# Configure WAL archiving in postgresql.conf
archive_command = 'cp %p /var/lib/postgresql/archive/%f'

# Point-in-time recovery setup
recovery_target_time = '2024-01-15 10:00:00+00'
```

### Recovery Procedures
```sql
-- Restore from backup
pg_restore -h localhost -U inventory -d inventory_db -v inventory_20240115.backup

-- Point-in-time recovery
# Stop PostgreSQL
# Move WAL files to recovery location
# Update recovery.conf
# Start PostgreSQL
```

## Performance Optimization

### Query Optimization
```sql
-- Common query patterns with optimizations
EXPLAIN ANALYZE
SELECT * FROM telemetry
WHERE device_id = $1
  AND timestamp >= $2
  AND timestamp < $3
  AND metric_type = $4
ORDER BY timestamp DESC
LIMIT 1000;

-- Use CTEs for complex aggregations
WITH daily_stats AS (
    SELECT
        device_id,
        DATE(timestamp) as date,
        AVG(value_numeric) as avg_value,
        MAX(value_numeric) as max_value,
        MIN(value_numeric) as min_value
    FROM telemetry
    WHERE metric_type = 'cpu_usage'
        AND timestamp >= CURRENT_DATE - INTERVAL '30 days'
    GROUP BY device_id, DATE(timestamp)
)
SELECT * FROM daily_stats
ORDER BY date DESC, device_id;
```

### Connection Pooling
```sql
-- PgBouncer configuration
[databases]
inventory = host=localhost port=5432 dbname=inventory_db

[pgbouncer]
listen_port = 6432
listen_addr = localhost
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
reserve_pool_size = 5
```

## Monitoring and Maintenance

### Health Checks
```sql
-- Database connectivity check
SELECT 1 as health_check;

-- Replication lag check
SELECT
    client_addr,
    state,
    sent_lsn,
    write_lsn,
    flush_lsn,
    replay_lsn,
    CASE
        WHEN pg_last_wal_receive_lsn() = pg_last_wal_replay_lsn() THEN 0
        ELSE EXTRACT(EPOCH FROM (pg_last_wal_receive_lsn() - pg_last_wal_replay_lsn()))
    END as lag_seconds
FROM pg_stat_replication;
```

### Maintenance Scripts
```sql
-- Vacuum and analyze (run nightly)
VACUUM ANALYZE devices, policies, commands, users;
VACUUM ANALYZE telemetry; -- Only on partitions

-- Update table statistics
ANALYZE;

-- Monitor table bloat
SELECT
    schemaname, tablename,
    n_dead_tup, n_live_tup,
    ROUND(n_dead_tup::float / (n_live_tup + n_dead_tup) * 100, 2) as bloat_ratio
FROM pg_stat_user_tables
ORDER BY bloat_ratio DESC;
```

## Migration Strategy

### Schema Migrations
```sql
-- Migration table
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT
);

-- Example migration
INSERT INTO schema_migrations (version, description)
VALUES ('001_initial_schema', 'Create initial database schema');

-- Rollback support
CREATE OR REPLACE FUNCTION rollback_migration(target_version VARCHAR)
RETURNS VOID AS $$
-- Implementation for rolling back to specific version
$$ LANGUAGE plpgsql;
```

## Security Considerations

### Row Level Security (RLS)
```sql
-- Enable RLS on sensitive tables
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE telemetry ENABLE ROW LEVEL SECURITY;

-- Create policies for data access
CREATE POLICY device_access ON devices
    FOR ALL USING (user_id = current_user_id() OR user_role() = 'admin');
```

### Data Encryption
```sql
-- Encrypt sensitive data at rest
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encrypted API keys
CREATE OR REPLACE FUNCTION encrypt_api_key(plain_key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(encrypt(plain_key::bytea, current_setting('encryption.key')::bytea, 'aes'), 'hex');
END;
$$ LANGUAGE plpgsql;
```

## Scaling Considerations

### Read Replicas
```sql
-- Create read replica
CREATE PUBLICATION inventory_pub FOR ALL TABLES;
CREATE SUBSCRIPTION inventory_sub
    CONNECTION 'host=replica_host dbname=inventory_db user=replication_user'
    PUBLICATION inventory_pub;
```

### Sharding Strategy (Future)
- **Shard Key**: `device_id` for telemetry data
- **Shard Count**: Based on expected device count
- **Rebalancing**: Automated shard rebalancing during low-traffic periods

This database design provides a solid foundation for handling large-scale Windows inventory data with optimal performance, maintainability, and scalability.