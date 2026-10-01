package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type SSETelemetryHandler struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

func NewSSETelemetryHandler(redisClient *redis.Client, logger *slog.Logger) *SSETelemetryHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SSETelemetryHandler{
		redisClient: redisClient,
		logger:      logger,
	}
}

// ServeHTTP maneja el streaming de telemetría Server-Sent Events hacia el frontend.
// Consume de canales Redis específicos por correlation_id emitidos por los agentes de LangGraph.
func (h *SSETelemetryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no soportado por el cliente/servidor", http.StatusInternalServerError)
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		http.Error(w, "El parámetro 'correlation_id' es requerido", http.StatusBadRequest)
		return
	}

	// Cabeceras estándares de SSE y desactivación de buffering en proxies/ingress
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	channelName := fmt.Sprintf("telemetry:%s", correlationID)
	pubsub := h.redisClient.Subscribe(r.Context(), channelName)
	defer pubsub.Close()

	ch := pubsub.Channel()

	h.logger.Info("Cliente SSE conectado a telemetría", "correlation_id", correlationID)

	// Notificación inicial de conexión establecida
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"STREAM_ESTABLISHED\",\"correlation_id\":\"%s\"}\n\n", correlationID)
	flusher.Flush()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.logger.Info("Cliente SSE desconectado", "correlation_id", correlationID)
			return

		case <-heartbeatTicker.C:
			// Comentario SSE de keep-alive para proxies y ALB
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				h.logger.Debug("Fallo enviando ping SSE, cliente desconectado", "error", err)
				return
			}
			flusher.Flush()

		case msg, ok := <-ch:
			if !ok {
				h.logger.Warn("Canal Redis cerrado para correlation_id", "correlation_id", correlationID)
				return
			}

			// Envío del chunk granular de telemetría
			_, err := fmt.Fprintf(w, "event: telemetry_event\ndata: %s\n\n", msg.Payload)
			if err != nil {
				h.logger.Error("Fallo al escribir en stream SSE", "error", err)
				return
			}
			flusher.Flush()
		}
	}
}
