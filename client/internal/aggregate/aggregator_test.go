package aggregate

import (
	"testing"
	"time"
)

func TestNew_DefaultWindow(t *testing.T) {
	agg := New(0)
	if agg.windowSeconds != 300 {
		t.Fatalf("expected default 300s window, got %d", agg.windowSeconds)
	}
}

func TestNew_NegativeWindow(t *testing.T) {
	agg := New(-5 * time.Second)
	if agg.windowSeconds != 300 {
		t.Fatalf("expected default 300s for negative input, got %d", agg.windowSeconds)
	}
}

func TestNew_CustomWindow(t *testing.T) {
	agg := New(10 * time.Second)
	if agg.windowSeconds != 10 {
		t.Fatalf("expected 10s window, got %d", agg.windowSeconds)
	}
}

func TestAggregator_EmitsWindowSummary(t *testing.T) {
	agg := New(2 * time.Second)
	events := make(chan struct{}, 16)
	stop := make(chan struct{})
	out := agg.Run(events, stop)

	// Send some activity events
	events <- struct{}{}
	events <- struct{}{}

	select {
	case got := <-out:
		if got.ActiveSeconds < 1 || got.ActiveSeconds > 2 {
			t.Fatalf("expected active seconds in [1,2], got %d", got.ActiveSeconds)
		}
		if got.EndAt.IsZero() {
			t.Fatal("expected non-zero EndAt")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for summary")
	}
	close(stop)
}

func TestAggregator_NoActivity_ZeroActiveSeconds(t *testing.T) {
	agg := New(2 * time.Second)
	events := make(chan struct{}, 16)
	stop := make(chan struct{})
	out := agg.Run(events, stop)

	// Don't send any events — just wait for the window

	select {
	case got := <-out:
		if got.ActiveSeconds != 0 {
			t.Fatalf("expected 0 active seconds with no events, got %d", got.ActiveSeconds)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for summary")
	}
	close(stop)
}

func TestAggregator_StopsOnClose(t *testing.T) {
	agg := New(60 * time.Second) // Long window so it won't emit naturally
	events := make(chan struct{}, 16)
	stop := make(chan struct{})
	out := agg.Run(events, stop)

	close(stop)

	// The output channel should be closed when stop fires
	select {
	case _, ok := <-out:
		if ok {
			// Got a summary before close — that's fine
		}
	case <-time.After(2 * time.Second):
		t.Fatal("output channel not closed after stop")
	}
}

func TestAggregator_MultipleWindows(t *testing.T) {
	agg := New(2 * time.Second)
	events := make(chan struct{}, 32)
	stop := make(chan struct{})
	out := agg.Run(events, stop)

	// Window 1: active
	events <- struct{}{}

	select {
	case <-out:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first window")
	}

	// Window 2: also active
	events <- struct{}{}

	select {
	case got := <-out:
		if got.ActiveSeconds < 0 || got.ActiveSeconds > 2 {
			t.Fatalf("second window: unexpected active seconds %d", got.ActiveSeconds)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for second window")
	}
	close(stop)
}
