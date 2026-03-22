package buffer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atk-tracker/shared/go/atkshared"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	return filepath.Join(dir, "test-buffer.db")
}

func TestStore_EnqueueAndDequeue(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	payload := atkshared.HeartbeatPayload{
		SessionID: "sess-1",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Duration:  120,
	}

	ctx := context.Background()

	if err := s.Enqueue(ctx, payload); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	items, err := s.DequeueBatch(ctx, 10)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Payload.SessionID != "sess-1" {
		t.Fatalf("expected sess-1, got %s", items[0].Payload.SessionID)
	}

	if items[0].Payload.Duration != 120 {
		t.Fatalf("expected duration 120, got %d", items[0].Payload.Duration)
	}
}

func TestStore_DequeueBatch_RespectsLimit(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "s", Timestamp: time.Now(), Duration: 10})
	}

	items, err := s.DequeueBatch(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items (limit), got %d", len(items))
	}
}

func TestStore_DequeueBatch_EmptyQueue(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	items, err := s.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 0 {
		t.Fatalf("expected 0 items from empty queue, got %d", len(items))
	}
}

func TestStore_DequeueBatch_ZeroLimit(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()

	_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "s", Timestamp: time.Now(), Duration: 10})

	items, err := s.DequeueBatch(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestStore_DeleteByID(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()

	_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "s1", Timestamp: time.Now(), Duration: 10})
	_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "s2", Timestamp: time.Now(), Duration: 20})

	items, _ := s.DequeueBatch(ctx, 10)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if err := s.DeleteByID(ctx, items[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	remaining, _ := s.DequeueBatch(ctx, 10)

	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}

	if remaining[0].Payload.SessionID != "s2" {
		t.Fatalf("expected s2, got %s", remaining[0].Payload.SessionID)
	}
}

func TestStore_DequeueBatch_OrderByID(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()

	_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "first", Timestamp: time.Now(), Duration: 10})
	_ = s.Enqueue(ctx, atkshared.HeartbeatPayload{SessionID: "second", Timestamp: time.Now(), Duration: 20})

	items, _ := s.DequeueBatch(ctx, 10)

	if items[0].Payload.SessionID != "first" || items[1].Payload.SessionID != "second" {
		t.Fatalf("expected FIFO order, got %s, %s", items[0].Payload.SessionID, items[1].Payload.SessionID)
	}
}

func TestStore_Ping(t *testing.T) {
	s, err := New(tempDB(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestNew_InvalidPath(t *testing.T) {
	_, err := New(filepath.Join(os.DevNull, "nonexistent", "buffer.db"))

	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
