import { requireUser } from "@/lib/auth/require-user";
import { createApiKey, listApiKeys } from "@/lib/auth/api-keys";
import {
  handleRouteError,
  jsonOk,
  parseJsonBody,
} from "@/lib/ahp/http";
import { createApiKeySchema } from "@/lib/ahp/validators";

export async function GET(request: Request) {
  try {
    const user = await requireUser(request);
    const keys = await listApiKeys(user.id);
    return jsonOk({ apiKeys: keys });
  } catch (err) {
    return handleRouteError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await requireUser(request);
    const body = createApiKeySchema.parse(await parseJsonBody(request));
    const apiKey = await createApiKey(user.id, body);
    return jsonOk({ apiKey }, 201);
  } catch (err) {
    return handleRouteError(err);
  }
}
