-- Partition management for raw_heartbeats.
-- Creates monthly partitions and drops partitions older than the retention window.
-- Schedule with pg_cron: SELECT cron.schedule('partition-mgmt', '0 0 1 * *', $$SELECT maintain_heartbeat_partitions(3)$$);

-- Ensure a partition exists for a given month.
CREATE OR REPLACE FUNCTION create_heartbeat_partition(start_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    partition_name := 'raw_heartbeats_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + INTERVAL '1 month';

    IF NOT EXISTS (
        SELECT 1 FROM pg_class WHERE relname = partition_name
    ) THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF raw_heartbeats FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
        RAISE NOTICE 'Created partition %', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Maintain partitions: create upcoming months, drop old ones beyond retention.
CREATE OR REPLACE FUNCTION maintain_heartbeat_partitions(
    retention_months INT DEFAULT 3
)
RETURNS VOID AS $$
DECLARE
    m INT;
    target DATE;
    old_partition TEXT;
    cutoff DATE;
BEGIN
    -- Create current month + next 2 months
    FOR m IN 0..2 LOOP
        target := date_trunc('month', CURRENT_DATE)::date + (m || ' months')::interval;
        PERFORM create_heartbeat_partition(target);
    END LOOP;

    -- Drop partitions older than retention window
    cutoff := (date_trunc('month', CURRENT_DATE) - (retention_months || ' months')::interval)::date;
    FOR old_partition IN
        SELECT c.relname
        FROM pg_inherits i
        JOIN pg_class c ON c.oid = i.inhrelid
        JOIN pg_class p ON p.oid = i.inhparent
        WHERE p.relname = 'raw_heartbeats'
          AND c.relname < 'raw_heartbeats_' || to_char(cutoff, 'YYYY_MM')
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I', old_partition);
        RAISE NOTICE 'Dropped old partition %', old_partition;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Bootstrap: create initial partitions for current and next 2 months.
SELECT maintain_heartbeat_partitions(3);
