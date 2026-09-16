# API style decision: REST (not GraphQL)

## Decision

**Primary HTTP API: REST + OpenAPI 3.**  
MCP tools are thin adapters over the same domain services.  
**GraphQL is deferred** (optional later read projection for the web UI only).

## Why REST for an MCP-first AHP app

1. **MCP tools are discrete operations.** Each tool has a fixed JSON schema (name, inputs, outputs). That maps cleanly to resource endpoints (`POST /decisions`, `PATCH /criteria/:id`) and to OpenAPI operationIds → tool names.
2. **AHP is resource-oriented.** Decisions, criteria, alternatives, pairwise judgments, chats, and snapshots form a clear hierarchy. REST URLs mirror the mental model.
3. **Agents do not need client-driven query graphs.** An agent calls `list_criteria` then `set_pairwise`; it does not benefit from GraphQL field selection the way a dense SPA dashboard might.
4. **Simpler auth and caching.** Bearer tokens, standard HTTP status codes, and idempotent GETs are easier to reason about for both browser and MCP clients.
5. **OpenAPI is the contract.** One schema generates: TypeScript clients, MCP tool descriptors, and docs.

## Why not GraphQL (for now)

| GraphQL promise | Reality here |
|-----------------|--------------|
| Flexible nested reads | Domain services can return nested DTOs on REST; agents rarely need arbitrary nesting |
| One round-trip | MCP already batches via sequential tool use; chat latency dominates |
| Strong typing | OpenAPI + Zod/Drizzle already give strong typing |
| Ecosystem | Extra server, resolvers, and N+1 discipline without MVP payoff |

Revisit GraphQL only if the web UI accumulates deep, variable nested reads that hurt REST fan-out.

## REST surface (MVP)

Base path: `/api/v1`

| Method | Path | Purpose |
|--------|------|---------|
| GET/POST | `/decisions` | List / create decisions |
| GET/PATCH | `/decisions/:id` | Read / update decision |
| GET/POST | `/decisions/:id/criteria` | List / add criteria |
| PATCH/DELETE | `/criteria/:id` | Update / remove criterion |
| GET/PUT | `/decisions/:id/pairwise` | Read / upsert pairwise judgments |
| GET/POST | `/decisions/:id/snapshots` | List / create snapshots |
| GET | `/snapshots/:id` | Read snapshot (immutable) |
| GET/POST | `/chats` | List / create AI chat sessions |
| POST | `/chats/:id/messages` | Record prompt + summarized output |

All mutating routes require auth (session cookie or Bearer).

## MCP tools (MVP) — mirror REST

| Tool | Maps to |
|------|---------|
| `create_decision` | POST `/decisions` |
| `list_criteria` | GET `/decisions/:id/criteria` |
| `upsert_criterion` | POST/PATCH criteria |
| `propose_pairwise` | PUT pairwise (as **proposal** status) |
| `create_snapshot` | POST snapshots |
| `get_decision_state` | Aggregated GET for agent context |

MCP lives at `/api/mcp` (Streamable HTTP via `WebStandardStreamableHTTPServerTransport`, stateless + JSON responses). Tools call domain services directly — they do not HTTP-loopback unless required for boundary testing. See [mcp.md](./mcp.md).

## Error model

JSON body:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": {} } }
```

Use HTTP 401/403/404/409/422 consistently for both REST and MCP tool error payloads.
