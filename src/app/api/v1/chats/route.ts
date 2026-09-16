import { requireUser } from "@/lib/auth/require-user";
import * as chatsService from "@/lib/ahp/chats";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { createChatSchema } from "@/lib/ahp/validators";

export async function GET(request: Request) {
  try {
    const user = await requireUser(request);
    const items = await chatsService.listChats(user.id);
    return jsonOk({ chats: items });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await requireUser(request);
    const body = createChatSchema.parse(await parseJsonBody(request));
    const chat = await chatsService.createChat(user.id, body);
    return jsonOk({ chat }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
