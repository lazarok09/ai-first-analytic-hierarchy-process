"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { IconCheck, IconArrowsExchange } from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";
import {
  SAATY_INTENSITIES,
  criterionPairs,
  decodeSaaty,
  encodeSaaty,
  findJudgmentForPair,
} from "@/lib/ahp/saaty";
import type { Criterion, PairwiseJudgment } from "@/lib/api/types";

type Props = {
  decisionId: string;
  criteria: Criterion[];
  initialJudgments: PairwiseJudgment[];
};

export function PairwisePanel({
  decisionId,
  criteria,
  initialJudgments,
}: Props) {
  const t = useTranslations("pairwise");
  const tc = useTranslations("common");
  const [judgments, setJudgments] = useState(initialJudgments);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const rootCriteria = useMemo(
    () => criteria.filter((c) => !c.parentId),
    [criteria],
  );
  const byId = useMemo(
    () => new Map(rootCriteria.map((c) => [c.id, c])),
    [rootCriteria],
  );
  const pairs = useMemo(
    () => criterionPairs(rootCriteria.map((c) => c.id)),
    [rootCriteria],
  );

  const rootJudgments = useMemo(
    () =>
      judgments.filter(
        (j) =>
          j.level === "criteria" &&
          (j.parentCriterionId === null || j.parentCriterionId === undefined),
      ),
    [judgments],
  );

  const proposalCount = rootJudgments.filter(
    (j) => j.status === "proposal",
  ).length;

  if (rootCriteria.length < 2) {
    return (
      <section className="section" aria-labelledby="pairwise-heading">
        <h2 id="pairwise-heading">{t("title")}</h2>
        <p className="callout" style={{ marginTop: "0.75rem" }}>
          {t("locked")}
        </p>
      </section>
    );
  }

  async function upsert(
    leftId: string,
    rightId: string,
    value: number,
    status: "proposal" | "committed",
  ) {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ judgments: PairwiseJudgment[] }>(
        `/api/v1/decisions/${decisionId}/pairwise`,
        {
          method: "PUT",
          body: JSON.stringify({
            judgments: [
              {
                parentCriterionId: null,
                level: "criteria",
                leftId,
                rightId,
                value,
                status,
              },
            ],
          }),
        },
      );
      setJudgments((prev) => mergeJudgments(prev, res.judgments));
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  async function commitAllProposals() {
    const proposals = rootJudgments.filter((j) => j.status === "proposal");
    if (proposals.length === 0) return;
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ judgments: PairwiseJudgment[] }>(
        `/api/v1/decisions/${decisionId}/pairwise`,
        {
          method: "PUT",
          body: JSON.stringify({
            judgments: proposals.map((j) => ({
              parentCriterionId: j.parentCriterionId,
              level: j.level,
              leftId: j.leftId,
              rightId: j.rightId,
              value: j.value,
              status: "committed" as const,
            })),
          }),
        },
      );
      setJudgments((prev) => mergeJudgments(prev, res.judgments));
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="section" aria-labelledby="pairwise-heading">
      <h2 id="pairwise-heading">{t("title")}</h2>
      <p className="section-lead">{t("subtitle")}</p>

      {proposalCount > 0 ? (
        <div className="row" style={{ marginBottom: "1rem" }}>
          <p className="callout callout-accent" style={{ margin: 0, flex: 1 }}>
            {t("proposalsPending", { count: proposalCount })}
          </p>
          <button
            type="button"
            className="btn btn-primary"
            disabled={busy}
            onClick={() => void commitAllProposals()}
          >
            <IconCheck size={18} />
            {t("commitAll")}
          </button>
        </div>
      ) : null}

      {error ? <p className="error-text">{error}</p> : null}

      {pairs.length === 0 ? (
        <p className="muted">{t("noPairs")}</p>
      ) : (
        <div>
          {pairs.map((pair) => {
            const left = byId.get(pair.leftId);
            const right = byId.get(pair.rightId);
            if (!left || !right) return null;

            const found = findJudgmentForPair(
              rootJudgments,
              pair.leftId,
              pair.rightId,
            );
            const decoded = found
              ? decodeSaaty(found.orientedValue)
              : { preferLeft: true, intensity: 1 };
            const status = found?.judgment.status;

            return (
              <div key={`${pair.leftId}-${pair.rightId}`} className="pair-row">
                <div>
                  <div className="row" style={{ gap: "0.5rem", marginBottom: "0.25rem" }}>
                    <strong>
                      {t("leftVsRight", { left: left.name, right: right.name })}
                    </strong>
                    <StatusBadge status={status} />
                  </div>
                  <p className="faint" style={{ margin: 0 }}>
                    {t("howMuchMore")}
                  </p>
                </div>

                <div className="stack" style={{ gap: "0.5rem" }}>
                  <select
                    className="select"
                    value={found ? String(decoded.intensity) : ""}
                    disabled={busy}
                    aria-label={t("howMuchMore")}
                    onChange={(e) => {
                      const intensity = Number(e.target.value);
                      if (!intensity) return;
                      const preferLeft = found
                        ? decoded.preferLeft
                        : true;
                      const value = encodeSaaty(preferLeft, intensity);
                      void upsert(pair.leftId, pair.rightId, value, "committed");
                    }}
                  >
                    {!found ? (
                      <option value="" disabled>
                        {t("statusUnset")}
                      </option>
                    ) : null}
                    {SAATY_INTENSITIES.map((n) => (
                      <option key={n} value={n}>
                        {t(`scale.${n}`)}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    className="btn btn-secondary"
                    disabled={busy}
                    onClick={() => {
                      const value = encodeSaaty(
                        !decoded.preferLeft,
                        decoded.intensity,
                      );
                      void upsert(pair.leftId, pair.rightId, value, "committed");
                    }}
                  >
                    <IconArrowsExchange size={16} />
                    {t("invert")}
                    {decoded.preferLeft
                      ? ` → ${right.name}`
                      : ` → ${left.name}`}
                  </button>
                </div>

                <div className="row">
                  {status === "proposal" && found ? (
                    <button
                      type="button"
                      className="btn btn-primary"
                      disabled={busy}
                      onClick={() =>
                        void upsert(
                          found.judgment.leftId,
                          found.judgment.rightId,
                          found.judgment.value,
                          "committed",
                        )
                      }
                    >
                      <IconCheck size={16} />
                      {t("commitOne")}
                    </button>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

function StatusBadge({
  status,
}: {
  status: "proposal" | "committed" | undefined;
}) {
  const t = useTranslations("pairwise");
  if (!status) {
    return <span className="badge badge-unset">{t("statusUnset")}</span>;
  }
  if (status === "proposal") {
    return <span className="badge badge-proposal">{t("statusProposal")}</span>;
  }
  return <span className="badge badge-committed">{t("statusCommitted")}</span>;
}

function mergeJudgments(
  prev: PairwiseJudgment[],
  updated: PairwiseJudgment[],
): PairwiseJudgment[] {
  const next = [...prev];
  for (const u of updated) {
    const idx = next.findIndex((j) => j.id === u.id);
    if (idx >= 0) {
      next[idx] = u;
      continue;
    }
    const samePair = next.findIndex(
      (j) =>
        j.decisionId === u.decisionId &&
        j.level === u.level &&
        (j.parentCriterionId ?? null) === (u.parentCriterionId ?? null) &&
        j.leftId === u.leftId &&
        j.rightId === u.rightId,
    );
    if (samePair >= 0) {
      next[samePair] = u;
    } else {
      next.push(u);
    }
  }
  return next;
}
