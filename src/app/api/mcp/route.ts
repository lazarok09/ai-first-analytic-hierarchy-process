/**
 * MCP Streamable HTTP endpoint for the AI-first AHP app.
 *
 * Transport: `@modelcontextprotocol/sdk` WebStandardStreamableHTTPServerTransport
 * in **stateless** mode with `enableJsonResponse: true` so Next.js App Router
 * Route Handlers return ordinary JSON-RPC responses (no long-lived SSE session
 * map). This is compatible with Cursor remote MCP over HTTP.
 *
 * Auth: `Authorization: Bearer <api_key>` (hashed `api_keys` table), with
 * `x-user-id` fallback in development (see `requireUser`).
 *
 * Tools call domain services in `src/lib/ahp/*` directly — no HTTP loopback.
 */
import { WebStandardStreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/webStandardStreamableHttp.js";
import { isAppError } from "@/lib/ahp/errors";
import { jsonError } from "@/lib/ahp/http";
import { requireUser } from "@/lib/auth/require-user";
import { createAhpMcpServer } from "@/mcp/server";

export const runtime = "nodejs";

const CORS_HEADERS: HeadersInit = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, DELETE, OPTIONS",
  "Access-Control-Allow-Headers":
    "Content-Type, Authorization, x-user-id, mcp-session-id, Last-Event-ID, mcp-protocol-version",
  "Access-Control-Expose-Headers": "mcp-session-id, mcp-protocol-version",
};

function withCors(response: Response): Response {
  const headers = new Headers(response.headers);
  for (const [key, value] of Object.entries(CORS_HEADERS)) {
    headers.set(key, value);
  }
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}

export async function OPTIONS() {
  return new Response(null, { status: 204, headers: CORS_HEADERS });
}

async function handleMcp(request: Request): Promise<Response> {
  try {
    const user = await requireUser(request);
    const transport = new WebStandardStreamableHTTPServerTransport({
      // Stateless: one transport + server per request (fits serverless / Next).
      sessionIdGenerator: undefined,
      enableJsonResponse: true,
    });
    const server = createAhpMcpServer(user.id);
    await server.connect(transport);
    const response = await transport.handleRequest(request);
    // Best-effort cleanup after response is produced.
    void transport.close().catch(() => {});
    void server.close().catch(() => {});
    return withCors(response);
  } catch (err) {
    if (isAppError(err)) {
      return withCors(jsonError(err.code, err.message, err.status, err.details));
    }
    console.error("[mcp] route error", err);
    return withCors(jsonError("INTERNAL_ERROR", "Internal server error", 500));
  }
}

export async function POST(request: Request) {
  return handleMcp(request);
}

export async function GET(request: Request) {
  return handleMcp(request);
}

export async function DELETE(request: Request) {
  return handleMcp(request);
}
