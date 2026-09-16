import { requireUser } from "@/lib/auth/require-user";
import * as criteriaService from "@/lib/ahp/criteria";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { updateCriterionSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function PATCH(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const body = updateCriterionSchema.parse(await parseJsonBody(request));
    const criterion = await criteriaService.updateCriterion(user.id, id, body);
    return jsonOk({ criterion });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function DELETE(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    await criteriaService.deleteCriterion(user.id, id);
    return jsonOk({ ok: true });
  } catch (err) {
    return handleRouteError(err);
  }
}
