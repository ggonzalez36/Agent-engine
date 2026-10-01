package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/platform/backend/internal/repository"
	"github.com/redis/go-redis/v9"
)

type RelayConfig struct {
	PollInterval   time.Duration
	BatchSize      int32
	MaxConcurrency int
}

type RelayWorker struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	queries     *repository.Queries
	cfg         RelayConfig
	logger      *slog.Logger
}

func NewRelayWorker(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	cfg RelayConfig,
	logger *slog.Logger,
) *RelayWorker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 250 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 10
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &RelayWorker{
		pool:        pool,
		redisClient: redisClient,
		queries:     repository.New(pool),
		cfg:         cfg,
		logger:      logger,
	}
}

// Start ejecuta el ciclo de vida del Relay Worker con graceful shutdown gobernado por ctx.
func (w *RelayWorker) Start(ctx context.Context) error {
	w.logger.Info("Iniciando Outbox Relay Worker",
		"poll_interval", w.cfg.PollInterval,
		"batch_size", w.cfg.BatchSize,
	)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Deteniendo Relay Worker limpiamente...")
			return ctx.Err()
		case <-ticker.C:
			processed, err := w.processBatch(ctx)
			if err != nil {
				w.logger.Error("Error en ciclo de procesamiento del Outbox", "error", err)
			}
			// Si procesamos un lote completo, no esperamos el ticker completo para drenar cola
			if processed == int(w.cfg.BatchSize) {
				ticker.Reset(10 * time.Millisecond)
			} else {
				ticker.Reset(w.cfg.PollInterval)
			}
		}
	}
}

// processBatch consulta eventos usando SELECT ... FOR UPDATE SKIP LOCKED
// garantizando que réplicas concurrentes en EKS no generen contención de locks.
func (w *RelayWorker) processBatch(ctx context.Context) (int, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("error iniciando tx para relay: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := w.queries.WithTx(tx)

	events, err := qtx.FetchPendingOutboxEvents(ctx, repository.FetchPendingOutboxEventsParams{
		ScheduledAt: time.Now().UTC(),
		Limit:       w.cfg.BatchSize,
	})
	if err != nil {
		return 0, fmt.Errorf("error en FetchPendingOutboxEvents: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	w.logger.Debug("Eventos pendientes tomados con lock exclusivo", "count", len(events))

	// Procesamiento en paralelo de publicación a Redis con rate-limiting mediante semáforo
	type publishResult struct {
		eventID uuid.UUID
		err     error
	}

	resultsChan := make(chan publishResult, len(events))
	sem := make(chan struct{}, w.cfg.MaxConcurrency)
	var wg sync.WaitGroup

	for _, event := range events {
		wg.Add(1)
		sem <- struct{}{}

		go func(evt repository.OutboxEvent) {
			defer wg.Done()
			defer func() { <-sem }()

			// 1. Publicación hacia Redis Pub/Sub (Garantía At-Least-Once concurrente)
			pubErr := w.redisClient.Publish(ctx, evt.Topic, evt.Payload).Err()
			resultsChan <- publishResult{
				eventID: evt.ID,
				err:     pubErr,
			}
		}(event)
	}

	wg.Wait()
	close(resultsChan)

	// 2. Aplicación secuencial y thread-safe de actualizaciones sobre la transacción de PostgreSQL
	for res := range resultsChan {
		if res.err != nil {
			w.logger.Warn("Fallo al publicar en Redis Pub/Sub, registrando reintento",
				"event_id", res.eventID,
				"error", res.err,
			)
			errMsg := res.err.Error()
			if markErr := qtx.MarkOutboxEventFailed(ctx, repository.MarkOutboxEventFailedParams{
				ID:        res.eventID,
				LastError: &errMsg,
			}); markErr != nil {
				w.logger.Error("Error marcando evento como FAILED", "event_id", res.eventID, "error", markErr)
			}
		} else {
			if markErr := qtx.MarkOutboxEventPublished(ctx, res.eventID); markErr != nil {
				w.logger.Error("Error marcando evento como PUBLISHED", "event_id", res.eventID, "error", markErr)
			}
		}
	}

	// Commit atómico de las actualizaciones de estado de los eventos procesados
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("error haciendo commit de eventos procesados: %w", err)
	}

	return len(events), nil
}
