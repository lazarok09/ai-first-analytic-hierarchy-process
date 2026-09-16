# SSO & agent authentication

## The question: “How will SSO connect?”

Cursor and ChatGPT do **not** offer a classic “Sign in with Cursor / ChatGPT” button that third-party apps can use like Google OAuth. Instead, **AI agents authenticate to your app as MCP clients** using the MCP Authorization specification (OAuth 2.1 + PKCE). Humans authenticate to the **web UI** with normal OAuth (Auth.js). Both paths must resolve to the **same user** in our database.

## Dual-path identity

```
Human (browser)                    Agent (Cursor / ChatGPT)
      │                                      │
      │ Auth.js OAuth                        │ MCP OAuth 2.1 + PKCE
      │ (Google / GitHub / …)                │ Dynamic Client Registration
      ▼                                      ▼
┌─────────────┐                      ┌──────────────────┐
│ Web session │                      │ Access token     │
│ cookie      │                      │ (Bearer, aud=MCP)│
└──────┬──────┘                      └────────┬─────────┘
       │                                      │
       └────────────► users.id ◄──────────────┘
```

## Path A — Human web SSO (Auth.js)

- Library: **Auth.js / NextAuth v5**
- MVP providers: **GitHub** + **Google** (widely available; map email → user)
- Optional later: OpenAI account linking if/when a stable OIDC provider is practical
- Session strategy: JWT or database sessions (database preferred once Drizzle users table exists)
- Protects: Next.js pages + REST when called with session cookie

## Path B — AI agent SSO (MCP OAuth)

Per MCP Authorization (2025–2026 revisions):

1. This app’s MCP endpoint is an **OAuth 2.1 resource server**.
2. Publish **Protected Resource Metadata** (RFC 9728) so clients discover the authorization server.
3. Support **PKCE**; prefer **Dynamic Client Registration** (RFC 7591) so Cursor/ChatGPT can register without manual pre-registration.
4. Issue access tokens whose **audience** is this MCP server (RFC 8707 resource indicators).
5. **Do not** pass the agent’s token through to third-party APIs (MCP forbids naive token passthrough). Use the bound `users.id` for all DB access.

### Cursor specifics

- Remote MCP via Streamable HTTP URL in `mcp.json`
- OAuth redirect used by Cursor desktop: `cursor://anysphere.cursor-mcp/oauth/callback`
- Optional static `auth.CLIENT_ID` / `CLIENT_SECRET` in `mcp.json` when DCR is disabled

### ChatGPT / other clients

- Same MCP OAuth endpoints; any client that implements MCP auth can connect
- Document the MCP URL and scopes in README

## MVP auth implementation plan

| Phase | Deliverable |
|-------|-------------|
| MVP-1 | Auth.js with GitHub (+ Google if secrets present); `users` + `accounts` tables |
| MVP-1 | Dev/local: personal access token / API key for MCP when OAuth AS is not fully wired |
| MVP-2 | Authorization server endpoints + PRM + PKCE code flow; link consent screen to logged-in user |
| MVP-2 | Cursor `mcp.json` example with OAuth |
| Later | WorkOS AuthKit / Connect if enterprise DCR + SSO become hard requirements |

For local demos before full OAuth AS: issue a **user-bound API key** from the web UI (“Connect agent”) and accept `Authorization: Bearer <api_key>` on `/api/mcp`. This unblocks Cursor/ChatGPT stdio or remote header auth while OAuth lands.

## Scopes (initial)

| Scope | Meaning |
|-------|---------|
| `ahp:read` | Read decisions, criteria, snapshots |
| `ahp:write` | Mutate model, create snapshots |
| `ahp:propose` | Write proposals only (not commit final judgments) |

Default agent grant: `ahp:read ahp:write ahp:propose`. Product UX should still treat pairwise **commit** as human-confirmable.

## Security notes

- Separate cookies (web) from bearer tokens (MCP)
- Rotate API keys; store hashes only
- Snapshot + audit log every agent write in MVP-2+
- CSRF protection for cookie-based REST mutations
