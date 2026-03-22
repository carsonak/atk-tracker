-- Demo dataset for ATK Tracker dashboards.
-- Safe to re-run: this script only resets demo-* rows.

BEGIN;

DELETE FROM raw_heartbeats
WHERE session_id IN (
    SELECT id FROM sessions WHERE apprentice_id LIKE 'demo-%'
);

DELETE FROM daily_summaries
WHERE apprentice_id LIKE 'demo-%';

DELETE FROM sessions
WHERE apprentice_id LIKE 'demo-%';

INSERT INTO sessions (id, apprentice_id, machine_id, login_time, logout_time)
VALUES
    (
        'demo-session-anna-01',
        'demo-anna',
        'zone01-node-01',
        NOW() - INTERVAL '1 day' - INTERVAL '4 hours',
        NOW() - INTERVAL '1 day'
    ),
    (
        'demo-session-anna-02',
        'demo-anna',
        'zone01-node-07',
        NOW() - INTERVAL '6 hours',
        NULL
    ),
    (
        'demo-session-bao-01',
        'demo-bao',
        'zone01-node-02',
        NOW() - INTERVAL '1 day' - INTERVAL '3 hours',
        NOW() - INTERVAL '1 day' - INTERVAL '30 minutes'
    ),
    (
        'demo-session-caro-01',
        'demo-caro',
        'zone01-node-08',
        NOW() - INTERVAL '8 hours',
        NULL
    );

-- Raw heartbeats for detailed charting (5-min windows).
INSERT INTO raw_heartbeats (session_id, ts, active_seconds)
SELECT
    'demo-session-anna-02',
    NOW() - INTERVAL '6 hours' + (gs * INTERVAL '5 minutes'),
    CASE WHEN gs % 5 = 0 THEN 0 ELSE 180 + ((gs % 3) * 40) END
FROM generate_series(1, 48) AS gs;

INSERT INTO raw_heartbeats (session_id, ts, active_seconds)
SELECT
    'demo-session-caro-01',
    NOW() - INTERVAL '8 hours' + (gs * INTERVAL '5 minutes'),
    CASE WHEN gs % 7 = 0 THEN 0 ELSE 120 + ((gs % 4) * 45) END
FROM generate_series(1, 52) AS gs;

INSERT INTO raw_heartbeats (session_id, ts, active_seconds)
SELECT
    'demo-session-bao-01',
    NOW() - INTERVAL '1 day' - INTERVAL '3 hours' + (gs * INTERVAL '5 minutes'),
    CASE WHEN gs % 6 = 0 THEN 60 ELSE 220 END
FROM generate_series(1, 30) AS gs;

-- Daily summaries for longer-range dashboard view.
INSERT INTO daily_summaries (summary_date, apprentice_id, total_active_minutes)
SELECT
    (CURRENT_DATE - offs),
    apprentice_id,
    CASE apprentice_id
        WHEN 'demo-anna' THEN 280 + ((offs % 4) * 18)
        WHEN 'demo-bao' THEN 210 + ((offs % 5) * 12)
        WHEN 'demo-caro' THEN 240 + ((offs % 3) * 25)
        ELSE 180
    END
FROM generate_series(0, 13) AS offs
CROSS JOIN (VALUES ('demo-anna'), ('demo-bao'), ('demo-caro')) AS users(apprentice_id);

COMMIT;
