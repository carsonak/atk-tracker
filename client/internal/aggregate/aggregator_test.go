package aggregate

import (
	"testing"
	"time"
)

func TestAggregatorEmitsWindowSummary(t *testing.T) {
	agg := New(2 * time.Second)
	events := make(chan struct{}, 8)
	stop := make(chan struct{})
	out := agg.Run(events, stop)

	events <- struct{}{}
	events <- struct{}{}

	select {
	case got := <-out:
		if got.ActiveSeconds < 1 || got.ActiveSeconds > 2 {
			t.Fatalf("expected active seconds in [1,2], got %d", got.ActiveSeconds)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for summary")
	}
	close(stop)
}
