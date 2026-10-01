"""Consumidor Asíncrono de Eventos Outbox desde Redis Pub/Sub.

Garantiza:
1. Idempotencia y deduplicación frente a entregas "At-Least-Once".
2. Sanitización estricta de inputs antes de activar el Supervisor.
3. Ejecución controlada con recursion_limit explícito.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
from typing import Any
import redis.asyncio as aioredis
from langchain_core.messages import HumanMessage
from src.graph.supervisor import run_agent_workflow
from src.security.sanitizer import sanitize_text
from src.state.agent_state import AgentState

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("redis_consumer")


class OutboxEventConsumer:
    def __init__(
        self,
        redis_url: str | None = None,
        topic: str = "orders.events",
        recursion_limit: int = 15,
    ) -> None:
        self.redis_url = redis_url or os.getenv("REDIS_URL", "redis://localhost:6379/0")
        self.topic = topic
        self.recursion_limit = recursion_limit
        self.redis_client = aioredis.from_url(self.redis_url, decode_responses=True)

    async def _is_duplicate_event(self, event_id: str) -> bool:
        """Control de idempotencia: Bloqueo atómico con TTL de 24 horas en Redis."""
        key = f"processed_event:{event_id}"
        # setnx retorna True solo si la clave fue adquirida por primera vez
        acquired = await self.redis_client.set(key, "1", nx=True, ex=86400)
        return not acquired

    async def process_event(self, raw_message: str) -> None:
        try:
            event_envelope: dict[str, Any] = json.loads(raw_message)
            event_id = event_envelope["id"]
            correlation_id = event_envelope.get("correlation_id", "corr_unknown")
            causation_id = event_envelope.get("causation_id", "cmd_unknown")
            event_type = event_envelope.get("type", "")

            # 1. Chequeo de idempotencia
            if await self._is_duplicate_event(event_id):
                logger.warning(f"Evento duplicado ignorado por idempotencia: {event_id}")
                return

            logger.info(f"Procesando evento {event_type} con ID {event_id} (corr: {correlation_id})")

            data = event_envelope.get("data", {})
            order_id = str(data.get("order_id", ""))
            customer_id = sanitize_text(str(data.get("customer_id", "")))

            # 2. Inicialización de AgentState Inmutable
            initial_state: AgentState = {
                "messages": [
                    HumanMessage(content=f"Iniciar procesamiento y auditoría para la orden: {order_id}")
                ],
                "order_id": order_id,
                "customer_id": customer_id,
                "correlation_id": correlation_id,
                "causation_id": causation_id,
                "current_phase": "INITIALIZED",
                "risk_assessment": None,
                "compliance_check": None,
                "audit_trail": [{
                    "step": "event_consumption",
                    "agent": "redis_consumer",
                    "action": "CONSUME_EVENT",
                    "timestamp": os.getenv("CURRENT_TIME", ""),
                    "details": {"event_id": event_id, "event_type": event_type},
                }],
                "errors": [],
            }

            # 3. Disparo del Grafo con límite de recursión explícito
            final_state = await run_agent_workflow(
                initial_state=initial_state,
                recursion_limit=self.recursion_limit,
            )

            logger.info(
                f"Flujo agéntico culminado para orden {order_id}. "
                f"Fase final: {final_state.get('current_phase')}, "
                f"Errores: {len(final_state.get('errors', []))}"
            )

        except Exception as exc:
            logger.error(f"Error crítico procesando mensaje outbox: {exc}", exc_info=True)

    async def start(self) -> None:
        """Inicia el bucle de suscripción a Redis Pub/Sub."""
        pubsub = self.redis_client.pubsub()
        await pubsub.subscribe(self.topic)
        logger.info(f"Suscrito a canal Redis '{self.topic}'. Esperando eventos outbox...")

        try:
            async for message in pubsub.listen():
                if message["type"] == "message":
                    asyncio.create_task(self.process_event(message["data"]))
        finally:
            await pubsub.unsubscribe(self.topic)
            await self.redis_client.aclose()


if __name__ == "__main__":
    consumer = OutboxEventConsumer()
    asyncio.run(consumer.start())
