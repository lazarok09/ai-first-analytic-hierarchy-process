import { and, asc, eq } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import { alternatives, type Alternative } from "@/db/schema";
import { requireDecisionOwned } from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";

export type CreateAlternativeInput = {
  name: string;
  description?: string | null;
  sortOrder?: number;
  source?: "user" | "agent" | "import";
};

export type UpdateAlternativeInput = {
  name?: string;
  description?: string | null;
  sortOrder?: number;
  source?: "user" | "agent" | "import";
};

export async function listAlternatives(
  userId: string,
  decisionId: string,
): Promise<Alternative[]> {
  await requireDecisionOwned(userId, decisionId);
  return db
    .select()
    .from(alternatives)
    .where(eq(alternatives.decisionId, decisionId))
    .orderBy(asc(alternatives.sortOrder), asc(alternatives.createdAt));
}

export async function createAlternative(
  userId: string,
  decisionId: string,
  input: CreateAlternativeInput,
): Promise<Alternative> {
  await requireDecisionOwned(userId, decisionId);
  const now = new Date();
  const row = {
    id: uuidv4(),
    decisionId,
    name: input.name,
    description: input.description ?? null,
    sortOrder: input.sortOrder ?? 0,
    source: input.source ?? ("agent" as const),
    createdAt: now,
    updatedAt: now,
  };
  await db.insert(alternatives).values(row);
  return row;
}

async function getAlternativeRow(alternativeId: string): Promise<Alternative> {
  const rows = await db
    .select()
    .from(alternatives)
    .where(eq(alternatives.id, alternativeId))
    .limit(1);
  const row = rows[0];
  if (!row) {
    throw new AppError("NOT_FOUND", "Alternative not found", 404);
  }
  return row;
}

export async function updateAlternative(
  userId: string,
  alternativeId: string,
  input: UpdateAlternativeInput,
): Promise<Alternative> {
  const existing = await getAlternativeRow(alternativeId);
  await requireDecisionOwned(userId, existing.decisionId);
  const now = new Date();
  await db
    .update(alternatives)
    .set({
      ...(input.name !== undefined ? { name: input.name } : {}),
      ...(input.description !== undefined
        ? { description: input.description }
        : {}),
      ...(input.sortOrder !== undefined ? { sortOrder: input.sortOrder } : {}),
      ...(input.source !== undefined ? { source: input.source } : {}),
      updatedAt: now,
    })
    .where(eq(alternatives.id, alternativeId));
  return getAlternativeRow(alternativeId);
}

export async function deleteAlternative(
  userId: string,
  alternativeId: string,
): Promise<void> {
  const existing = await getAlternativeRow(alternativeId);
  await requireDecisionOwned(userId, existing.decisionId);
  await db
    .delete(alternatives)
    .where(and(eq(alternatives.id, alternativeId)));
}
