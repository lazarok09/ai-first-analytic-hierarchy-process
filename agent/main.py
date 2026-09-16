"""FastAPI entrypoint for the AHP ChatKit agent."""

from __future__ import annotations

import os
from typing import Any

from dotenv import load_dotenv
from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from chatkit.server import StreamingResult

from ahp_chatkit.memory_store import MemoryStore
from ahp_chatkit.server import AhpChatKitServer

# Load repo-root .env when running from agent/
load_dotenv(dotenv_path=os.path.join(os.path.dirname(__file__), "..", ".env"))
load_dotenv()

app = FastAPI(title="AHP ChatKit Agent", version="0.1.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

store = MemoryStore()
server = AhpChatKitServer(store)


def _context_from_request(request: Request) -> dict[str, Any]:
    auth = request.headers.get("authorization") or ""
    token = ""
    if auth.lower().startswith("bearer "):
        token = auth.split(" ", 1)[1].strip()
    user_id = request.headers.get("x-ahp-user-id") or ""
    return {
        "agent_token": token,
        "user_id": user_id,
    }


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/chatkit")
async def chatkit_endpoint(request: Request):
    context = _context_from_request(request)
    if not context["agent_token"]:
        return Response(
            content='{"error":"missing agent token"}',
            status_code=401,
            media_type="application/json",
        )

    result = await server.process(await request.body(), context)
    if isinstance(result, StreamingResult):
        return StreamingResponse(result, media_type="text/event-stream")
    return Response(content=result.json, media_type="application/json")


def main() -> None:
    import uvicorn

    host = os.getenv("CHATKIT_HOST", "127.0.0.1")
    port = int(os.getenv("CHATKIT_PORT", "8000"))
    uvicorn.run("main:app", host=host, port=port, reload=True)


if __name__ == "__main__":
    main()
