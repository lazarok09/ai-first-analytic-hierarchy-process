import { requireUser } from "@/lib/auth/require-user";
import * as chatsService from "@/lib/ahp/chats";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { ensureChatSchema } from "@/lib/ahp/validators";

/** Idempotent ChatKit thread → chats.externalSessionId mapping. */
export async function POST(request: Request) {
  try {
    const user = await requireUser(request);
    const body = ensureChatSchema.parse(await parseJsonBody(request));
    const result = await chatsService.ensureChatByExternalSession(user.id, body);
    return jsonOk(result, result.created ? 201 : 200);
  } catch (err) {
    return handleRouteError(err);
  }
}
