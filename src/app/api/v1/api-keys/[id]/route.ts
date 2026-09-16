import { requireUser } from "@/lib/auth/require-user";
import { revokeApiKey } from "@/lib/auth/api-keys";
import { handleRouteError, jsonOk } from "@/lib/ahp/http";

type Ctx = { params: Promise<{ id: string }> };

export async function DELETE(request: Request, ctx: Ctx) {
  try {
    const user = await requireUser(request);
    const { id } = await ctx.params;
    const apiKey = await revokeApiKey(user.id, id);
    return jsonOk({ apiKey });
  } catch (err) {
    return handleRouteError(err);
  }
}
