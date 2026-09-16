import { createHash, randomBytes, timingSafeEqual } from "node:crypto";
import { and, desc, eq, isNull } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import { apiKeys, type ApiKey } from "@/db/schema";
import { AppError } from "@/lib/ahp/errors";

const KEY_PREFIX = "ahp";

/** Default agent scopes (docs/sso.md). */
export const DEFAULT_API_KEY_SCOPES = [
  "ahp:read",
  "ahp:write",
  "ahp:propose",
] as const;

export type VerifiedApiKey = {
  userId: string;
  keyId: string;
  scopes: string[];
};

/** Public API key row — never includes keyHash. */
export type ApiKeyPublic = Omit<ApiKey, "keyHash">;

export type CreatedApiKey = ApiKeyPublic & {
  /** Full secret — shown once; never stored. */
  key: string;
};

function toPublic(row: ApiKey): ApiKeyPublic {
  const { keyHash: _omit, ...rest } = row;
  return rest;
}

function hashKey(rawKey: string): string {
  return createHash("sha256").update(rawKey, "utf8").digest("hex");
}

function safeEqualHex(a: string, b: string): boolean {
  try {
    const ba = Buffer.from(a, "hex");
    const bb = Buffer.from(b, "hex");
    if (ba.length !== bb.length) return false;
    return timingSafeEqual(ba, bb);
  } catch {
    return false;
  }
}

/** Parse `ahp_<prefix>_<secret>` → lookup prefix. */
export function parseApiKeyPrefix(rawKey: string): string | null {
  const parts = rawKey.trim().split("_");
  if (parts.length < 3 || parts[0] !== KEY_PREFIX) return null;
  const prefix = parts[1];
  if (!prefix || prefix.length < 4) return null;
  return prefix;
}

/**
 * Create a user-bound API key. Returns the raw secret once.
 * Format: `ahp_<prefix>_<secret>`.
 */
export async function createApiKey(
  userId: string,
  input: { name: string; scopes?: string[] },
): Promise<CreatedApiKey> {
  const name = input.name.trim();
  if (!name) {
    throw new AppError("VALIDATION_ERROR", "API key name is required", 422);
  }
  const scopes = input.scopes?.length
    ? input.scopes
    : [...DEFAULT_API_KEY_SCOPES];

  const prefix = randomBytes(4).toString("hex");
  const secret = randomBytes(24).toString("base64url");
  const rawKey = `${KEY_PREFIX}_${prefix}_${secret}`;
  const now = new Date();
  const record: ApiKey = {
    id: uuidv4(),
    userId,
    name,
    keyHash: hashKey(rawKey),
    prefix,
    scopes,
    createdAt: now,
    lastUsedAt: null,
    revokedAt: null,
  };
  await db.insert(apiKeys).values(record);
  return { ...toPublic(record), key: rawKey };
}

export async function listApiKeys(userId: string): Promise<ApiKeyPublic[]> {
  const rows = await db
    .select()
    .from(apiKeys)
    .where(eq(apiKeys.userId, userId))
    .orderBy(desc(apiKeys.createdAt));
  return rows.map(toPublic);
}

/** Verify a Bearer API key; updates lastUsedAt on success. */
export async function verifyApiKey(
  rawKey: string,
): Promise<VerifiedApiKey | null> {
  const prefix = parseApiKeyPrefix(rawKey);
  if (!prefix) return null;

  const rows = await db
    .select()
    .from(apiKeys)
    .where(and(eq(apiKeys.prefix, prefix), isNull(apiKeys.revokedAt)))
    .limit(5);

  const expected = hashKey(rawKey);
  const match = rows.find((row) => safeEqualHex(row.keyHash, expected));
  if (!match) return null;

  const now = new Date();
  await db
    .update(apiKeys)
    .set({ lastUsedAt: now })
    .where(eq(apiKeys.id, match.id));

  return {
    userId: match.userId,
    keyId: match.id,
    scopes: match.scopes ?? [],
  };
}

export async function revokeApiKey(
  userId: string,
  keyId: string,
): Promise<ApiKeyPublic> {
  const rows = await db
    .select()
    .from(apiKeys)
    .where(and(eq(apiKeys.id, keyId), eq(apiKeys.userId, userId)))
    .limit(1);
  if (!rows[0]) {
    throw new AppError("NOT_FOUND", "API key not found", 404);
  }
  if (rows[0].revokedAt) {
    return toPublic(rows[0]);
  }
  const [updated] = await db
    .update(apiKeys)
    .set({ revokedAt: new Date() })
    .where(eq(apiKeys.id, keyId))
    .returning();
  return toPublic(updated);
}
