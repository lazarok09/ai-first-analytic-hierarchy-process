import { requireUser } from "@/lib/auth/require-user";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import * as snapshotsService from "@/lib/ahp/snapshots";
import { createSnapshotSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const url = new URL(request.url);
    const chatId = url.searchParams.get("chatId") ?? undefined;
    const items = await snapshotsService.listSnapshots(user.id, decisionId, {
      chatId,
    });
    return jsonOk({ snapshots: items });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const body = createSnapshotSchema.parse(await parseJsonBody(request));
    const snapshot = await snapshotsService.createSnapshot(
      user.id,
      decisionId,
      body,
    );
    return jsonOk({ snapshot }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
