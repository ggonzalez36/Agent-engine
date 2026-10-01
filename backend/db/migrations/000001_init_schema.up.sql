-- 000001_init_schema.up.sql
-- Extensiones requeridas para generación de UUIDs v4 deterministas
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tabla de Negocio: Pedidos / Órdenes (Entidad Transaccional de Dominio)
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id VARCHAR(64) NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(32) NOT NULL DEFAULT 'CREATED',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

-- Tabla Outbox: Registro inmutable de eventos transaccionales
-- Estados permitidos: PENDING, PROCESSING, PUBLISHED, FAILED
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    topic VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    last_error TEXT,
    correlation_id VARCHAR(128) NOT NULL,
    causation_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

-- ÍNDICE CRÍTICO DE ALTO RENDIMIENTO:
-- Permite que la consulta SELECT ... FOR UPDATE SKIP LOCKED ejecute un Index Scan
-- directo sobre los eventos pendientes sin escanear millones de registros PUBLISHED.
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending_polling
ON outbox_events (scheduled_at ASC)
WHERE status IN ('PENDING', 'PROCESSING');

CREATE INDEX IF NOT EXISTS idx_outbox_events_correlation_id ON outbox_events (correlation_id);
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate ON outbox_events (aggregate_type, aggregate_id);
