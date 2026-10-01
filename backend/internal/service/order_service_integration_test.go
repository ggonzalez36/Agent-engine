//go:build integration
// +build integration

package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/platform/backend/internal/repository"
	"github.com/platform/backend/internal/service"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestOrderServiceOutboxAtomicIntegration valida con Testcontainers la persistencia
// atómica de la orden y del evento outbox dentro de la misma transacción en un motor real PostgreSQL.
func TestOrderServiceOutboxAtomicIntegration(t *testing.T) {
	ctx := context.Background()

	// 1. Inicialización del contenedor PostgreSQL mediante Testcontainers
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test_db"),
		tcpostgres.WithUsername("test_user"),
		tcpostgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Fallo iniciando contenedor PostgreSQL con Testcontainers: %v", err)
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Fallo obteniendo connection string del contenedor: %v", err)
	}

	// 2. Conexión al pool de pgx/v5
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("Error conectando a PostgreSQL de prueba: %v", err)
	}
	defer pool.Close()

	// 3. Aplicación del esquema DDL (migración 000001_init_schema.up.sql)
	schemaBytes, err := os.ReadFile("../../db/migrations/000001_init_schema.up.sql")
	if err != nil {
		t.Fatalf("Error leyendo archivo de migración DDL: %v", err)
	}

	if _, err := pool.Exec(ctx, string(schemaBytes)); err != nil {
		t.Fatalf("Error ejecutando DDL en base de datos de prueba: %v", err)
	}

	// 4. Invocación del servicio con Outbox Transaccional
	orderService := service.NewOrderService(pool)
	queries := repository.New(pool)

	corrID := "corr_" + uuid.NewString()
	causID := "cmd_" + uuid.NewString()

	order, err := orderService.CreateOrderWithOutbox(ctx, service.CreateOrderInput{
		CustomerID:    "cust_integration_test",
		AmountCents:   45000,
		Currency:      "USD",
		Metadata:      []byte(`{"source":"integration_suite"}`),
		CorrelationID: corrID,
		CausationID:   causID,
	})
	if err != nil {
		t.Fatalf("Fallo en CreateOrderWithOutbox: %v", err)
	}

	// 5. Asertos de verificación de atomicidad y consistencia de datos
	if order.Status != "CREATED" {
		t.Errorf("Estado de orden esperado 'CREATED', obtenido '%s'", order.Status)
	}

	// Verificar existencia de la orden en PostgreSQL
	savedOrder, err := queries.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("La orden no fue encontrada en la BD: %v", err)
	}
	if savedOrder.AmountCents != 45000 {
		t.Errorf("Monto esperado 45000, obtenido %d", savedOrder.AmountCents)
	}

	// Verificar existencia y consistencia del evento en outbox_events
	events, err := queries.FetchPendingOutboxEvents(ctx, repository.FetchPendingOutboxEventsParams{
		ScheduledAt: time.Now().UTC().Add(time.Minute),
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Fallo consultando outbox_events pendientes: %v", err)
	}

	foundOutbox := false
	for _, evt := range events {
		if evt.AggregateID == order.ID.String() && evt.CorrelationID == corrID {
			foundOutbox = true
			if evt.Status != "PENDING" {
				t.Errorf("Estado de outbox esperado 'PENDING', obtenido '%s'", evt.Status)
			}
			if evt.Topic != "orders.events" {
				t.Errorf("Topic esperado 'orders.events', obtenido '%s'", evt.Topic)
			}
			break
		}
	}

	if !foundOutbox {
		t.Fatal("El evento outbox asociado a la orden no fue encontrado en la base de datos")
	}
}
