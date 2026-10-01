package repository

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	CreateOrder(ctx context.Context, arg CreateOrderParams) (Order, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (Order, error)
	UpdateOrderStatus(ctx context.Context, arg UpdateOrderStatusParams) (Order, error)
	InsertOutboxEvent(ctx context.Context, arg InsertOutboxEventParams) (OutboxEvent, error)
	FetchPendingOutboxEvents(ctx context.Context, arg FetchPendingOutboxEventsParams) ([]OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id uuid.UUID) error
	MarkOutboxEventFailed(ctx context.Context, arg MarkOutboxEventFailedParams) error
}

var _ Querier = (*Queries)(nil)
