package buffer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"atk-tracker/shared/go/atkshared"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	stmt := `
	CREATE TABLE IF NOT EXISTS heartbeat_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		payload TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(stmt); err != nil {
		return nil, fmt.Errorf("create queue table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Enqueue(ctx context.Context, payload atkshared.HeartbeatPayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO heartbeat_queue(payload) VALUES(?)", string(b))
	return err
}

type QueuedHeartbeat struct {
	ID      int64
	Payload atkshared.HeartbeatPayload
}

func (s *Store) DequeueBatch(ctx context.Context, limit int) ([]QueuedHeartbeat, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, payload FROM heartbeat_queue ORDER BY id ASC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]QueuedHeartbeat, 0, limit)

	for rows.Next() {
		var id int64
		var payloadJSON string

		if err := rows.Scan(&id, &payloadJSON); err != nil {
			return nil, err
		}

		var payload atkshared.HeartbeatPayload

		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			continue
		}

		items = append(items, QueuedHeartbeat{ID: id, Payload: payload})
	}

	return items, rows.Err()
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM heartbeat_queue WHERE id = ?", id)

	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)

	defer cancel()
	return s.db.PingContext(ctx)
}
