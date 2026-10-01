package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

type contextKey string

const (
	ServiceIdentityKey contextKey = "service_identity"
)

// ServiceAuthMiddleware implementa el control de acceso Zero Trust (ACL)
// para comunicaciones Service-to-Service (e.g., Python Agent Engine -> Go Backend).
type ServiceAuthMiddleware struct {
	expectedToken string
}

func NewServiceAuthMiddleware(expectedToken string) *ServiceAuthMiddleware {
	if expectedToken == "" {
		expectedToken = os.Getenv("SERVICE_AUTH_TOKEN")
	}
	return &ServiceAuthMiddleware{
		expectedToken: expectedToken,
	}
}

// RequireServiceAuth valida que la petición contenga un Bearer Token válido
// con comparación de tiempo constante (subtle.ConstantTimeCompare) para mitigar timing attacks.
func (m *ServiceAuthMiddleware) RequireServiceAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// En entornos donde se haya configurado el token de servicio
		if m.expectedToken != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized: falta cabecera Authorization"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"unauthorized: formato Bearer token inválido"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]
			// Comparación en tiempo constante para evitar ataques de canal lateral (Timing Attack)
			if subtle.ConstantTimeCompare([]byte(token), []byte(m.expectedToken)) != 1 {
				http.Error(w, `{"error":"forbidden: credenciales de servicio no autorizadas"}`, http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), ServiceIdentityKey, "agent-engine-service")
		next(w, r.WithContext(ctx))
	}
}
