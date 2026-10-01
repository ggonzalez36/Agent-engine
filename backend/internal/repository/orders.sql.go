package repository

import (
	"context"

	"github.com/google/uuid"
)

const createOrder = `-- name: CreateOrder :one
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
RETURNING id, customer_id, amount_cents, currency, status, metadata, created_at, updated_at
`

type CreateOrderParams struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  string    `json:"customer_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Metadata    []byte    `json:"metadata"`
}

func (q *Queries) CreateOrder(ctx context.Context, arg CreateOrderParams) (Order, error) {
	row := q.db.QueryRow(ctx, createOrder,
		arg.ID,
		arg.CustomerID,
		arg.AmountCents,
		arg.Currency,
		arg.Status,
		arg.Metadata,
	)
	var i Order
	err := row.Scan(
		&i.ID,
		&i.CustomerID,
		&i.AmountCents,
		&i.Currency,
		&i.Status,
		&i.Metadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getOrderByID = `-- name: GetOrderByID :one
SELECT id, customer_id, amount_cents, currency, status, metadata, created_at, updated_at FROM orders
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetOrderByID(ctx context.Context, id uuid.UUID) (Order, error) {
	row := q.db.QueryRow(ctx, getOrderByID, id)
	var i Order
	err := row.Scan(
		&i.ID,
		&i.CustomerID,
		&i.AmountCents,
		&i.Currency,
		&i.Status,
		&i.Metadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateOrderStatus = `-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, amount_cents, currency, status, metadata, created_at, updated_at
`

type UpdateOrderStatusParams struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

func (q *Queries) UpdateOrderStatus(ctx context.Context, arg UpdateOrderStatusParams) (Order, error) {
	row := q.db.QueryRow(ctx, updateOrderStatus, arg.ID, arg.Status)
	var i Order
	err := row.Scan(
		&i.ID,
		&i.CustomerID,
		&i.AmountCents,
		&i.Currency,
		&i.Status,
		&i.Metadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
