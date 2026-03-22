package live

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"atk-tracker/shared/go/atkshared"
)

// SessionCloser is implemented by db.Store to avoid a circular import.
type SessionCloser interface {
	CloseStaleSessions(ctx context.Context, staleThreshold time.Duration) (int64, error)
}

type Tracker struct {
	mu       sync.RWMutex
	ttl      time.Duration
	lastSeen map[string]atkshared.LivePresence
}

func NewTracker(ttl time.Duration) *Tracker {
	return &Tracker{ttl: ttl, lastSeen: map[string]atkshared.LivePresence{}}
}

func (t *Tracker) Touch(apprenticeID, machineID string, seenAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastSeen[compositeKey(apprenticeID, machineID)] = atkshared.LivePresence{
		ApprenticeID: apprenticeID,
		MachineID:    machineID,
		LastSeen:     seenAt.UTC(),
	}
}

func (t *Tracker) List(now time.Time) []atkshared.LivePresence {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]atkshared.LivePresence, 0, len(t.lastSeen))

	for _, p := range t.lastSeen {
		if now.UTC().Sub(p.LastSeen) <= t.ttl {
			out = append(out, p)
		}
	}

	return out
}

func (t *Tracker) StartCleanup(ctx context.Context) {
	interval := t.ttl / 2
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.cleanupExpired(time.Now().UTC())
		}
	}
}

func (t *Tracker) cleanupExpired(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for key, p := range t.lastSeen {
		if now.Sub(p.LastSeen) > t.ttl {
			delete(t.lastSeen, key)
		}
	}
}

func compositeKey(apprenticeID, machineID string) string {
	return fmt.Sprintf("%s:%s", apprenticeID, machineID)
}

// StartSessionReaper periodically closes sessions that have not received
// a heartbeat within the given staleThreshold (e.g. 30 minutes).
func (t *Tracker) StartSessionReaper(ctx context.Context, closer SessionCloser, staleThreshold time.Duration) {
	interval := staleThreshold / 2
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			closed, err := closer.CloseStaleSessions(ctx, staleThreshold)
			if err != nil {
				log.Printf("session reaper error: %v", err)
			} else if closed > 0 {
				log.Printf("session reaper: closed %d stale sessions", closed)
			}
		}
	}
}
