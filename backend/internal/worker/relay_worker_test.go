package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/platform/backend/internal/worker"
)

func TestRelayWorkerDefaultConfig(t *testing.T) {
	w := worker.NewRelayWorker(nil, nil, worker.RelayConfig{}, nil)
	if w == nil {
		t.Fatal("Se esperaba una instancia de RelayWorker y fue nil")
	}
}

func TestRelayWorkerGracefulShutdown(t *testing.T) {
	w := worker.NewRelayWorker(nil, nil, worker.RelayConfig{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    10,
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// El worker debe respetar la cancelación de contexto y salir con context.Canceled o DeadlineExceeded
	err := w.Start(ctx)
	if err == nil {
		t.Fatal("Se esperaba error por cancelación de contexto y fue nil")
	}
}
