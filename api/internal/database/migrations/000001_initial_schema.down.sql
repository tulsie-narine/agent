-- +migrate Down
-- Drop all tables in reverse order

DROP TRIGGER IF EXISTS update_commands_updated_at ON commands;
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS commands;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS telemetry_latest;

-- Drop all telemetry partitions
DO $$
DECLARE
    partition_name TEXT;
BEGIN
    FOR partition_name IN
        SELECT tablename FROM pg_tables
        WHERE tablename LIKE 'telemetry_y%'
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || partition_name;
    END LOOP;
END $$;

DROP TABLE IF EXISTS telemetry;
DROP TABLE IF EXISTS agents;

DROP FUNCTION IF EXISTS update_updated_at_column();