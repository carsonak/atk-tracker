package db

import (
	"context"
	"errors"
	"time"

	"atk-tracker/shared/go/atkshared"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Connection pool tuning for production workloads.
	config.MinConns = 2
	config.MaxConns = 20
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) CountActiveSessions(ctx context.Context, apprenticeID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sessions
		WHERE apprentice_id = $1 AND logout_time IS NULL
	`, apprenticeID).Scan(&count)
	return count, err
}

func (s *Store) CloseStaleSessions(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-staleThreshold)
	cmd, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET logout_time = NOW()
		WHERE logout_time IS NULL
		  AND id IN (
		    SELECT s.id
		    FROM sessions s
		    LEFT JOIN raw_heartbeats r ON r.session_id = s.id
		    GROUP BY s.id
		    HAVING COALESCE(MAX(r.ts), s.login_time) < $1
		  )
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

func (s *Store) CreateSession(ctx context.Context, apprenticeID, machineID string) (string, error) {
	sid := uuid.NewString()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions(id, apprentice_id, machine_id, login_time)
		VALUES($1, $2, $3, NOW())
	`, sid, apprenticeID, machineID)

	return sid, err
}

func (s *Store) EndSession(ctx context.Context, sessionID string, endTime time.Time) error {
	cmd, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET logout_time = $2
		WHERE id = $1 AND logout_time IS NULL
	`, sessionID, endTime.UTC())
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (s *Store) ValidateSession(ctx context.Context, sessionID string) (bool, string, string, error) {
	var apprenticeID, machineID string
	err := s.pool.QueryRow(ctx, `
		SELECT apprentice_id, machine_id
		FROM sessions
		WHERE id = $1 AND logout_time IS NULL
	`, sessionID).Scan(&apprenticeID, &machineID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", "", nil
		}

		return false, "", "", err
	}

	return true, apprenticeID, machineID, nil
}

func (s *Store) InsertHeartbeat(ctx context.Context, hb atkshared.HeartbeatPayload) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO raw_heartbeats(session_id, ts, active_seconds)
		VALUES($1, $2, $3)
	`, hb.SessionID, hb.Timestamp.UTC(), hb.Duration)

	return err
}

func (s *Store) LiveRawSeries(ctx context.Context, apprenticeID string, from, to time.Time) ([]atkshared.HistoricalPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ts, active_seconds
		FROM raw_heartbeats r
		JOIN sessions s ON s.id = r.session_id
		WHERE s.apprentice_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts
	`, apprenticeID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []atkshared.HistoricalPoint{}

	for rows.Next() {
		var ts time.Time
		var secs int

		if err := rows.Scan(&ts, &secs); err != nil {
			return nil, err
		}

		out = append(out, atkshared.HistoricalPoint{Timestamp: ts.UTC(), ActiveMins: float64(secs) / 60.0})
	}

	return out, rows.Err()
}

func (s *Store) DailySummarySeries(ctx context.Context, apprenticeID string, from, to time.Time) ([]atkshared.HistoricalPoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT summary_date, total_active_minutes
		FROM daily_summaries
		WHERE apprentice_id = $1 AND summary_date >= $2 AND summary_date <= $3
		ORDER BY summary_date
	`, apprenticeID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []atkshared.HistoricalPoint{}

	for rows.Next() {
		var d time.Time
		var mins int

		if err := rows.Scan(&d, &mins); err != nil {
			return nil, err
		}

		out = append(out, atkshared.HistoricalPoint{Timestamp: d.UTC(), ActiveHours: float64(mins) / 60.0})
	}

	return out, rows.Err()
}

func (s *Store) RollupPreviousDay(ctx context.Context, day time.Time, loc *time.Location) error {
	// Compute the day boundaries in the configured timezone, then convert to UTC for the query.
	localDay := day.In(loc)
	startOfDay := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, loc).UTC()
	endOfDay := startOfDay.Add(24 * time.Hour)
	_, err := s.pool.Exec(ctx, rollupSQL, startOfDay, endOfDay)
	return err
}

const rollupSQL = `
WITH heartbeat_windows AS (
  SELECT s.apprentice_id,
         r.ts - interval '5 minute' AS window_start,
         r.ts AS window_end,
         LEAST(300, GREATEST(0, r.active_seconds)) AS active_seconds
  FROM raw_heartbeats r
  JOIN sessions s ON s.id = r.session_id
  WHERE r.ts >= $1 AND r.ts < $2
),
expanded AS (
  SELECT apprentice_id,
				 generate_series(
						 window_end - (active_seconds || ' seconds')::interval,
						 window_end - interval '1 second',
						 interval '1 second'
				 ) AS sec
  FROM heartbeat_windows
	WHERE active_seconds > 0
),
flattened AS (
  SELECT apprentice_id, sec
  FROM expanded
  GROUP BY apprentice_id, sec
),
aggregated AS (
  SELECT ($1::date) AS summary_date,
         apprentice_id,
         COUNT(*)::int / 60 AS total_active_minutes
  FROM flattened
  GROUP BY apprentice_id
)
INSERT INTO daily_summaries(summary_date, apprentice_id, total_active_minutes)
SELECT summary_date, apprentice_id, total_active_minutes
FROM aggregated
ON CONFLICT(summary_date, apprentice_id)
DO UPDATE SET total_active_minutes = EXCLUDED.total_active_minutes;
`
