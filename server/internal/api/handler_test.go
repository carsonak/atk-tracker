package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atk-tracker/server/internal/live"
	"atk-tracker/shared/go/atkshared"
)

// --- mock store ---

type mockStore struct {
	createSession      func(ctx context.Context, apprenticeID, machineID string) (string, error)
	endSession         func(ctx context.Context, sessionID string, endTime time.Time) error
	validateSession    func(ctx context.Context, sessionID string) (bool, string, string, error)
	insertHeartbeat    func(ctx context.Context, hb atkshared.HeartbeatPayload) error
	countActive        func(ctx context.Context, apprenticeID string) (int, error)
	liveRawSeries      func(ctx context.Context, apprenticeID string, from, to time.Time) ([]atkshared.HistoricalPoint, error)
	dailySummarySeries func(ctx context.Context, apprenticeID string, from, to time.Time) ([]atkshared.HistoricalPoint, error)
}

func (m *mockStore) CreateSession(ctx context.Context, aid, mid string) (string, error) {
	if m.createSession != nil {
		return m.createSession(ctx, aid, mid)
	}
	return "mock-session-id", nil
}
func (m *mockStore) EndSession(ctx context.Context, sid string, end time.Time) error {
	if m.endSession != nil {
		return m.endSession(ctx, sid, end)
	}
	return nil
}
func (m *mockStore) ValidateSession(ctx context.Context, sid string) (bool, string, string, error) {
	if m.validateSession != nil {
		return m.validateSession(ctx, sid)
	}
	return true, "user-1", "machine-1", nil
}
func (m *mockStore) InsertHeartbeat(ctx context.Context, hb atkshared.HeartbeatPayload) error {
	if m.insertHeartbeat != nil {
		return m.insertHeartbeat(ctx, hb)
	}
	return nil
}
func (m *mockStore) CountActiveSessions(ctx context.Context, aid string) (int, error) {
	if m.countActive != nil {
		return m.countActive(ctx, aid)
	}
	return 0, nil
}
func (m *mockStore) LiveRawSeries(ctx context.Context, aid string, from, to time.Time) ([]atkshared.HistoricalPoint, error) {
	if m.liveRawSeries != nil {
		return m.liveRawSeries(ctx, aid, from, to)
	}
	return nil, nil
}
func (m *mockStore) DailySummarySeries(ctx context.Context, aid string, from, to time.Time) ([]atkshared.HistoricalPoint, error) {
	if m.dailySummarySeries != nil {
		return m.dailySummarySeries(ctx, aid, from, to)
	}
	return nil, nil
}

// --- helpers ---

func newTestHandler(store DataStore, apiKey string) *Handler {
	tr := live.NewTracker(10 * time.Minute)
	return NewHandlerWithKey(store, tr, apiKey)
}

func jsonBody(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewReader(b)
}

