"""HTTP client for AHP Studio REST (/api/v1) using ChatKit agent tokens."""

from __future__ import annotations

import os
from typing import Any

import httpx


class AhpRestClient:
    def __init__(self, agent_token: str, base_url: str | None = None) -> None:
        self.base_url = (
            base_url
            or os.getenv("AHP_API_BASE_URL", "http://127.0.0.1:3000")
        ).rstrip("/")
        self.agent_token = agent_token

    def _headers(self) -> dict[str, str]:
        return {
            "Authorization": f"Bearer {self.agent_token}",
            "Content-Type": "application/json",
            "Accept": "application/json",
        }

    async def request(
        self,
        method: str,
        path: str,
        *,
        json: dict[str, Any] | None = None,
    ) -> Any:
        url = f"{self.base_url}{path}"
        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.request(
                method,
                url,
                headers=self._headers(),
                json=json,
            )
            if response.status_code >= 400:
                detail = response.text
                try:
                    detail = response.json()
                except Exception:
                    pass
                raise RuntimeError(
                    f"AHP API {method} {path} failed ({response.status_code}): {detail}"
                )
            if response.status_code == 204 or not response.content:
                return None
            return response.json()
