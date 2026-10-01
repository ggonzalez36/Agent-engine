package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const insertOutboxEvent = `-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    topic,
    payload,
    status,
    retry_count,
    max_retries,
    correlation_id,
    causation_id,
    created_at,
    scheduled_at
) VALUES (
    $1, $2, $3, $4, $5, $6, 'PENDING', 0, $7, $8, $9, NOW(), NOW()
)
RETURNING id, aggregate_type, aggregate_id, event_type, topic, payload, status, retry_count, max_retries, last_error, correlation_id, causation_id, created_at, scheduled_at, processed_at
`

type InsertOutboxEventParams struct {
	ID            uuid.UUID `json:"id"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	EventType     string    `json:"event_type"`
	Topic         string    `json:"topic"`
	Payload       []byte    `json:"payload"`
	MaxRetries    int32     `json:"max_retries"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
}

func (q *Queries) InsertOutboxEvent(ctx context.Context, arg InsertOutboxEventParams) (OutboxEvent, error) {
	row := q.db.QueryRow(ctx, insertOutboxEvent,
		arg.ID,
		arg.AggregateType,
		arg.AggregateID,
		arg.EventType,
		arg.Topic,
		arg.Payload,
		arg.MaxRetries,
		arg.CorrelationID,
		arg.CausationID,
	)
	var i OutboxEvent
	err := row.Scan(
		&i.ID,
		&i.AggregateType,
		&i.AggregateID,
		&i.EventType,
		&i.Topic,
		&i.Payload,
		&i.Status,
		&i.RetryCount,
		&i.MaxRetries,
		&i.LastError,
		&i.CorrelationID,
		&i.CausationID,
		&i.CreatedAt,
		&i.ScheduledAt,
		&i.ProcessedAt,
	)
	return i, err
}

const fetchPendingOutboxEvents = `-- name: FetchPendingOutboxEvents :many
SELECT id, aggregate_type, aggregate_id, event_type, topic, payload, status, retry_count, max_retries, last_error, correlation_id, causation_id, created_at, scheduled_at, processed_at FROM outbox_events
WHERE status IN ('PENDING', 'PROCESSING')
  AND scheduled_at <= $1
  AND retry_count < max_retries
ORDER BY scheduled_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED
`

type FetchPendingOutboxEventsParams struct {
	ScheduledAt time.Time `json:"scheduled_at"`
	Limit       int32     `json:"limit"`
}

func (q *Queries) FetchPendingOutboxEvents(ctx context.Context, arg FetchPendingOutboxEventsParams) ([]OutboxEvent, error) {
	rows, err := q.db.Query(ctx, fetchPendingOutboxEvents, arg.ScheduledAt, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OutboxEvent
	for rows.Next() {
		var i OutboxEvent
		if err := rows.Scan(
			&i.ID,
			&i.AggregateType,
			&i.AggregateID,
			&i.EventType,
			&i.Topic,
			&i.Payload,
			&i.Status,
			&i.RetryCount,
			&i.MaxRetries,
			&i.LastError,
			&i.CorrelationID,
			&i.CausationID,
			&i.CreatedAt,
			&i.ScheduledAt,
			&i.ProcessedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const markOutboxEventPublished = `-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'PUBLISHED',
    processed_at = NOW()
WHERE id = $1
`

func (q *Queries) MarkOutboxEventPublished(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, markOutboxEventPublished, id)
	return err
}

const markOutboxEventFailed = `-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET retry_count = retry_count + 1,
    last_error = $2,
    status = CASE WHEN retry_count + 1 >= max_retries THEN 'FAILED' ELSE 'PENDING' END,
    scheduled_at = NOW() + (POWER(2, retry_count) * INTERVAL '1 second')
WHERE id = $1
`

type MarkOutboxEventFailedParams struct {
	ID        uuid.UUID `json:"id"`
	LastError *string   `json:"last_error"`
}

func (q *Queries) MarkOutboxEventFailed(ctx context.Context, arg MarkOutboxEventFailedParams) error {
	_, err := q.db.Exec(ctx, markOutboxEventFailed, arg.ID, arg.LastError)
	return err
}
