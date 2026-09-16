import { requireUser } from "@/lib/auth/require-user";
import * as decisionsService from "@/lib/ahp/decisions";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { createDecisionSchema } from "@/lib/ahp/validators";

export async function GET(request: Request) {
  try {
    const user = await requireUser(request);
    const items = await decisionsService.listDecisions(user.id);
    return jsonOk({ decisions: items });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await requireUser(request);
    const body = createDecisionSchema.parse(await parseJsonBody(request));
    const decision = await decisionsService.createDecision(user.id, body);
    return jsonOk({ decision }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
