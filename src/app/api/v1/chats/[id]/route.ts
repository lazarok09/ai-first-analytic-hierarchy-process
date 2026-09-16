import { requireUser } from "@/lib/auth/require-user";
import * as chatsService from "@/lib/ahp/chats";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import {
  attachChatMessageSchema,
  updateChatSchema,
} from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const chat = await chatsService.getChat(user.id, id);
    return jsonOk({ chat });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function PATCH(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const body = updateChatSchema.parse(await parseJsonBody(request));
    const chat = await chatsService.updateChat(user.id, id, body);
    return jsonOk({ chat });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const body = attachChatMessageSchema.parse(await parseJsonBody(request));
    const summary = await chatsService.attachMessageSummary(user.id, id, body);
    return jsonOk({ summary }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
