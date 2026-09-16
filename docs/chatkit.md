# In-app ChatKit agent

Primary AI surface for AHP Studio: a floating ChatKit widget over `/app` so users start with a prompt (“compare X vs Y”) while the agent fills criteria / alternatives / **proposal** judgments via tools.

Cursor MCP remains an optional power-user path — see [mcp.md](./mcp.md) and [sso.md](./sso.md).

## Architecture

```
Browser ChatKit widget
  → POST /api/chatkit/bootstrap   (Auth.js session → ck1 agent token)
  → POST /api/chatkit             (Next.js proxy)
  → Python FastAPI :8000/chatkit (openai-chatkit + Agents SDK)
  → REST /api/v1/*               (Bearer ck1… token → same users.id)
```

- Widget: [`src/components/chatkit/chatkit-panel.tsx`](../src/components/chatkit/chatkit-panel.tsx)
- Bootstrap: [`src/app/api/chatkit/bootstrap/route.ts`](../src/app/api/chatkit/bootstrap/route.ts)
- Proxy: [`src/app/api/chatkit/route.ts`](../src/app/api/chatkit/route.ts)
- Agent: [`agent/`](../agent/) (`openai-chatkit`, `openai-agents`, FastAPI)

## Local run

1. Copy env and set secrets:

```bash
cp .env.example .env
# AUTH_SECRET=…  AUTH_DEV_LOGIN=true  OPENAI_API_KEY=sk-…
```

2. Two processes:

```bash
bun run agent   # http://127.0.0.1:8000
bun run dev     # http://localhost:3000
```

3. Sign in → open `/app` → click the chat FAB → prompt e.g. “Compare iPhone 16 vs Pixel 9”.

4. Verify: a decision appears, criteria/alternatives fill, pairwise shows **Proposal**, and a `chats` row has `externalSessionId` = ChatKit thread id.

## Agent tools

| Tool | Effect |
|------|--------|
| `create_decision` | POST `/api/v1/decisions` |
| `get_decision_state` | GET `/api/v1/decisions/:id/state` |
| `list_criteria` / `upsert_criterion` | criteria REST |
| `create_alternative` | POST alternatives |
| `propose_pairwise` | PUT pairwise with `status: proposal` |
| `create_snapshot` | snapshot + prompt lineage |
| `ensure_chat_lineage` | POST `/api/v1/chats/ensure` |
| `refresh_decision_ui` | client tool → `router.refresh()` |

## Env

| Variable | Purpose |
|----------|---------|
| `OPENAI_API_KEY` | Required by the Python agent |
| `CHATKIT_AGENT_URL` | Upstream base (default `http://127.0.0.1:8000`) |
| `CHATKIT_PUBLIC_URL` | Widget URL (default `/api/chatkit`) |
| `CHATKIT_DOMAIN_KEY` | ChatKit `domainKey` (default `domain_pk_localhost`) |
| `AHP_API_BASE_URL` | Where the agent calls Next REST (default `http://127.0.0.1:3000`) |
| `AUTH_SECRET` | Signs/verifies `ck1.` agent tokens |
