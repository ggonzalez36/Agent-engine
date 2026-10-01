package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/platform/backend/internal/worker"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/platform_db?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Error analizando configuración de DB para Relay", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("Error conectando a PostgreSQL para Relay", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	relay := worker.NewRelayWorker(pool, rdb, worker.RelayConfig{
		PollInterval:   200 * time.Millisecond,
		BatchSize:      100,
		MaxConcurrency: 20,
	}, logger)

	logger.Info("Iniciando servicio de Outbox Relay Daemon...")
	if err := relay.Start(ctx); err != nil && err != context.Canceled {
		logger.Error("Relay Worker terminó con error", "error", err)
		os.Exit(1)
	}
	logger.Info("Outbox Relay Daemon finalizado limpiamente")
}
