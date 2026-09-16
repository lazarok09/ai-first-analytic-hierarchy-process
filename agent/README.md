# AHP ChatKit agent

Python FastAPI sidecar that implements the **self-hosted ChatKit** protocol for AHP Studio.

## Run

From the repo root (loads `../.env` for `OPENAI_API_KEY` / `AHP_API_BASE_URL`):

```bash
bun run agent
# or: cd agent && uv run uvicorn main:app --host 127.0.0.1 --port 8000 --reload
```

Health: `GET http://127.0.0.1:8000/health`  
Protocol: `POST http://127.0.0.1:8000/chatkit` (proxied by Next at `/api/chatkit`)

## Layout

- `main.py` — FastAPI app
- `ahp_chatkit/server.py` — `ChatKitServer` + Agents SDK runner
- `ahp_chatkit/tools.py` — AHP tools calling Next REST
- `ahp_chatkit/memory_store.py` — in-memory thread store (local MVP)
- `ahp_chatkit/rest.py` — httpx client with ChatKit agent bearer token

See [docs/chatkit.md](../docs/chatkit.md).
