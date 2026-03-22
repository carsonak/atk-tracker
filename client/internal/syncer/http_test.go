package syncer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atk-tracker/shared/go/atkshared"
)

func TestHTTPClient_CreateSession_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected application/json, got %s", ct)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(atkshared.CreateSessionResponse{SessionID: "new-sid"})
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	sid, err := c.CreateSession(context.Background(), "alice", "node-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sid != "new-sid" {
		t.Fatalf("expected 'new-sid', got %q", sid)
	}
}

func TestHTTPClient_CreateSession_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	_, err := c.CreateSession(context.Background(), "alice", "node-1")

	if err == nil {
		t.Fatal("expected error on 500 response")
	}
}

func TestHTTPClient_SendHeartbeat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.SendHeartbeat(context.Background(), atkshared.HeartbeatPayload{
		SessionID: "s", Timestamp: time.Now(), Duration: 60,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_SendHeartbeat_InvalidSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.SendHeartbeat(context.Background(), atkshared.HeartbeatPayload{
		SessionID: "s", Timestamp: time.Now(), Duration: 60,
	})

	if err != ErrSessionInvalid {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestHTTPClient_SendHeartbeat_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.SendHeartbeat(context.Background(), atkshared.HeartbeatPayload{
		SessionID: "s", Timestamp: time.Now(), Duration: 60,
	})

	if err != ErrSessionInvalid {
		t.Fatalf("expected ErrSessionInvalid for 404, got %v", err)
	}
}

func TestHTTPClient_SendHeartbeat_OtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.SendHeartbeat(context.Background(), atkshared.HeartbeatPayload{
		SessionID: "s", Timestamp: time.Now(), Duration: 60,
	})

	if err == nil {
		t.Fatal("expected error on 502")
	}

	if err == ErrSessionInvalid {
		t.Fatal("502 should not be ErrSessionInvalid")
	}
}

func TestHTTPClient_EndSession_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.EndSession(context.Background(), "sess-1", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_EndSession_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer srv.Close()

	c := New(srv.URL, 5*time.Second)
	err := c.EndSession(context.Background(), "bad-id", time.Now())

	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestHTTPClient_APIKeyHeader_Injected(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(atkshared.CreateSessionResponse{SessionID: "s"})
	}))

	defer srv.Close()

	c := &HTTPClient{
		baseURL: srv.URL,
		client:  &http.Client{Timeout: 5 * time.Second},
		apiKey:  "my-secret",
	}
	_, err := c.CreateSession(context.Background(), "a", "m")
	if err != nil {
		t.Fatal(err)
	}

	if gotKey != "my-secret" {
		t.Fatalf("expected X-API-Key 'my-secret', got %q", gotKey)
	}
}

func TestHTTPClient_APIKeyHeader_EmptyWhenUnset(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(atkshared.CreateSessionResponse{SessionID: "s"})
	}))

	defer srv.Close()

	c := &HTTPClient{
		baseURL: srv.URL,
		client:  &http.Client{Timeout: 5 * time.Second},
		apiKey:  "",
	}
	_, err := c.CreateSession(context.Background(), "a", "m")
	if err != nil {
		t.Fatal(err)
	}

	if gotKey != "" {
		t.Fatalf("expected no X-API-Key header, got %q", gotKey)
	}
}
