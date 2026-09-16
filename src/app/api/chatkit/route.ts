import { AppError, isAppError } from "@/lib/ahp/errors";
import { jsonError } from "@/lib/ahp/http";
import {
  isChatKitAgentToken,
  verifyChatKitAgentToken,
} from "@/lib/auth/chatkit-token";

export const runtime = "nodejs";

const CORS_HEADERS: HeadersInit = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
  "Access-Control-Allow-Headers":
    "Content-Type, Authorization, OpenAI-Beta, mcp-protocol-version",
};

function agentBaseUrl(): string {
  return (
    process.env.CHATKIT_AGENT_URL?.trim().replace(/\/$/, "") ||
    "http://127.0.0.1:8000"
  );
}

function extractBearer(request: Request): string | null {
  const header = request.headers.get("authorization");
  if (!header) return null;
  const match = /^Bearer\s+(.+)$/i.exec(header.trim());
  return match?.[1]?.trim() || null;
}

/**
 * Same-origin proxy to the Python ChatKit server.
 * Validates the ChatKit agent token, then streams the upstream response.
 */
async function proxyChatKit(request: Request): Promise<Response> {
  try {
    const bearer = extractBearer(request);
    if (!bearer || !isChatKitAgentToken(bearer)) {
      throw new AppError(
        "UNAUTHORIZED",
        "ChatKit agent token required (bootstrap first)",
        401,
      );
    }
    const verified = verifyChatKitAgentToken(bearer);
    if (!verified) {
      throw new AppError("UNAUTHORIZED", "Invalid or expired ChatKit agent token", 401);
    }

    const upstreamUrl = `${agentBaseUrl()}/chatkit`;
    const headers = new Headers();
    headers.set("Authorization", `Bearer ${bearer}`);
    headers.set("X-AHP-User-Id", verified.userId);
    const contentType = request.headers.get("content-type");
    if (contentType) headers.set("Content-Type", contentType);

    const body =
      request.method === "GET" || request.method === "HEAD"
        ? undefined
        : await request.arrayBuffer();

    const upstream = await fetch(upstreamUrl, {
      method: request.method,
      headers,
      body,
    });

    const responseHeaders = new Headers(upstream.headers);
    for (const [key, value] of Object.entries(CORS_HEADERS)) {
      responseHeaders.set(key, value);
    }

    return new Response(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: responseHeaders,
    });
  } catch (err) {
    if (isAppError(err)) {
      return jsonError(err.code, err.message, err.status, err.details);
    }
    console.error("[chatkit proxy]", err);
    return jsonError(
      "INTERNAL_ERROR",
      "ChatKit agent unreachable — is `bun run agent` running?",
      502,
    );
  }
}

export async function OPTIONS() {
  return new Response(null, { status: 204, headers: CORS_HEADERS });
}

export async function POST(request: Request) {
  return proxyChatKit(request);
}

export async function GET(request: Request) {
  return proxyChatKit(request);
}
