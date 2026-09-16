import { requireUser } from "@/lib/auth/require-user";
import * as alternativesService from "@/lib/ahp/alternatives";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { createAlternativeSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const items = await alternativesService.listAlternatives(user.id, decisionId);
    return jsonOk({ alternatives: items });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const body = createAlternativeSchema.parse(await parseJsonBody(request));
    const alternative = await alternativesService.createAlternative(
      user.id,
      decisionId,
      body,
    );
    return jsonOk({ alternative }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
