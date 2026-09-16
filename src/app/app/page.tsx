import { getTranslations } from "next-intl/server";
import { auth } from "@/auth";
import { listDecisions } from "@/lib/ahp/decisions";
import { DecisionsPanel } from "@/components/decisions/decisions-panel";
import { toClientJson } from "@/lib/api/serialize";

export default async function AppHomePage() {
  const t = await getTranslations("decisions");
  const session = await auth();
  const userId = session!.user!.id;
  const decisions = toClientJson(await listDecisions(userId));

  return (
    <>
      <header className="page-header">
        <h1>{t("title")}</h1>
        <p>{t("subtitle")}</p>
      </header>
      <DecisionsPanel initialDecisions={decisions} />
    </>
  );
}
