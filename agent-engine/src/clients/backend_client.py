"""Cliente HTTP de alta resiliencia para el Backend Transaccional en Go.

Cumple estrictamente con el principio Zero Trust:
- El motor en Python carece de credenciales o drivers para PostgreSQL.
- Toda lectura o mutación se realiza sobre la API REST de Go mediante HTTP tipado.
"""

from __future__ import annotations

import os
from typing import Any
import httpx
from tenacity import retry, retry_if_exception_type, stop_after_attempt, wait_exponential


class BackendClientError(Exception):
    """Error base de comunicación con el backend transaccional."""
    pass


class BackendClient:
    def __init__(
        self,
        base_url: str | None = None,
        auth_token: str | None = None,
        timeout_seconds: float = 5.0,
    ) -> None:
        self.base_url = (base_url or os.getenv("BACKEND_API_URL", "http://localhost:8080")).rstrip("/")
        self.auth_token = auth_token or os.getenv("SERVICE_AUTH_TOKEN", "")
        self.timeout = timeout_seconds

    def _headers(self) -> dict[str, str]:
        headers = {
            "Content-Type": "application/json",
            "User-Agent": "LangGraph-AgentEngine/1.0",
        }
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        return headers

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=0.5, min=0.5, max=2.0),
        retry=retry_if_exception_type((httpx.RequestError, httpx.HTTPStatusError)),
        reraise=True,
    )
    async def get_order(self, order_id: str) -> dict[str, Any]:
        """Consulta los detalles de una orden en Go (Read-Only)."""
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.get(
                f"{self.base_url}/api/v1/orders/{order_id}",
                headers=self._headers(),
            )
            resp.raise_for_status()
            return resp.json()

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=0.5, min=0.5, max=2.0),
        retry=retry_if_exception_type((httpx.RequestError, httpx.HTTPStatusError)),
        reraise=True,
    )
    async def update_order_status(self, order_id: str, new_status: str) -> dict[str, Any]:
        """Mutación de estado transaccional autorizada en Go."""
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.patch(
                f"{self.base_url}/api/v1/orders/{order_id}/status",
                headers=self._headers(),
                json={"status": new_status},
            )
            resp.raise_for_status()
            return resp.json()
