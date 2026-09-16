# Architecture

## Stack (locked)

| Layer | Choice | Notes |
|-------|--------|-------|
| Runtime / package manager | **Bun** | Install, scripts, local tooling |
| App framework | **Next.js** (App Router) | Web UI + Route Handlers for REST |
| ORM / schema | **Drizzle** | Type-safe SQL; SQLite for local MVP, Postgres-ready |
| SQLite driver | **better-sqlite3** | Required for Next.js Node route handlers — do **not** use `bun:sqlite` in `src/db/index.ts` |
| UI primitives | **Radix UI** | Accessible headless components |
| Icons | **@tabler/icons-react** | Consistent icon set |
| Agent interface | **MCP** (Streamable HTTP) | Tools wrap the same domain services as REST |
| HTTP API | **REST + OpenAPI** | See [api.md](./api.md) — GraphQL deferred |
| Auth (humans) | **Auth.js (NextAuth v5)** | OAuth providers for the browser |
| Auth (agents) | **MCP OAuth 2.1 + PKCE** | Cursor / ChatGPT as MCP clients — see [sso.md](./sso.md) |
| i18n | **next-intl** | `en` + `pt-BR` |

## System shape

```
┌─────────────────┐     ┌─────────────────┐
│  Browser (human)│     │ Cursor / ChatGPT│
│  Next.js UI     │     │ MCP client      │
└────────┬────────┘     └────────┬────────┘
         │ Auth.js session        │ OAuth 2.1 bearer
         ▼                        ▼
┌────────────────────────────────────────────┐
│              Next.js app                    │
│  /api/v1/*  REST  │  /api/mcp  MCP server   │
│         └──── shared domain services ────┘  │
│                      │                      │
│                 Drizzle ORM                 │
│                      │                      │
│              SQLite / Postgres              │
└────────────────────────────────────────────┘
```

## Domain concepts

- **User** — authenticated human; owns decisions and chats.
- **Chat / AI session** — one modeling conversation; stores external AI session id.
- **Decision** — the question being answered (“Which phone?”).
- **Criterion / Alternative** — nodes in the AHP hierarchy.
- **Pairwise judgment** — Saaty-scale comparison (1–9) between two siblings under a parent.
- **Snapshot** — immutable point-in-time copy of the decision model + prompt + AI summary.
- **Data source record** — provenance for criteria/attributes pulled from the web (quality score, URL, discrepancy flags).

## Progressive UI rule

Render a UI block only when its backing data exists:

- Criteria list → when ≥1 criterion
- Importance / pairwise panel → when ≥2 siblings
- Consistency indicator → when a complete pairwise matrix exists
- Ranking / synthesis → when judgments are complete enough to compute

## Package layout (target)

```
apps/web/                 # or repo root if single package
  app/                    # Next.js App Router
  components/             # Radix-based UI
  db/                     # Drizzle schema + client
  lib/                    # domain services, auth, i18n
  mcp/                    # MCP server tools + handlers
  messages/               # en.json, pt-BR.json
docs/                     # this guidance
```

MVP may live at repo root as a single Next.js app for speed; split into monorepo only if needed later.

## Conventions for agents continuing this work

1. Read [VISION.md](./VISION.md), [api.md](./api.md), [sso.md](./sso.md), [schema.md](./schema.md), [MVP.md](./MVP.md) before large changes.
2. Prefer domain services (`lib/ahp/*`) shared by REST and MCP — never duplicate business logic in route handlers.
3. Never let MCP tools silently set final subjective weights without a user confirmation path in the product model (proposals vs committed judgments).
4. Every mutating agent action that changes the model should be snapshot-capable (at least create snapshot on “commit proposals”).
5. All user-facing strings go through i18n keys — no hard-coded English in components.
