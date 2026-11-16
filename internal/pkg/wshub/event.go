package gowshub

import (
	"encoding/json"
	"time"
)

// Event standardizes real-time WebSocket payloads across microservices and clients.
type Event struct {
	Event     string `json:"event"`
	Payload   any    `json:"payload"`
	Timestamp string `json:"timestamp"`
}

// NewEvent builds a standardized Event stamped with current RFC3339 timestamp.
func NewEvent(name string, payload any) *Event {
	return &Event{
		Event:     name,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Bytes marshals the event to JSON byte slice.
func (e *Event) Bytes() ([]byte, error) {
	return json.Marshal(e)
}
