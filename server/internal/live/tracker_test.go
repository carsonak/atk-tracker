package live

import (
	"testing"
	"time"
)

func TestTrackerTTL(t *testing.T) {
	tr := NewTracker(1 * time.Minute)
	now := time.Now().UTC()
	tr.Touch("uid-1000", "machine-a", now)
	tr.Touch("uid-1001", "machine-b", now.Add(-2*time.Minute))

	live := tr.List(now)
	if len(live) != 1 {
		t.Fatalf("expected 1 active record, got %d", len(live))
	}
	if live[0].ApprenticeID != "uid-1000" {
		t.Fatalf("unexpected apprentice %s", live[0].ApprenticeID)
	}
}
