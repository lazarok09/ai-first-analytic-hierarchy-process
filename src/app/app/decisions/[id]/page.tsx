import Link from "next/link";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { IconArrowLeft } from "@tabler/icons-react";
import { auth } from "@/auth";
import { getDecision } from "@/lib/ahp/decisions";
import { listCriteria } from "@/lib/ahp/criteria";
import { listPairwise } from "@/lib/ahp/pairwise";
import { listSnapshots } from "@/lib/ahp/snapshots";
import { DecisionWorkspace } from "@/components/decisions/decision-workspace";
import { AppError, isAppError } from "@/lib/ahp/errors";
import { toClientJson } from "@/lib/api/serialize";

type Ctx = { params: Promise<{ id: string }> };

export default async function DecisionDetailPage({ params }: Ctx) {
  const t = await getTranslations("decision");
  const session = await auth();
  const userId = session!.user!.id;
  const { id } = await params;

  let decision;
  try {
    decision = await getDecision(userId, id);
  } catch (err) {
    if (isAppError(err) && (err as AppError).status === 404) {
      notFound();
    }
    throw err;
  }

  const [criteria, judgments, snapshots] = await Promise.all([
    listCriteria(userId, id),
    listPairwise(userId, id),
    listSnapshots(userId, id),
  ]);

  return (
    <>
      <Link
        href="/app"
        className="row muted"
        style={{ gap: "0.35rem", marginBottom: "1rem", width: "fit-content" }}
      >
        <IconArrowLeft size={16} />
        {t("back")}
      </Link>

      <header className="page-header">
        <h1>{decision.title}</h1>
        {decision.description ? <p>{decision.description}</p> : null}
      </header>

      <DecisionWorkspace
        decisionId={decision.id}
        initialCriteria={toClientJson(criteria)}
        initialJudgments={toClientJson(judgments)}
        initialSnapshots={toClientJson(snapshots)}
      />
    </>
  );
}
