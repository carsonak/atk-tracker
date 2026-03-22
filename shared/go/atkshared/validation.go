package atkshared

import "fmt"

func ValidateHeartbeatDuration(seconds int) error {
	if seconds < HeartbeatMinSeconds || seconds > HeartbeatMaxSeconds {
		return fmt.Errorf("duration must be between %d and %d seconds", HeartbeatMinSeconds, HeartbeatMaxSeconds)
	}
	return nil
}
