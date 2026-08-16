package heartbeat

import "time"

// Payload is the liveness signal a worker sends to a coordinator.
type Payload struct {
	Worker    string    `json:"worker"`
	Timestamp time.Time `json:"timestamp"`
}
