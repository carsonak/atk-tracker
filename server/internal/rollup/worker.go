package rollup

import (
	"context"
	"log"
	"time"

	"atk-tracker/server/internal/db"
)

type Worker struct {
	store    *db.Store
	interval time.Duration
}

func NewWorker(store *db.Store, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Worker{store: store, interval: interval}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			day := time.Now().UTC().Add(-24 * time.Hour)
			if err := w.store.RollupPreviousDay(ctx, day); err != nil {
				log.Printf("rollup failed for %s: %v", day.Format(time.DateOnly), err)
			}
		}
	}
}
