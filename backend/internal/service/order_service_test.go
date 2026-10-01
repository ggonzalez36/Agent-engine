package service_test

import (
	"context"
	"testing"

	"github.com/platform/backend/internal/domain"
	"github.com/platform/backend/internal/service"
)

func TestCreateOrderInputValidation(t *testing.T) {
	svc := service.NewOrderService(nil)

	t.Run("Rechazar monto menor o igual a cero", func(t *testing.T) {
		_, err := svc.CreateOrderWithOutbox(context.Background(), service.CreateOrderInput{
			CustomerID:  "cust_123",
			AmountCents: 0,
			Currency:    "USD",
		})
		if err == nil {
			t.Fatal("Se esperaba error por monto menor o igual a cero y fue nil")
		}
	})

	t.Run("Rechazar customer_id vacío", func(t *testing.T) {
		_, err := svc.CreateOrderWithOutbox(context.Background(), service.CreateOrderInput{
			CustomerID:  "",
			AmountCents: 1000,
			Currency:    "USD",
		})
		if err == nil {
			t.Fatal("Se esperaba error por customer_id vacío y fue nil")
		}
	})
}

func TestCloudEventEnvelopeConstruction(t *testing.T) {
	payload := domain.OrderCreatedPayload{
		CustomerID:  "cust_test_1",
		AmountCents: 5000,
		Currency:    "USD",
		Status:      "CREATED",
	}

	event, bytes, err := domain.NewCloudEvent(
		"order.created.v1",
		"backend.orders.service",
		"corr_test_123",
		"cmd_test_456",
		payload,
	)

	if err != nil {
		t.Fatalf("Fallo inesperado creando CloudEvent: %v", err)
	}

	if event.SpecVersion != "1.0" {
		t.Errorf("SpecVersion esperado '1.0', obtenido '%s'", event.SpecVersion)
	}

	if event.Type != "order.created.v1" {
		t.Errorf("Tipo esperado 'order.created.v1', obtenido '%s'", event.Type)
	}

	if event.CorrelationID != "corr_test_123" {
		t.Errorf("CorrelationID esperado 'corr_test_123', obtenido '%s'", event.CorrelationID)
	}

	if len(bytes) == 0 {
		t.Error("Los bytes del sobre serializado no deben estar vacíos")
	}
}
