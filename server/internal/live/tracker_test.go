package live

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTracker_Touch_And_List_Active(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now)
	tr.Touch("bob", "node-2", now.Add(-1*time.Minute))

	list := tr.List(now)

	if len(list) != 2 {
		t.Fatalf("expected 2 active, got %d", len(list))
	}
}

func TestTracker_List_ExcludesExpired(t *testing.T) {
	tr := NewTracker(1 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now)
	tr.Touch("bob", "node-2", now.Add(-2*time.Minute))

	list := tr.List(now)

	if len(list) != 1 {
		t.Fatalf("expected 1 active, got %d", len(list))
	}

	if list[0].ApprenticeID != "alice" {
		t.Fatalf("expected alice, got %s", list[0].ApprenticeID)
	}
}

func TestTracker_Touch_UpdatesExistingEntry(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now.Add(-4*time.Minute))
	tr.Touch("alice", "node-1", now) // refresh

	list := tr.List(now)

	if len(list) != 1 {
		t.Fatalf("expected 1 entry (updated), got %d", len(list))
	}

	if !list[0].LastSeen.Equal(now) {
		t.Fatalf("expected LastSeen to be updated to %v, got %v", now, list[0].LastSeen)
	}
}

func TestTracker_SameUser_DifferentMachines(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now)
	tr.Touch("alice", "node-2", now)

	list := tr.List(now)

	if len(list) != 2 {
		t.Fatalf("expected 2 entries for same user on different machines, got %d", len(list))
	}
}

func TestTracker_List_EmptyTracker(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	list := tr.List(time.Now().UTC())

	if list == nil {
		t.Fatal("expected non-nil empty slice")
	}

	if len(list) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(list))
	}
}

func TestTracker_ExactTTLBoundary(t *testing.T) {
	ttl := 5 * time.Minute
	tr := NewTracker(ttl)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now.Add(-ttl)) // exactly at boundary

	list := tr.List(now)
	// At exactly the TTL boundary, now.Sub(lastSeen) == ttl, which is <= ttl, so included
	if len(list) != 1 {
		t.Fatalf("expected 1 at exact boundary, got %d", len(list))
	}
}

func TestTracker_JustPastTTLBoundary(t *testing.T) {
	ttl := 5 * time.Minute
	tr := NewTracker(ttl)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now.Add(-ttl-time.Millisecond))

	list := tr.List(now)

	if len(list) != 0 {
		t.Fatalf("expected 0 just past boundary, got %d", len(list))
	}
}

func TestTracker_CleanupExpired(t *testing.T) {
	tr := NewTracker(1 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now)
	tr.Touch("bob", "node-2", now.Add(-5*time.Minute))

	tr.cleanupExpired(now)

	tr.mu.RLock()
	defer tr.mu.RUnlock()
	if len(tr.lastSeen) != 1 {
		t.Fatalf("expected 1 entry after cleanup, got %d", len(tr.lastSeen))
	}
}

func TestTracker_CleanupExpired_AllExpired(t *testing.T) {
	tr := NewTracker(1 * time.Minute)
	now := time.Now().UTC()

	tr.Touch("alice", "node-1", now.Add(-5*time.Minute))
	tr.Touch("bob", "node-2", now.Add(-5*time.Minute))

	tr.cleanupExpired(now)

	tr.mu.RLock()
	defer tr.mu.RUnlock()
	if len(tr.lastSeen) != 0 {
		t.Fatalf("expected 0 entries after full cleanup, got %d", len(tr.lastSeen))
	}
}

func TestTracker_ConcurrentTouchAndList(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	now := time.Now().UTC()

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			tr.Touch("user", "machine", now)
		}(i)
		go func() {
			defer wg.Done()
			_ = tr.List(now)
		}()
	}

	wg.Wait()

	list := tr.List(now)

	if len(list) != 1 {
		t.Fatalf("expected 1 entry after concurrent ops, got %d", len(list))
	}
}

func TestTracker_StartCleanup_StopsOnCancel(t *testing.T) {
	tr := NewTracker(100 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		tr.StartCleanup(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartCleanup did not stop after context cancel")
	}
}

// --- mock SessionCloser ---

type mockSessionCloser struct {
	mu     sync.Mutex
	calls  int
	result int64
	err    error
}

func (m *mockSessionCloser) CloseStaleSessions(_ context.Context, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.result, m.err
}

func TestTracker_StartSessionReaper_ClosesStale(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	closer := &mockSessionCloser{result: 3}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	done := make(chan struct{})

	go func() {
		tr.StartSessionReaper(ctx, closer, 200*time.Millisecond)
		close(done)
	}()

	// Wait enough for at least one tick (interval = threshold/2 = 100ms)
	time.Sleep(350 * time.Millisecond)
	cancel()
	<-done

	closer.mu.Lock()
	defer closer.mu.Unlock()
	if closer.calls == 0 {
		t.Fatal("expected reaper to call CloseStaleSessions at least once")
	}
}

func TestTracker_StartSessionReaper_StopsOnCancel(t *testing.T) {
	tr := NewTracker(5 * time.Minute)
	closer := &mockSessionCloser{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		tr.StartSessionReaper(ctx, closer, 1*time.Second)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartSessionReaper did not stop after context cancel")
	}
}
