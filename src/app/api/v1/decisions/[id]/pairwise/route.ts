import { requireUser } from "@/lib/auth/require-user";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import * as pairwiseService from "@/lib/ahp/pairwise";
import { upsertPairwiseSchema } from "@/lib/ahp/validators";

type Ctx = { params: Promise<{ id: string }> };

export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const judgments = await pairwiseService.listPairwise(user.id, decisionId);
    return jsonOk({ judgments });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function PUT(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const body = upsertPairwiseSchema.parse(await parseJsonBody(request));
    const judgments = await pairwiseService.upsertPairwise(
      user.id,
      decisionId,
      body.judgments,
    );
    return jsonOk({ judgments });
  } catch (err) {
    return handleRouteError(err);
  }
}
