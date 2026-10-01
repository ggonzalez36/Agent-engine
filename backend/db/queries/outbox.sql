-- name: InsertOutboxEvent :one
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
RETURNING *;

-- name: FetchPendingOutboxEvents :many
-- Bloqueo atómico no conflictivo entre múltiples réplicas del worker
SELECT * FROM outbox_events
WHERE status IN ('PENDING', 'PROCESSING')
  AND scheduled_at <= $1
  AND retry_count < max_retries
ORDER BY scheduled_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'PUBLISHED',
    processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET retry_count = retry_count + 1,
    last_error = $2,
    status = CASE WHEN retry_count + 1 >= max_retries THEN 'FAILED' ELSE 'PENDING' END,
    -- Backoff exponencial simple: 2 ^ retry_count segundos
    scheduled_at = NOW() + (POWER(2, retry_count) * INTERVAL '1 second')
WHERE id = $1;
