package api

import (
	"encoding/json"
	"net/http"
	"time"

	"atk-tracker/server/internal/db"
	"atk-tracker/server/internal/live"
	"atk-tracker/shared/go/atkshared"

	"github.com/go-chi/chi/v5"
)

const liveHeartbeatRecencyWindow = 5 * time.Minute

type Handler struct {
	store *db.Store
	live  *live.Tracker
}

func NewHandler(store *db.Store, live *live.Tracker) *Handler {
	return &Handler{store: store, live: live}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/sessions", h.createSession)
	r.Put("/sessions/{id}/end", h.endSession)
	r.Post("/heartbeats", h.createHeartbeat)
	r.Get("/live", h.liveView)
	r.Get("/stats", h.statsView)
	return r
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var req atkshared.CreateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApprenticeID == "" || req.MachineID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	sid, err := h.store.CreateSession(r.Context(), req.ApprenticeID, req.MachineID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(atkshared.CreateSessionResponse{SessionID: sid})
}

func (h *Handler) endSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var req atkshared.EndSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.store.EndSession(r.Context(), id, req.EndTime); err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) createHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb atkshared.HeartbeatPayload

	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if hb.SessionID == "" || hb.Timestamp.IsZero() {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if err := atkshared.ValidateHeartbeatDuration(hb.Duration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ok, apprenticeID, machineID, err := h.store.ValidateSession(r.Context(), hb.SessionID)
	if err != nil {
		http.Error(w, "validation failed", http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	hb.ApprenticeID = apprenticeID
	hb.MachineID = machineID
	if err := h.store.InsertHeartbeat(r.Context(), hb); err != nil {
		http.Error(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	if now.Sub(hb.Timestamp.UTC()) <= liveHeartbeatRecencyWindow {
		h.live.Touch(apprenticeID, machineID, now)
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) liveView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.live.List(time.Now().UTC()))
}

func (h *Handler) statsView(w http.ResponseWriter, r *http.Request) {
	apprenticeID := r.URL.Query().Get("apprentice_id")
	from, to, err := parseRange(r)
	if err != nil {
		http.Error(w, "invalid date range", http.StatusBadRequest)
		return
	}

	raw, err := h.store.LiveRawSeries(r.Context(), apprenticeID, from, to)
	if err != nil {
		http.Error(w, "raw query failed", http.StatusInternalServerError)
		return
	}

	summary, err := h.store.DailySummarySeries(r.Context(), apprenticeID, from, to)
	if err != nil {
		http.Error(w, "summary query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"raw":     raw,
		"summary": summary,
	})
}

func parseRange(r *http.Request) (time.Time, time.Time, error) {
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")

	if fromRaw == "" || toRaw == "" {
		now := time.Now().UTC()

		return now.Add(-7 * 24 * time.Hour), now, nil
	}

	from, err := time.Parse(time.DateOnly, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	to, err := time.Parse(time.DateOnly, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return from.UTC(), to.UTC().Add(24 * time.Hour), nil
}
