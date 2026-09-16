import { headers } from "next/headers";
import { getTranslations } from "next-intl/server";
import { auth } from "@/auth";
import { listApiKeys } from "@/lib/auth/api-keys";
import { ConnectAgentPanel } from "@/components/connect/connect-agent-panel";
import { toClientJson } from "@/lib/api/serialize";

export default async function ConnectPage() {
  const t = await getTranslations("connect");
  const session = await auth();
  const keys = toClientJson(await listApiKeys(session!.user!.id));
  const h = await headers();
  const host = h.get("x-forwarded-host") ?? h.get("host") ?? "localhost:3000";
  const proto = h.get("x-forwarded-proto") ?? "http";
  const mcpEndpoint = `${proto}://${host}/api/mcp`;

  return (
    <>
      <header className="page-header">
        <h1>{t("title")}</h1>
        <p>{t("subtitle")}</p>
      </header>
      <ConnectAgentPanel initialKeys={keys} mcpEndpoint={mcpEndpoint} />
    </>
  );
}
