# MCP endpoint

## What we implemented

Streamable HTTP MCP at **`POST/GET/DELETE /api/mcp`** using:

- `@modelcontextprotocol/sdk` **`McpServer`**
- **`WebStandardStreamableHTTPServerTransport`** (Web Fetch `Request`/`Response` — fits Next.js App Router)
- **Stateless** mode (`sessionIdGenerator: undefined`) + **`enableJsonResponse: true`**

JSON responses avoid holding an in-memory SSE session map across Next.js invocations. Cursor remote MCP works with this shape.

Tools call `src/lib/ahp/*` domain services directly (no HTTP loopback to `/api/v1`).

## Auth

1. Preferred: `Authorization: Bearer <api_key>` (`ahp_<prefix>_<secret>`, stored hashed in `api_keys`)
2. Development: `x-user-id: <users.id>` when `NODE_ENV !== "production"` or `ALLOW_X_USER_ID=1`

## Tools

| Tool | Notes |
|------|--------|
| `create_decision` | Create decision |
| `get_decision_state` | Decision + criteria + alternatives + pairwise |
| `list_criteria` | List criteria for a decision |
| `upsert_criterion` | Create (`decisionId`) or update (`criterionId`) |
| `propose_pairwise` | **Always** `status: proposal` |
| `create_snapshot` | Requires `prompt` + `outputSummary` |

## Cursor

See [mcp-cursor.example.json](./mcp-cursor.example.json). Copy into Cursor MCP settings (or `.cursor/mcp.json`).

**Dev without an API key yet** — use only `x-user-id` (do not send a fake `Authorization` header; invalid Bearer fails before the header fallback):

```json
{
  "mcpServers": {
    "ahp": {
      "url": "http://localhost:3000/api/mcp",
      "headers": {
        "x-user-id": "YOUR_DEV_USER_UUID"
      }
    }
  }
}
```

Create a real key with `createApiKey` (`src/lib/auth/api-keys.ts`) or the web “Connect agent” UI when available, then switch to `Authorization: Bearer ahp_…`.
