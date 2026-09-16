import { and, desc, eq } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import {
  alternatives,
  criteria,
  pairwiseJudgments,
  snapshots,
  type Snapshot,
} from "@/db/schema";

export type { Snapshot };
import { requireDecisionOwned } from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";

export type CreateSnapshotInput = {
  label: string;
  chatId?: string | null;
  prompt?: string | null;
  outputSummary?: string | null;
};

export type DecisionState = {
  criteria: unknown[];
  alternatives: unknown[];
  pairwiseJudgments: unknown[];
  serializedAt: string;
};

export async function serializeDecisionState(
  decisionId: string,
): Promise<DecisionState> {
  const [criteriaRows, alternativeRows, judgmentRows] = await Promise.all([
    db.select().from(criteria).where(eq(criteria.decisionId, decisionId)),
    db
      .select()
      .from(alternatives)
      .where(eq(alternatives.decisionId, decisionId)),
    db
      .select()
      .from(pairwiseJudgments)
      .where(eq(pairwiseJudgments.decisionId, decisionId)),
  ]);

  return {
    criteria: criteriaRows,
    alternatives: alternativeRows,
    pairwiseJudgments: judgmentRows,
    serializedAt: new Date().toISOString(),
  };
}

export async function createSnapshot(
  userId: string,
  decisionId: string,
  input: CreateSnapshotInput,
): Promise<Snapshot> {
  await requireDecisionOwned(userId, decisionId);
  const state = await serializeDecisionState(decisionId);
  const now = new Date();
  const row: Snapshot = {
    id: uuidv4(),
    decisionId,
    chatId: input.chatId ?? null,
    label: input.label,
    prompt: input.prompt ?? "",
    outputSummary: input.outputSummary ?? null,
    state,
    createdAt: now,
  };
  await db.insert(snapshots).values(row);
  return row;
}

export async function listSnapshots(
  userId: string,
  decisionId: string,
  opts?: { chatId?: string },
): Promise<Snapshot[]> {
  await requireDecisionOwned(userId, decisionId);
  const where = opts?.chatId
    ? and(
        eq(snapshots.decisionId, decisionId),
        eq(snapshots.chatId, opts.chatId),
      )
    : eq(snapshots.decisionId, decisionId);

  return db
    .select()
    .from(snapshots)
    .where(where)
    .orderBy(desc(snapshots.createdAt));
}

export async function getSnapshot(
  userId: string,
  snapshotId: string,
): Promise<Snapshot> {
  const rows = await db
    .select()
    .from(snapshots)
    .where(eq(snapshots.id, snapshotId))
    .limit(1);
  const row = rows[0];
  if (!row) {
    throw new AppError("NOT_FOUND", "Snapshot not found", 404);
  }
  await requireDecisionOwned(userId, row.decisionId);
  return row;
}
