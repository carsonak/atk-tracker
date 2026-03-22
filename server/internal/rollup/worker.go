package rollup

import (
	"context"
	"log"
	"os"
	"time"

	"atk-tracker/server/internal/db"
)

type Worker struct {
	store    *db.Store
	interval time.Duration
	loc      *time.Location
}

func NewWorker(store *db.Store, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	loc := time.UTC
	if tz := os.Getenv("TZ"); tz != "" {
		if parsed, err := time.LoadLocation(tz); err == nil {
			loc = parsed
		} else {
			log.Printf("rollup: invalid TZ %q, falling back to UTC: %v", tz, err)
		}
	}

	return &Worker{store: store, interval: interval, loc: loc}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			day := time.Now().In(w.loc).Add(-24 * time.Hour)

			if err := w.store.RollupPreviousDay(ctx, day, w.loc); err != nil {
				log.Printf("rollup failed for %s: %v", day.Format(time.DateOnly), err)
			}
		}
	}
}
