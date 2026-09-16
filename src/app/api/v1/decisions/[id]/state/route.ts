import { requireUser } from "@/lib/auth/require-user";
import * as decisionsService from "@/lib/ahp/decisions";
import { handleRouteError, jsonOk } from "@/lib/ahp/http";
import * as snapshotsService from "@/lib/ahp/snapshots";

type Ctx = { params: Promise<{ id: string }> };

/** Aggregated decision state for in-app / ChatKit agents. */
export async function GET(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id: decisionId } = await ctx.params;
    const decision = await decisionsService.getDecision(user.id, decisionId);
    const state = await snapshotsService.serializeDecisionState(decisionId);
    return jsonOk({
      decision,
      criteria: state.criteria,
      alternatives: state.alternatives,
      pairwiseJudgments: state.pairwiseJudgments,
      serializedAt: state.serializedAt,
    });
  } catch (err) {
    return handleRouteError(err);
  }
}
