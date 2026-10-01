"""Módulo de Emisión de Telemetría Granular hacia Redis Pub/Sub.

Permite a los nodos del motor agéntico transmitir eventos de progreso
hacia el backend de Go, el cual los retransmite al frontend vía Server-Sent Events (SSE).
"""

from __future__ import annotations

import json
import logging
import os
from datetime import datetime, timezone
from typing import Any
import redis.asyncio as aioredis

logger = logging.getLogger("telemetry_publisher")

_redis_client: aioredis.Redis | None = None


def get_redis_client() -> aioredis.Redis:
    global _redis_client
    if _redis_client is None:
        redis_url = os.getenv("REDIS_URL", "redis://localhost:6379/0")
        _redis_client = aioredis.from_url(redis_url, decode_responses=True)
    return _redis_client


async def emit_telemetry(
    correlation_id: str,
    event_type: str,
    data: dict[str, Any],
) -> None:
    """Emite un evento de telemetría tipado al canal Pub/Sub correspondiente."""
    try:
        client = get_redis_client()
        channel = f"telemetry:{correlation_id}"
        payload = json.dumps({
            "event_type": event_type,
            "correlation_id": correlation_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "data": data,
        })
        await client.publish(channel, payload)
    except Exception as exc:
        logger.warning(f"No se pudo emitir telemetría ({event_type}): {exc}")
