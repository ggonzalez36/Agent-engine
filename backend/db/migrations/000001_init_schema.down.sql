-- 000001_init_schema.down.sql
DROP INDEX IF EXISTS idx_outbox_events_aggregate;
DROP INDEX IF EXISTS idx_outbox_events_correlation_id;
DROP INDEX IF EXISTS idx_outbox_events_pending_polling;
DROP TABLE IF EXISTS outbox_events;

DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_customer_id;
DROP TABLE IF EXISTS orders;
