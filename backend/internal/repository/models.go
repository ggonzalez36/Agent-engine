package repository

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  string    `json:"customer_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Metadata    []byte    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OutboxEvent struct {
	ID            uuid.UUID  `json:"id"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	EventType     string     `json:"event_type"`
	Topic         string     `json:"topic"`
	Payload       []byte     `json:"payload"`
	Status        string     `json:"status"`
	RetryCount    int32      `json:"retry_count"`
	MaxRetries    int32      `json:"max_retries"`
	LastError     *string    `json:"last_error"`
	CorrelationID string     `json:"correlation_id"`
	CausationID   string     `json:"causation_id"`
	CreatedAt     time.Time  `json:"created_at"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	ProcessedAt   *time.Time `json:"processed_at"`
}
