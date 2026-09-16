# MVP checklist & agent playbook

## MVP definition of done

A developer can:

1. `bun install && bun run agent && bun run dev` ✅
2. Sign in with GitHub (or dev credentials) ✅ (`AUTH_DEV_LOGIN=true`)
3. Open the ChatKit FAB and prompt a comparison (agent creates decision/criteria) ✅
4. Adjust criterion importance / pairwise Saaty values in the UI ✅
5. (Optional) Create an API key, point Cursor MCP at `/api/mcp` ✅
6. See UI update; create a snapshot with prompt + summary ✅
7. Switch locale between English and Portuguese (Brazil) ✅

## Workstreams (spawn agents against these)

### W1 — Scaffold
- Next.js App Router + TypeScript + Bun
- Tailwind (minimal; Radix unstyled)
- ESLint, `bun run typecheck`, `bun run db:*` scripts
- `.env.example`

### W2 — Database
- Drizzle schema per [schema.md](./schema.md)
- Migrate + seed script (demo decision optional)

### W3 — Domain + REST
- `lib/ahp/*` services
- `/api/v1/*` route handlers + Zod validation
- OpenAPI stub or zod-to-openapi optional

### W4 — Auth
- Auth.js + Drizzle adapter
- API keys for MCP bearer
- Middleware protecting `/api/v1` and app routes

### W5 — MCP
- Streamable HTTP MCP server at `/api/mcp`
- Tools: create_decision, list/upsert criteria, get_decision_state, propose_pairwise, create_snapshot
- Auth via Bearer API key (OAuth AS can follow)

### W6 — UI
- Decision page: criteria list (Radix), add/edit
- Importance / pairwise matrix editor (Saaty 1–9)
- Progressive reveal of matrix when ≥2 items
- Tabler icons
- next-intl `en` / `pt-BR`

### W7 — Docs polish
- Root README: run, env, Cursor mcp.json example
- Keep these docs authoritative

## Agent rules

- Follow [architecture.md](./architecture.md) conventions.
- Do not introduce GraphQL.
- Do not let agents auto-commit final judgments without `status: proposal` distinction.
- Prefer small PRs / commits of vertical slices.
- After each stream: `bun run typecheck` (and tests if present).

## Post-MVP backlog

- Full MCP OAuth AS + PRM + DCR
- Data quality gates + discrepancy UI
- Live matrix animation as cells fill
- Multi-source fusion workflows
- Consistency ratio (CR) display and guided repair
