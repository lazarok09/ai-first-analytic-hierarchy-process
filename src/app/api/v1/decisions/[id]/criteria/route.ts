import { requireUser } from "@/lib/auth/require-user";
import * as criteriaService from "@/lib/ahp/criteria";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { createCriterionSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const items = await criteriaService.listCriteria(user.id, decisionId);
    return jsonOk({ criteria: items });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const body = createCriterionSchema.parse(await parseJsonBody(request));
    const criterion = await criteriaService.createCriterion(
      user.id,
      decisionId,
      body,
    );
    return jsonOk({ criterion }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
