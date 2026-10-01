package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/platform/backend/internal/domain"
	"github.com/platform/backend/internal/repository"
)

type CreateOrderInput struct {
	CustomerID    string          `json:"customer_id"`
	AmountCents   int64           `json:"amount_cents"`
	Currency      string          `json:"currency"`
	Metadata      json.RawMessage `json:"metadata"`
	CorrelationID string          `json:"correlation_id"`
	CausationID   string          `json:"causation_id"`
}

type OrderService struct {
	pool    *pgxpool.Pool
	queries *repository.Queries
}

func NewOrderService(pool *pgxpool.Pool) *OrderService {
	return &OrderService{
		pool:    pool,
		queries: repository.New(pool),
	}
}

// CreateOrderWithOutbox garantiza persistencia atómica mediante sqlc.WithTx().
// Escribe la entidad transaccional 'orders' y el evento en 'outbox_events'
// en la misma transacción ACID de PostgreSQL, eliminando el problema de la doble escritura.
func (s *OrderService) CreateOrderWithOutbox(ctx context.Context, input CreateOrderInput) (*repository.Order, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	if input.CustomerID == "" {
		return nil, fmt.Errorf("customer_id es mandatorio")
	}
	if input.CorrelationID == "" {
		input.CorrelationID = "corr_" + uuid.NewString()
	}
	if input.CausationID == "" {
		input.CausationID = "cmd_" + uuid.NewString()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción postgres: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := s.queries.WithTx(tx)

	orderID := uuid.New()
	orderMetadata := input.Metadata
	if len(orderMetadata) == 0 {
		orderMetadata = []byte("{}")
	}

	// 1. Escritura de Entidad de Dominio
	order, err := qtx.CreateOrder(ctx, repository.CreateOrderParams{
		ID:          orderID,
		CustomerID:  input.CustomerID,
		AmountCents: input.AmountCents,
		Currency:    input.Currency,
		Status:      "CREATED",
		Metadata:    orderMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("error insertando orden: %w", err)
	}

	// 2. Construcción de Envelope CloudEvents v1.0
	payload := domain.OrderCreatedPayload{
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		AmountCents: order.AmountCents,
		Currency:    order.Currency,
		Status:      order.Status,
	}

	cloudEvent, eventBytes, err := domain.NewCloudEvent(
		"order.created.v1",
		"backend.orders.service",
		input.CorrelationID,
		input.CausationID,
		payload,
	)
	if err != nil {
		return nil, fmt.Errorf("error serializando cloudevent: %w", err)
	}

	// 3. Escritura Atómica en la tabla outbox_events
	_, err = qtx.InsertOutboxEvent(ctx, repository.InsertOutboxEventParams{
		ID:            uuid.New(),
		AggregateType: "order",
		AggregateID:   order.ID.String(),
		EventType:     cloudEvent.Type,
		Topic:         "orders.events",
		Payload:       eventBytes,
		MaxRetries:    5,
		CorrelationID: cloudEvent.CorrelationID,
		CausationID:   cloudEvent.CausationID,
	})
	if err != nil {
		return nil, fmt.Errorf("error persistiendo outbox event: %w", err)
	}

	// 4. Commit Atómico
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error haciendo commit de la transacción: %w", err)
	}

	return &order, nil
}
