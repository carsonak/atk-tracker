CREATE OR REPLACE FUNCTION cleanup_raw_heartbeats() RETURNS VOID AS $$
BEGIN
    DELETE FROM raw_heartbeats
    WHERE ts < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Schedule this function with pg_cron or an external cron runner.
