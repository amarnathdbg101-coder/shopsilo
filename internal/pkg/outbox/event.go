package gooutbox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the lifecycle progression of an outbox event.
type Status string

const (
	// StatusPending indicates the event was committed to the outbox table and awaits dispatch.
	StatusPending Status = "pending"
	// StatusProcessing indicates the event was locked by a worker and is actively being published.
	StatusProcessing Status = "processing"
	// StatusCompleted indicates the event was successfully published and acknowledged.
	StatusCompleted Status = "completed"
	// StatusFailed indicates delivery failed and is scheduled for exponential backoff retry.
	StatusFailed Status = "failed"
	// StatusDeadLetter indicates all max retries were exhausted; placed into DLQ for manual inspection.
	StatusDeadLetter Status = "dead_letter"
)

// Event represents an outbox message written atomically alongside domain entities.
type Event struct {
	ID            string            `json:"id"`
	Topic         string            `json:"topic"`
	AggregateType string            `json:"aggregate_type,omitempty"`
	AggregateID   string            `json:"aggregate_id,omitempty"`
	Payload       []byte            `json:"payload"`
	Headers       map[string]string `json:"headers,omitempty"`
	Status        Status            `json:"status"`
	RetryCount    int               `json:"retry_count"`
	MaxRetries    int               `json:"max_retries"`
	LastError     string            `json:"last_error,omitempty"`
	NextRetryAt   time.Time         `json:"next_retry_at"`
	CreatedAt     time.Time         `json:"created_at"`
	ProcessedAt   *time.Time        `json:"processed_at,omitempty"`
}

// NewEvent creates a fresh Event with a cryptographically unique ID.
func NewEvent(topic, aggregateID string, payload any) (*Event, error) {
	if topic == "" {
		return nil, ErrEmptyTopic
	}

	var payloadBytes []byte
	if b, ok := payload.([]byte); ok {
		payloadBytes = b
	} else if s, ok := payload.(string); ok {
		payloadBytes = []byte(s)
	} else if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("go-outbox: marshal payload: %w", err)
		}
	} else {
		return nil, ErrEmptyPayload
	}

	if len(payloadBytes) == 0 {
		return nil, ErrEmptyPayload
	}

	id := generateID()
	now := time.Now().UTC()

	return &Event{
		ID:          id,
		Topic:       topic,
		AggregateID: aggregateID,
		Payload:     payloadBytes,
		Headers:     make(map[string]string),
		Status:      StatusPending,
		RetryCount:  0,
		MaxRetries:  5,
		NextRetryAt: now,
		CreatedAt:   now,
	}, nil
}

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
