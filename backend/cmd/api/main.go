package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/platform/backend/internal/handler"
	"github.com/platform/backend/internal/middleware"
	"github.com/platform/backend/internal/repository"
	"github.com/platform/backend/internal/service"
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

	serviceAuthToken := os.Getenv("SERVICE_AUTH_TOKEN")

	// 1. Connection Pool a PostgreSQL (pgx/v5)
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("Error analizando configuración de PostgreSQL", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 10
	poolConfig.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("No se pudo conectar al pool de PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 2. Cliente Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	// 3. Inyección de Dependencias
	queries := repository.New(pool)
	orderService := service.NewOrderService(pool)
	orderHandler := handler.NewOrderHandler(orderService, queries)
	sseHandler := handler.NewSSETelemetryHandler(rdb, logger)
	authMiddleware := middleware.NewServiceAuthMiddleware(serviceAuthToken)

	// 4. Configuración del Router HTTP (Librería Estándar net/http)
	mux := http.NewServeMux()

	// Endpoints de salud
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})

	// Endpoints de Órdenes (Transaccional)
	mux.HandleFunc("POST /api/v1/orders", orderHandler.CreateOrder)
	// Endpoints protegidos mediante Zero Trust ACL (Service-to-Service con Python)
	mux.HandleFunc("GET /api/v1/orders/{id}", authMiddleware.RequireServiceAuth(orderHandler.GetByID))
	mux.HandleFunc("PATCH /api/v1/orders/{id}/status", authMiddleware.RequireServiceAuth(orderHandler.UpdateStatus))

	// Endpoint SSE para Telemetría hacia el Frontend
	mux.Handle("GET /v1/telemetry/stream", sseHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // 0 requerido para no interrumpir streaming persistente SSE
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Iniciando servidor HTTP de Go", "port", 8080)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Servidor HTTP falló inesperadamente", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Señal de apagado recibida, cerrando servidor HTTP ordenadamente...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error en apagado ordenado del servidor HTTP", "error", err)
	}
	logger.Info("Servidor HTTP detenido con éxito")
}
