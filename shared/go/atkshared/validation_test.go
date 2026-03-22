package atkshared

import (
"testing"
)

func TestValidateHeartbeatDuration_ValidRange(t *testing.T) {
tests := []struct {
name    string
seconds int
}{
{"zero", 0},
{"mid-range", 150},
{"max", 300},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
if err := ValidateHeartbeatDuration(tc.seconds); err != nil {
t.Fatalf("expected no error for %d, got %v", tc.seconds, err)
}
})
}
}

func TestValidateHeartbeatDuration_Invalid(t *testing.T) {
tests := []struct {
name    string
seconds int
}{
{"negative", -1},
{"just over max", 301},
{"large positive", 10000},
{"large negative", -999},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
if err := ValidateHeartbeatDuration(tc.seconds); err == nil {
t.Fatalf("expected error for %d, got nil", tc.seconds)
}
})
}
}

func TestValidateHeartbeatDuration_Boundary(t *testing.T) {
// Exactly at boundaries
if err := ValidateHeartbeatDuration(HeartbeatMinSeconds); err != nil {
t.Fatalf("expected no error at min boundary, got %v", err)
}
if err := ValidateHeartbeatDuration(HeartbeatMaxSeconds); err != nil {
t.Fatalf("expected no error at max boundary, got %v", err)
}
// Just outside boundaries
if err := ValidateHeartbeatDuration(HeartbeatMinSeconds - 1); err == nil {
t.Fatal("expected error just below min boundary")
}
if err := ValidateHeartbeatDuration(HeartbeatMaxSeconds + 1); err == nil {
t.Fatal("expected error just above max boundary")
}
}
