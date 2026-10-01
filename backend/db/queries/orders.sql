-- name: CreateOrder :one
INSERT INTO orders (
    id,
    customer_id,
    amount_cents,
    currency,
    status,
    metadata,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW(), NOW()
)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
