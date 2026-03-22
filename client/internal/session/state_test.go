package session

import (
	"sync"
	"testing"
)

func TestState_SetAndGet(t *testing.T) {
	s := &State{}

	s.Set("sess-1", "alice")

	sid, aid := s.Get()

	if sid != "sess-1" {
		t.Fatalf("expected session 'sess-1', got %q", sid)
	}

	if aid != "alice" {
		t.Fatalf("expected apprentice 'alice', got %q", aid)
	}
}

func TestState_ZeroValue(t *testing.T) {
	s := &State{}
	sid, aid := s.Get()

	if sid != "" || aid != "" {
		t.Fatalf("expected empty strings from zero-value State, got %q, %q", sid, aid)
	}
}

func TestState_Clear(t *testing.T) {
	s := &State{}

	s.Set("sess-1", "alice")
	s.Clear()

	sid, aid := s.Get()

	if sid != "" || aid != "" {
		t.Fatalf("expected empty after Clear, got %q, %q", sid, aid)
	}
}

func TestState_OverwritesPrevious(t *testing.T) {
	s := &State{}

	s.Set("sess-1", "alice")
	s.Set("sess-2", "bob")

	sid, aid := s.Get()

	if sid != "sess-2" || aid != "bob" {
		t.Fatalf("expected overwritten values, got %q, %q", sid, aid)
	}
}

func TestState_ConcurrentAccess(t *testing.T) {
	s := &State{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.Set("s", "a")
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Get()
		}()
	}

	wg.Wait()
}
