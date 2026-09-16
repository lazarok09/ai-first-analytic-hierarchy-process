import { and, asc, eq } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import { criteria, type Criterion } from "@/db/schema";
import { requireDecisionOwned } from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";

export type CreateCriterionInput = {
  name: string;
  description?: string | null;
  parentId?: string | null;
  sortOrder?: number;
  source?: "user" | "agent" | "import";
};

export type UpdateCriterionInput = {
  name?: string;
  description?: string | null;
  parentId?: string | null;
  sortOrder?: number;
  source?: "user" | "agent" | "import";
};

export async function listCriteria(
  userId: string,
  decisionId: string,
): Promise<Criterion[]> {
  await requireDecisionOwned(userId, decisionId);
  return db
    .select()
    .from(criteria)
    .where(eq(criteria.decisionId, decisionId))
    .orderBy(asc(criteria.sortOrder), asc(criteria.createdAt));
}

export async function createCriterion(
  userId: string,
  decisionId: string,
  input: CreateCriterionInput,
): Promise<Criterion> {
  await requireDecisionOwned(userId, decisionId);
  const now = new Date();
  const row = {
    id: uuidv4(),
    decisionId,
    parentId: input.parentId ?? null,
    name: input.name,
    description: input.description ?? null,
    sortOrder: input.sortOrder ?? 0,
    source: input.source ?? ("user" as const),
    createdAt: now,
    updatedAt: now,
  };
  await db.insert(criteria).values(row);
  return row;
}

async function getCriterionRow(criterionId: string): Promise<Criterion> {
  const rows = await db
    .select()
    .from(criteria)
    .where(eq(criteria.id, criterionId))
    .limit(1);
  const row = rows[0];
  if (!row) {
    throw new AppError("NOT_FOUND", "Criterion not found", 404);
  }
  return row;
}

export async function updateCriterion(
  userId: string,
  criterionId: string,
  input: UpdateCriterionInput,
): Promise<Criterion> {
  const existing = await getCriterionRow(criterionId);
  await requireDecisionOwned(userId, existing.decisionId);
  const now = new Date();
  await db
    .update(criteria)
    .set({
      ...(input.name !== undefined ? { name: input.name } : {}),
      ...(input.description !== undefined
        ? { description: input.description }
        : {}),
      ...(input.parentId !== undefined ? { parentId: input.parentId } : {}),
      ...(input.sortOrder !== undefined ? { sortOrder: input.sortOrder } : {}),
      ...(input.source !== undefined ? { source: input.source } : {}),
      updatedAt: now,
    })
    .where(eq(criteria.id, criterionId));
  return getCriterionRow(criterionId);
}

export async function deleteCriterion(
  userId: string,
  criterionId: string,
): Promise<void> {
  const existing = await getCriterionRow(criterionId);
  await requireDecisionOwned(userId, existing.decisionId);
  await db.delete(criteria).where(eq(criteria.id, criterionId));
}
