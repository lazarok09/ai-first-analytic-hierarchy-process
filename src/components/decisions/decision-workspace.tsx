"use client";

import { useState } from "react";
import type { Criterion, PairwiseJudgment, Snapshot } from "@/lib/api/types";
import { CriteriaEditor } from "@/components/criteria/criteria-editor";
import { PairwisePanel } from "@/components/pairwise/pairwise-panel";
import { SnapshotsPanel } from "@/components/snapshots/snapshots-panel";

type Props = {
  decisionId: string;
  initialCriteria: Criterion[];
  initialJudgments: PairwiseJudgment[];
  initialSnapshots: Snapshot[];
};

export function DecisionWorkspace({
  decisionId,
  initialCriteria,
  initialJudgments,
  initialSnapshots,
}: Props) {
  const [criteria, setCriteria] = useState(initialCriteria);

  return (
    <>
      <CriteriaEditor
        decisionId={decisionId}
        initialCriteria={initialCriteria}
        onCriteriaChange={setCriteria}
      />
      <PairwisePanel
        decisionId={decisionId}
        criteria={criteria}
        initialJudgments={initialJudgments}
      />
      <SnapshotsPanel
        decisionId={decisionId}
        initialSnapshots={initialSnapshots}
      />
    </>
  );
}
