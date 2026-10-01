package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platform/backend/internal/middleware"
)

func TestServiceAuthMiddleware(t *testing.T) {
	expectedToken := "secret-agent-token-12345"
	mw := middleware.NewServiceAuthMiddleware(expectedToken)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	protected := mw.RequireServiceAuth(dummyHandler)

	t.Run("Rechazar petición sin cabecera Authorization", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orders/123", nil)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("Esperado status 401 Unauthorized, obtenido %d", rec.Code)
		}
	})

	t.Run("Rechazar token con esquema no Bearer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orders/123", nil)
		req.Header.Set("Authorization", "Basic secret-agent-token-12345")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("Esperado status 401 Unauthorized, obtenido %d", rec.Code)
		}
	})

	t.Run("Rechazar token incorrecto con 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orders/123", nil)
		req.Header.Set("Authorization", "Bearer token-invalido-hacker")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("Esperado status 403 Forbidden, obtenido %d", rec.Code)
		}
	})

	t.Run("Permitir acceso con Bearer token legítimo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orders/123", nil)
		req.Header.Set("Authorization", "Bearer "+expectedToken)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Esperado status 200 OK, obtenido %d", rec.Code)
		}
		if rec.Body.String() != "ok" {
			t.Fatalf("Cuerpo de respuesta esperado 'ok', obtenido '%s'", rec.Body.String())
		}
	})
}
