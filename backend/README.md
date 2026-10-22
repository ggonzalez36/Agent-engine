# Transactional Backend & Outbox Relay (Golang)

High-performance backend service built with Go 1.24 utilizing the standard library `net/http`, `pgx/v5`, and `sqlc` over PostgreSQL 16 and Redis 7.

## Architecture Highlights
- **Transactional Outbox Pattern:** Atomic double-write (`sqlc.WithTx()`) persisting orders and outbox events in the same ACID transaction.
- **Asynchronous Relay Worker:** High-performance polling with `SELECT ... FOR UPDATE SKIP LOCKED` backed by a partial index on `scheduled_at`.
- **Server-Sent Events (SSE) Telemetry:** Streaming multiplexer with keep-alive heartbeat (`: ping\n\n`) and clean connection cleanup.
- **Service-to-Service Access Control:** Constant-time comparison (`subtle.ConstantTimeCompare`) Bearer token authentication middleware for Service-to-Service authorization.

## Build & Test
```bash
# Build production binaries
go build -o bin/api ./cmd/api
go build -o bin/relay ./cmd/relay

# Run unit tests with race detection
go test -v -race ./...

# Run integration tests with Testcontainers
go test -v -tags=integration ./...
```
