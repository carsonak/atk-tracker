CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    apprentice_id TEXT NOT NULL,
    machine_id TEXT NOT NULL,
    login_time TIMESTAMPTZ NOT NULL,
    logout_time TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_apprentice_active
    ON sessions(apprentice_id) WHERE logout_time IS NULL;

CREATE TABLE IF NOT EXISTS raw_heartbeats (
    id BIGSERIAL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL,
    active_seconds INTEGER NOT NULL CHECK (active_seconds BETWEEN 0 AND 300),
    PRIMARY KEY (id, ts)
) PARTITION BY RANGE (ts);

CREATE INDEX IF NOT EXISTS idx_raw_heartbeats_session_id ON raw_heartbeats(session_id);

CREATE TABLE IF NOT EXISTS daily_summaries (
    summary_date DATE NOT NULL,
    apprentice_id TEXT NOT NULL,
    total_active_minutes INTEGER NOT NULL,
    PRIMARY KEY(summary_date, apprentice_id)
);

CREATE TABLE IF NOT EXISTS housekeeping_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
