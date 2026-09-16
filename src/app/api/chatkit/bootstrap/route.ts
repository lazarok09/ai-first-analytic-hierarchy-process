import { auth } from "@/auth";
import { handleRouteError, jsonOk } from "@/lib/ahp/http";
import { AppError } from "@/lib/ahp/errors";
import { mintChatKitAgentToken } from "@/lib/auth/chatkit-token";

/**
 * Mint a ChatKit agent credential bound to the signed-in Auth.js user.
 * The floating widget uses the token against `/api/chatkit` (proxied) and REST.
 */
export async function POST() {
  try {
    const session = await auth();
    const userId = session?.user?.id;
    if (!userId) {
      throw new AppError("UNAUTHORIZED", "Sign in required", 401);
    }

    const { token, expiresAt } = mintChatKitAgentToken(userId);
    const chatkitUrl =
      process.env.CHATKIT_PUBLIC_URL?.trim() || "/api/chatkit";
    const domainKey =
      process.env.CHATKIT_DOMAIN_KEY?.trim() || "domain_pk_localhost";

    return jsonOk({
      userId,
      agentToken: token,
      expiresAt,
      chatkitUrl,
      domainKey,
    });
  } catch (err) {
    return handleRouteError(err);
  }
}
