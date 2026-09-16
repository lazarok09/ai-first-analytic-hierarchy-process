import { requireUser } from "@/lib/auth/require-user";
import { handleRouteError, jsonOk } from "@/lib/ahp/http";
import * as snapshotsService from "@/lib/ahp/snapshots";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const snapshot = await snapshotsService.getSnapshot(user.id, id);
    return jsonOk({ snapshot });
  } catch (err) {
    return handleRouteError(err);
  }
}
