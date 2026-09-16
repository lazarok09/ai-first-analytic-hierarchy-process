import { createHmac, timingSafeEqual } from "node:crypto";

const TOKEN_PREFIX = "ck1";
const DEFAULT_TTL_SEC = 60 * 60 * 24; // 24h

type TokenPayload = {
  sub: string;
  exp: number;
};

function secret(): string {
  const value = process.env.AUTH_SECRET?.trim();
  if (!value) {
    throw new Error("AUTH_SECRET is required to mint ChatKit agent tokens");
  }
  return value;
}

function b64url(input: string | Buffer): string {
  return Buffer.from(input)
    .toString("base64url")
    .replace(/=+$/, "");
}

function sign(payloadB64: string): string {
  return createHmac("sha256", secret()).update(payloadB64).digest("base64url");
}

function safeEqual(a: string, b: string): boolean {
  const ba = Buffer.from(a);
  const bb = Buffer.from(b);
  if (ba.length !== bb.length) return false;
  return timingSafeEqual(ba, bb);
}

/** Mint a short-lived bearer token the ChatKit agent uses against REST. */
export function mintChatKitAgentToken(
  userId: string,
  ttlSec = DEFAULT_TTL_SEC,
): { token: string; expiresAt: number } {
  const exp = Math.floor(Date.now() / 1000) + ttlSec;
  const payload: TokenPayload = { sub: userId, exp };
  const payloadB64 = b64url(JSON.stringify(payload));
  const sig = sign(payloadB64);
  return {
    token: `${TOKEN_PREFIX}.${payloadB64}.${sig}`,
    expiresAt: exp,
  };
}

/** Verify a `ck1.…` agent token; returns user id or null. */
export function verifyChatKitAgentToken(token: string): { userId: string } | null {
  const parts = token.trim().split(".");
  if (parts.length !== 3 || parts[0] !== TOKEN_PREFIX) return null;
  const [, payloadB64, sig] = parts;
  if (!payloadB64 || !sig) return null;
  if (!safeEqual(sign(payloadB64), sig)) return null;

  try {
    const json = Buffer.from(payloadB64, "base64url").toString("utf8");
    const payload = JSON.parse(json) as TokenPayload;
    if (!payload?.sub || typeof payload.exp !== "number") return null;
    if (payload.exp < Math.floor(Date.now() / 1000)) return null;
    return { userId: payload.sub };
  } catch {
    return null;
  }
}

export function isChatKitAgentToken(token: string): boolean {
  return token.trim().startsWith(`${TOKEN_PREFIX}.`);
}
