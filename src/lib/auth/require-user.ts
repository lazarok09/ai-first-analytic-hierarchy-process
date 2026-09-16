import { eq } from "drizzle-orm";
import { auth } from "@/auth";
import { db } from "@/db";
import { users } from "@/db/schema";
import { AppError } from "@/lib/ahp/errors";
import { verifyApiKey } from "@/lib/auth/api-keys";
import {
  isChatKitAgentToken,
  verifyChatKitAgentToken,
} from "@/lib/auth/chatkit-token";

export type AuthUser = {
  id: string;
};

/**
 * Resolve the authenticated user for REST / domain calls.
 *
 * Order:
 * 1. Auth.js session cookie (`auth()`)
 * 2. `Authorization: Bearer <api_key | ck1 agent token>`
 * 3. `x-user-id` — development / ALLOW_X_USER_ID only (auto-provisions stub user)
 */
export async function requireUser(request: Request): Promise<AuthUser> {
  const session = await auth();
  if (session?.user?.id) {
    return { id: session.user.id };
  }

  const bearer = extractBearerToken(request);
  if (bearer) {
    if (isChatKitAgentToken(bearer)) {
      const agent = verifyChatKitAgentToken(bearer);
      if (agent) {
        return { id: agent.userId };
      }
      throw new AppError("UNAUTHORIZED", "Invalid or expired ChatKit agent token", 401);
    }
    const verified = await verifyApiKey(bearer);
    if (verified) {
      return { id: verified.userId };
    }
    throw new AppError("UNAUTHORIZED", "Invalid or revoked API key", 401);
  }

  const allowHeader =
    process.env.NODE_ENV !== "production" ||
    process.env.ALLOW_X_USER_ID === "1";

  if (allowHeader) {
    const headerId = request.headers.get("x-user-id")?.trim();
    if (headerId) {
      await ensureDevUser(headerId);
      return { id: headerId };
    }
  }

  throw new AppError(
    "UNAUTHORIZED",
    "Authentication required (sign in, or pass Authorization: Bearer <api_key>)",
    401,
  );
}

function extractBearerToken(request: Request): string | null {
  const header = request.headers.get("authorization");
  if (!header) return null;
  const match = /^Bearer\s+(.+)$/i.exec(header.trim());
  return match?.[1]?.trim() || null;
}

async function ensureDevUser(userId: string): Promise<void> {
  const existing = await db
    .select({ id: users.id })
    .from(users)
    .where(eq(users.id, userId))
    .limit(1);

  if (existing[0]) return;

  const now = new Date();
  await db.insert(users).values({
    id: userId,
    email: `${userId}@dev.local`,
    name: "Dev User",
    locale: "en",
    createdAt: now,
    updatedAt: now,
  });
}
