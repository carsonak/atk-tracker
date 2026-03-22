package live

import (
	"sync"
	"time"

	"atk-tracker/shared/go/atkshared"
)

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
	t.lastSeen[apprenticeID] = atkshared.LivePresence{
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
