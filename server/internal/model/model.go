package model

import "time"

type Session struct {
	ID           string
	ApprenticeID string
	MachineID    string
	LoginTime    time.Time
	LogoutTime   *time.Time
}

type Heartbeat struct {
	ID            int64
	SessionID     string
	Timestamp     time.Time
	ActiveSeconds int
}

type DailySummary struct {
	Date              time.Time
	ApprenticeID      string
	TotalActiveMinute int
}