func doRequest(handler http.Handler, method, path string, body *bytes.Reader, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// =====================================================================
// API Key Middleware tests
// =====================================================================

func TestRequireAPIKey_NoKeyConfigured_PassesThrough(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireAPIKey_ValidXAPIKey(t *testing.T) {
	h := newTestHandler(&mockStore{}, "secret-key")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, map[string]string{"X-API-Key": "secret-key"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireAPIKey_ValidBearerToken(t *testing.T) {
	h := newTestHandler(&mockStore{}, "secret-key")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, map[string]string{"Authorization": "Bearer secret-key"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireAPIKey_MissingKey(t *testing.T) {
	h := newTestHandler(&mockStore{}, "secret-key")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAPIKey_WrongKey(t *testing.T) {
	h := newTestHandler(&mockStore{}, "secret-key")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, map[string]string{"X-API-Key": "wrong-key"})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAPIKey_ReadEndpointsPublic(t *testing.T) {
	h := newTestHandler(&mockStore{}, "secret-key")
	rr := doRequest(h.Routes(), http.MethodGet, "/live", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /live should be public, got %d", rr.Code)
	}
}

// =====================================================================
// POST /sessions tests
// =====================================================================

func TestCreateSession_Success(t *testing.T) {
	store := &mockStore{
		createSession: func(_ context.Context, aid, mid string) (string, error) {
			if aid != "alice" || mid != "node-1" {
				t.Fatalf("unexpected args: %s, %s", aid, mid)
			}
			return "new-sid", nil
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "alice", MachineID: "node-1"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp atkshared.CreateSessionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.SessionID != "new-sid" {
		t.Fatalf("expected session_id 'new-sid', got %q", resp.SessionID)
	}
}

func TestCreateSession_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		req  atkshared.CreateSessionRequest
	}{
		{"empty apprentice_id", atkshared.CreateSessionRequest{MachineID: "m"}},
		{"empty machine_id", atkshared.CreateSessionRequest{ApprenticeID: "a"}},
		{"both empty", atkshared.CreateSessionRequest{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(&mockStore{}, "")
			body := jsonBody(t, tc.req)
			rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestCreateSession_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", bytes.NewReader([]byte("not json")), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateSession_ConcurrentSessionCap(t *testing.T) {
	store := &mockStore{
		countActive: func(_ context.Context, _ string) (int, error) {
			return 2, nil
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 when at session cap, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateSession_CountActiveError(t *testing.T) {
	store := &mockStore{
		countActive: func(_ context.Context, _ string) (int, error) {
			return 0, errors.New("db down")
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCreateSession_StoreError(t *testing.T) {
	store := &mockStore{
		createSession: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("insert failed")
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.CreateSessionRequest{ApprenticeID: "a", MachineID: "m"})
	rr := doRequest(h.Routes(), http.MethodPost, "/sessions", body, nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// =====================================================================
// PUT /sessions/{id}/end tests
// =====================================================================

func TestEndSession_Success(t *testing.T) {
	called := false
	store := &mockStore{
		endSession: func(_ context.Context, sid string, _ time.Time) error {
			called = true
			if sid != "sess-42" {
				t.Fatalf("unexpected session id: %s", sid)
			}
			return nil
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.EndSessionRequest{EndTime: time.Now().UTC()})
	rr := doRequest(h.Routes(), http.MethodPut, "/sessions/sess-42/end", body, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatal("EndSession was not called on store")
	}
}

func TestEndSession_NotFound(t *testing.T) {
	store := &mockStore{
		endSession: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("session not found")
		},
	}
	h := newTestHandler(store, "")
	body := jsonBody(t, atkshared.EndSessionRequest{EndTime: time.Now().UTC()})
	rr := doRequest(h.Routes(), http.MethodPut, "/sessions/no-exist/end", body, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestEndSession_InvalidBody(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	rr := doRequest(h.Routes(), http.MethodPut, "/sessions/sess-1/end", bytes.NewReader([]byte("bad")), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// =====================================================================
// POST /heartbeats tests
// =====================================================================

func TestCreateHeartbeat_Success(t *testing.T) {
	var captured atkshared.HeartbeatPayload
	store := &mockStore{
		insertHeartbeat: func(_ context.Context, hb atkshared.HeartbeatPayload) error {
			captured = hb
			return nil
		},
	}
	h := newTestHandler(store, "")
	payload := atkshared.HeartbeatPayload{
		SessionID: "sess-1",
		Timestamp: time.Now().UTC(),
		Duration:  120,
	}
	body := jsonBody(t, payload)
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if captured.ApprenticeID != "user-1" {
		t.Fatalf("expected handler to enrich apprentice_id from session, got %q", captured.ApprenticeID)
	}
}

func TestCreateHeartbeat_MissingSessionID(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	payload := atkshared.HeartbeatPayload{Timestamp: time.Now(), Duration: 100}
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", jsonBody(t, payload), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateHeartbeat_MissingTimestamp(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	payload := atkshared.HeartbeatPayload{SessionID: "s", Duration: 100}
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", jsonBody(t, payload), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateHeartbeat_InvalidDuration(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	tests := []struct {
		name     string
		duration int
	}{
		{"negative", -1},
		{"over max", 301},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := atkshared.HeartbeatPayload{SessionID: "s", Timestamp: time.Now(), Duration: tc.duration}
			rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", jsonBody(t, payload), nil)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestCreateHeartbeat_InvalidSession(t *testing.T) {
	store := &mockStore{
		validateSession: func(_ context.Context, _ string) (bool, string, string, error) {
			return false, "", "", nil
		},
	}
	h := newTestHandler(store, "")
	payload := atkshared.HeartbeatPayload{SessionID: "bad", Timestamp: time.Now(), Duration: 100}
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", jsonBody(t, payload), nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid session, got %d", rr.Code)
	}
}

func TestCreateHeartbeat_InsertError(t *testing.T) {
	store := &mockStore{
		insertHeartbeat: func(_ context.Context, _ atkshared.HeartbeatPayload) error {
			return errors.New("disk full")
		},
	}
	h := newTestHandler(store, "")
	payload := atkshared.HeartbeatPayload{SessionID: "s", Timestamp: time.Now(), Duration: 100}
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", jsonBody(t, payload), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestCreateHeartbeat_RecentTouchesLiveTracker(t *testing.T) {
	tracker := live.NewTracker(10 * time.Minute)
	store := &mockStore{}
	h := NewHandlerWithKey(store, tracker, "")
	payload := atkshared.HeartbeatPayload{
		SessionID: "s",
		Timestamp: time.Now().UTC(),
		Duration:  60,
	}
	body := jsonBody(t, payload)
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	entries := tracker.List(time.Now().UTC())
	if len(entries) != 1 {
		t.Fatalf("expected live tracker to have 1 entry, got %d", len(entries))
	}
}

func TestCreateHeartbeat_OldTimestampSkipsLiveTracker(t *testing.T) {
	tracker := live.NewTracker(10 * time.Minute)
	store := &mockStore{}
	h := NewHandlerWithKey(store, tracker, "")
	payload := atkshared.HeartbeatPayload{
		SessionID: "s",
		Timestamp: time.Now().Add(-1 * time.Hour).UTC(),
		Duration:  60,
	}
	body := jsonBody(t, payload)
	rr := doRequest(h.Routes(), http.MethodPost, "/heartbeats", body, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	entries := tracker.List(time.Now().UTC())
	if len(entries) != 0 {
		t.Fatalf("old heartbeat should not touch live tracker, got %d entries", len(entries))
	}
}

// =====================================================================
// GET /live tests
// =====================================================================

func TestLiveView_ReturnsJSON(t *testing.T) {
	h := newTestHandler(&mockStore{}, "")
	rr := doRequest(h.Routes(), http.MethodGet, "/live", nil, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

// =====================================================================
// parseRange edge cases
// =====================================================================

func TestParseRange_InvalidFromDate(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?from=not-a-date&to=2026-03-07", nil)
	_, _, err := parseRange(r)
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
}

func TestParseRange_InvalidToDate(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?from=2026-03-01&to=not-a-date", nil)
	_, _, err := parseRange(r)
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
}

func TestParseRange_OnlyFromProvided(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?from=2026-03-01", nil)
	from, to, err := parseRange(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When only one param is provided, falls back to 7-day default
	if !to.After(from) {
		t.Fatal("expected to > from for default range")
	}
}

func TestParseRange_SameDayRange(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?from=2026-03-15&to=2026-03-15", nil)
	from, to, err := parseRange(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from.Format("2006-01-02") != "2026-03-15" {
		t.Fatalf("unexpected from: %s", from)
	}
	// to should be day+1 (exclusive end)
	if to.Format("2006-01-02") != "2026-03-16" {
		t.Fatalf("expected to to be next day, got %s", to)
	}
}
