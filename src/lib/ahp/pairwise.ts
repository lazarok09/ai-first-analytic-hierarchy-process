import { and, asc, eq, isNull } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import {
  pairwiseJudgments,
  type PairwiseJudgment,
} from "@/db/schema";
import { requireDecisionOwned } from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";

export type UpsertJudgmentInput = {
  parentCriterionId?: string | null;
  level: "criteria" | "alternatives";
  leftId: string;
  rightId: string;
  value: number;
  /** Agent proposals must use `proposal`. */
  status?: "proposal" | "committed";
};

export async function listPairwise(
  userId: string,
  decisionId: string,
): Promise<PairwiseJudgment[]> {
  await requireDecisionOwned(userId, decisionId);
  return db
    .select()
    .from(pairwiseJudgments)
    .where(eq(pairwiseJudgments.decisionId, decisionId))
    .orderBy(asc(pairwiseJudgments.createdAt));
}

export async function upsertPairwise(
  userId: string,
  decisionId: string,
  judgments: UpsertJudgmentInput[],
): Promise<PairwiseJudgment[]> {
  await requireDecisionOwned(userId, decisionId);

  const results: PairwiseJudgment[] = [];
  for (const j of judgments) {
    if (j.leftId === j.rightId) {
      throw new AppError(
        "VALIDATION_ERROR",
        "leftId and rightId must differ",
        422,
      );
    }
    results.push(await upsertOne(decisionId, j));
  }
  return results;
}

async function upsertOne(
  decisionId: string,
  input: UpsertJudgmentInput,
): Promise<PairwiseJudgment> {
  const parentCriterionId = input.parentCriterionId ?? null;
  const status = input.status ?? "proposal";
  const now = new Date();

  const parentClause =
    parentCriterionId === null
      ? isNull(pairwiseJudgments.parentCriterionId)
      : eq(pairwiseJudgments.parentCriterionId, parentCriterionId);

  const existing = await db
    .select()
    .from(pairwiseJudgments)
    .where(
      and(
        eq(pairwiseJudgments.decisionId, decisionId),
        parentClause,
        eq(pairwiseJudgments.level, input.level),
        eq(pairwiseJudgments.leftId, input.leftId),
        eq(pairwiseJudgments.rightId, input.rightId),
      ),
    )
    .limit(1);

  if (existing[0]) {
    await db
      .update(pairwiseJudgments)
      .set({
        value: input.value,
        status,
        updatedAt: now,
      })
      .where(eq(pairwiseJudgments.id, existing[0].id));
    return {
      ...existing[0],
      value: input.value,
      status,
      updatedAt: now,
    };
  }

  const row: PairwiseJudgment = {
    id: uuidv4(),
    decisionId,
    parentCriterionId,
    level: input.level,
    leftId: input.leftId,
    rightId: input.rightId,
    value: input.value,
    status,
    createdAt: now,
    updatedAt: now,
  };
  await db.insert(pairwiseJudgments).values(row);
  return row;
}
