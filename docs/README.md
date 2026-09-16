# Docs index

Guidance for building the AI-first AHP web app. **Start here**, then follow links in order for new agents.

| Doc | Purpose |
|-----|---------|
| [VISION.md](./VISION.md) | Product problem, principles, success criteria |
| [architecture.md](./architecture.md) | Stack, system diagram, conventions |
| [chatkit.md](./chatkit.md) | **In-app ChatKit agent** (primary AI UX) |
| [api.md](./api.md) | **REST over GraphQL** decision + endpoint/tool map |
| [sso.md](./sso.md) | Human Auth.js + agent MCP OAuth / API keys |
| [schema.md](./schema.md) | Drizzle entities and snapshot policy |
| [MVP.md](./MVP.md) | Definition of done + workstream playbook |
| [mcp.md](./mcp.md) | Optional MCP Streamable HTTP + Cursor setup |
| [mcp-cursor.example.json](./mcp-cursor.example.json) | Example Cursor `mcp.json` |

## Quick decisions

- **Primary agent UX:** In-app OpenAI ChatKit (not Cursor embed). See [chatkit.md](./chatkit.md).
- **API:** REST + OpenAPI; ChatKit tools + MCP wrap domain services. GraphQL deferred.
- **SSO:** Web = Auth.js (GitHub/Google). In-app agent = signed `ck1` tokens. External agents = MCP API keys (OAuth 2.1 later).
- **Stack:** Next.js, Bun, Drizzle, Radix UI, Tabler icons, next-intl (`en` / `pt-BR`), Python ChatKit sidecar.
