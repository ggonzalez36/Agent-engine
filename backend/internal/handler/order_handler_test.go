package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platform/backend/internal/handler"
)

func TestOrderHandlerInvalidMethods(t *testing.T) {
	h := handler.NewOrderHandler(nil, nil)

	t.Run("Rechazar GET en CreateOrder", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("Esperado 405 Method Not Allowed, obtenido %d", rec.Code)
		}
	})

	t.Run("Rechazar payload JSON malformado en CreateOrder", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString("{invalido"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Esperado 400 Bad Request, obtenido %d", rec.Code)
		}
	})

	t.Run("Rechazar ID malformado en GetByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/no-es-uuid", nil)
		rec := httptest.NewRecorder()

		h.GetByID(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Esperado 400 Bad Request por UUID inválido, obtenido %d", rec.Code)
		}
	})
}
