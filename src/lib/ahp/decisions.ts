import { and, desc, eq } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import { decisions, type Decision } from "@/db/schema";
import { AppError } from "@/lib/ahp/errors";

export type CreateDecisionInput = {
  title: string;
  description?: string | null;
  status?: "draft" | "active" | "archived";
};

export type UpdateDecisionInput = {
  title?: string;
  description?: string | null;
  status?: "draft" | "active" | "archived";
};

export async function listDecisions(userId: string): Promise<Decision[]> {
  return db
    .select()
    .from(decisions)
    .where(eq(decisions.userId, userId))
    .orderBy(desc(decisions.updatedAt));
}

export async function createDecision(
  userId: string,
  input: CreateDecisionInput,
): Promise<Decision> {
  const now = new Date();
  const id = uuidv4();
  const row = {
    id,
    userId,
    title: input.title,
    description: input.description ?? null,
    status: input.status ?? ("draft" as const),
    createdAt: now,
    updatedAt: now,
  };
  await db.insert(decisions).values(row);
  return row;
}

export async function getDecision(
  userId: string,
  decisionId: string,
): Promise<Decision> {
  const rows = await db
    .select()
    .from(decisions)
    .where(and(eq(decisions.id, decisionId), eq(decisions.userId, userId)))
    .limit(1);
  const row = rows[0];
  if (!row) {
    throw new AppError("NOT_FOUND", "Decision not found", 404);
  }
  return row;
}

export async function updateDecision(
  userId: string,
  decisionId: string,
  input: UpdateDecisionInput,
): Promise<Decision> {
  await getDecision(userId, decisionId);
  const now = new Date();
  await db
    .update(decisions)
    .set({
      ...(input.title !== undefined ? { title: input.title } : {}),
      ...(input.description !== undefined
        ? { description: input.description }
        : {}),
      ...(input.status !== undefined ? { status: input.status } : {}),
      updatedAt: now,
    })
    .where(and(eq(decisions.id, decisionId), eq(decisions.userId, userId)));
  return getDecision(userId, decisionId);
}

/** Ownership check for nested resources. */
export async function requireDecisionOwned(
  userId: string,
  decisionId: string,
): Promise<Decision> {
  return getDecision(userId, decisionId);
}
