package atkshared

import "time"

type CreateSessionRequest struct {
	ApprenticeID string `json:"apprentice_id"`
	MachineID    string `json:"machine_id"`
}

type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
}

type EndSessionRequest struct {
	EndTime time.Time `json:"end_time"`
}

type HeartbeatPayload struct {
	SessionID     string    `json:"session_id"`
	Timestamp     time.Time `json:"timestamp"`
	Duration      int       `json:"duration"`
	MachineID     string    `json:"machine_id,omitempty"`
	ApprenticeID  string    `json:"apprentice_id,omitempty"`
	ActivityStart time.Time `json:"activity_start,omitempty"`
	ActivityEnd   time.Time `json:"activity_end,omitempty"`
}

type LivePresence struct {
	ApprenticeID string    `json:"apprentice_id"`
	MachineID    string    `json:"machine_id"`
	LastSeen     time.Time `json:"last_seen"`
}

type HistoricalPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	ActiveMins  int       `json:"active_minutes"`
	ActiveHours float64   `json:"active_hours"`
}

const (
	HeartbeatWindowSeconds = 300
	HeartbeatMaxSeconds    = 300
	HeartbeatMinSeconds    = 0
)
