# AI-first Analytic Hierarchy Process

Cognitive AHP for the web: prompt-first ChatKit agent, live hierarchical matrices, MCP optional, and chronotopic snapshots.

Humans own subjective judgments. The in-app agent assembles criteria and data through typed tools backed by a **REST** API — it does not silently decide for you.

## Stack

- **Next.js** + **Bun**
- **Drizzle** + SQLite (`better-sqlite3`)
- **OpenAI ChatKit** (floating in-app agent) + Python FastAPI sidecar
- **Radix UI** + **Tabler** icons
- **Auth.js** (web) + ChatKit agent tokens / MCP Bearer API keys
- **next-intl** — English / Portuguese (Brazil)

## Docs (start here)

See [`docs/README.md`](./docs/README.md) for vision, architecture, ChatKit, API, SSO, schema, and MVP playbook.

### How agents connect

| Who | How |
|-----|-----|
| Human (browser) | Auth.js — GitHub, Google, or **Dev Login** (`AUTH_DEV_LOGIN=true`) |
| **In-app agent (primary)** | Floating **ChatKit** widget → `/api/chatkit` → Python agent tools → REST. See [`docs/chatkit.md`](./docs/chatkit.md). |
| Cursor / external MCP | Optional: `/api/mcp` with `Authorization: Bearer <api_key>` from **Connect agent**. |

Cursor does **not** ship an embeddable ChatKit-like widget for your product; use OpenAI ChatKit in-app.

## Quick start

```bash
cp .env.example .env
# AUTH_SECRET=…  AUTH_DEV_LOGIN=true  OPENAI_API_KEY=sk-…

bun install
bun run db:push   # optional if ensureSchema already created tables

# Terminal 1 — ChatKit agent
bun run agent

# Terminal 2 — Next.js
bun run dev
```

Open http://localhost:3000 → **Dev Login** → `/app` → open the chat FAB → prompt e.g. “Compare iPhone 16 vs Pixel 9”. Review **proposal** judgments in the UI and commit what you agree with.

### Optional: Cursor MCP

1. Create an API key in `/app/connect`
2. Copy [`docs/mcp-cursor.example.json`](./docs/mcp-cursor.example.json) into your Cursor MCP config
3. Replace the Bearer token with your key

## Scripts

| Script | Purpose |
|--------|---------|
| `bun run agent` | ChatKit Python agent (`:8000`) |
| `bun run dev` | Next.js dev server |
| `bun run build` / `start` | Production |
| `bun run typecheck` | TypeScript |
| `bun run db:push` | Push Drizzle schema |
| `bun run db:seed` | Demo data |
| `bun run db:studio` | Drizzle Studio |

## MVP status

- [x] Docs + architecture decisions
- [x] Drizzle schema (decisions, criteria, pairwise, snapshots, chats, api keys)
- [x] REST `/api/v1/*` + MCP `/api/mcp`
- [x] Auth.js + API keys + ChatKit agent tokens
- [x] In-app ChatKit floating agent (`docs/chatkit.md`)
- [x] UI: criteria, Saaty pairwise, snapshots, i18n en/pt-BR
- [ ] Full MCP OAuth AS (PRM + DCR) for Cursor/ChatGPT without API keys
- [ ] Data-quality / multi-source discrepancy UI
