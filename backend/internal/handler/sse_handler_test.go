package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platform/backend/internal/handler"
)

func TestSSEHandlerMissingCorrelationID(t *testing.T) {
	sse := handler.NewSSETelemetryHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/telemetry/stream", nil)
	rec := httptest.NewRecorder()

	sse.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Esperado status 400 Bad Request por falta de correlation_id, obtenido %d", rec.Code)
	}
}
