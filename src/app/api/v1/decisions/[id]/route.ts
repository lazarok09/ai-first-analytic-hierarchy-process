import { requireUser } from "@/lib/auth/require-user";
import * as decisionsService from "@/lib/ahp/decisions";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { updateDecisionSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const decision = await decisionsService.getDecision(user.id, id);
    return jsonOk({ decision });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function PATCH(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const body = updateDecisionSchema.parse(await parseJsonBody(request));
    const decision = await decisionsService.updateDecision(user.id, id, body);
    return jsonOk({ decision });
  } catch (err) {
    return handleRouteError(err);
  }
}
