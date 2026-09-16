import { and, desc, eq } from "drizzle-orm";
import { v4 as uuidv4 } from "uuid";
import { db } from "@/db";
import { chats, type Chat } from "@/db/schema";
import { requireDecisionOwned } from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";
import { createSnapshot, type Snapshot } from "@/lib/ahp/snapshots";

export type CreateChatInput = {
  title: string;
  decisionId?: string | null;
  externalSessionId?: string | null;
};

export type AttachMessageInput = {
  prompt: string;
  outputSummary: string;
  /** Snapshot label when a decision is linked; defaults to a timestamped label. */
  label?: string;
};

export type PromptSummary = {
  chatId: string;
  prompt: string;
  outputSummary: string;
  snapshot?: Snapshot;
};

export async function listChats(userId: string): Promise<Chat[]> {
  return db
    .select()
    .from(chats)
    .where(eq(chats.userId, userId))
    .orderBy(desc(chats.updatedAt));
}

export async function createChat(
  userId: string,
  input: CreateChatInput,
): Promise<Chat> {
  if (input.decisionId) {
    await requireDecisionOwned(userId, input.decisionId);
  }
  const now = new Date();
  const row: Chat = {
    id: uuidv4(),
    userId,
    title: input.title,
    decisionId: input.decisionId ?? null,
    externalSessionId: input.externalSessionId ?? null,
    createdAt: now,
    updatedAt: now,
  };
  await db.insert(chats).values(row);
  return row;
}

export async function getChat(userId: string, chatId: string): Promise<Chat> {
  const rows = await db
    .select()
    .from(chats)
    .where(and(eq(chats.id, chatId), eq(chats.userId, userId)))
    .limit(1);
  const row = rows[0];
  if (!row) {
    throw new AppError("NOT_FOUND", "Chat not found", 404);
  }
  return row;
}

export async function findChatByExternalSessionId(
  userId: string,
  externalSessionId: string,
): Promise<Chat | null> {
  const rows = await db
    .select()
    .from(chats)
    .where(
      and(
        eq(chats.userId, userId),
        eq(chats.externalSessionId, externalSessionId),
      ),
    )
    .limit(1);
  return rows[0] ?? null;
}

export type EnsureChatInput = {
  externalSessionId: string;
  title?: string;
  decisionId?: string | null;
};

/**
 * Idempotent ChatKit thread → local chat mapping.
 * Creates a row when missing; optionally links/updates decisionId + title.
 */
export async function ensureChatByExternalSession(
  userId: string,
  input: EnsureChatInput,
): Promise<{ chat: Chat; created: boolean }> {
  const existing = await findChatByExternalSessionId(
    userId,
    input.externalSessionId,
  );
  if (existing) {
    const nextTitle = input.title?.trim();
    const nextDecisionId =
      input.decisionId === undefined ? undefined : input.decisionId;
    if (nextDecisionId) {
      await requireDecisionOwned(userId, nextDecisionId);
    }
    const needsUpdate =
      (nextTitle && nextTitle !== existing.title) ||
      (nextDecisionId !== undefined &&
        nextDecisionId !== existing.decisionId);
    if (!needsUpdate) {
      return { chat: existing, created: false };
    }
    const now = new Date();
    await db
      .update(chats)
      .set({
        ...(nextTitle ? { title: nextTitle } : {}),
        ...(nextDecisionId !== undefined
          ? { decisionId: nextDecisionId }
          : {}),
        updatedAt: now,
      })
      .where(eq(chats.id, existing.id));
    return { chat: await getChat(userId, existing.id), created: false };
  }

  const chat = await createChat(userId, {
    title: input.title?.trim() || "ChatKit session",
    decisionId: input.decisionId ?? null,
    externalSessionId: input.externalSessionId,
  });
  return { chat, created: true };
}

export type UpdateChatInput = {
  title?: string;
  decisionId?: string | null;
};

export async function updateChat(
  userId: string,
  chatId: string,
  input: UpdateChatInput,
): Promise<Chat> {
  await getChat(userId, chatId);
  if (input.decisionId) {
    await requireDecisionOwned(userId, input.decisionId);
  }
  const now = new Date();
  await db
    .update(chats)
    .set({
      ...(input.title !== undefined ? { title: input.title } : {}),
      ...(input.decisionId !== undefined
        ? { decisionId: input.decisionId }
        : {}),
      updatedAt: now,
    })
    .where(eq(chats.id, chatId));
  return getChat(userId, chatId);
}

/**
 * Record a prompt + AI summary for a chat.
 * When the chat is linked to a decision, also creates a snapshot (prompt lineage).
 */
export async function attachMessageSummary(
  userId: string,
  chatId: string,
  input: AttachMessageInput,
): Promise<PromptSummary> {
  const chat = await getChat(userId, chatId);
  const now = new Date();
  await db
    .update(chats)
    .set({ updatedAt: now })
    .where(eq(chats.id, chatId));

  let snapshot: Snapshot | undefined;
  if (chat.decisionId) {
    snapshot = await createSnapshot(userId, chat.decisionId, {
      label: input.label ?? `chat-${now.toISOString()}`,
      chatId: chat.id,
      prompt: input.prompt,
      outputSummary: input.outputSummary,
    });
  }

  return {
    chatId: chat.id,
    prompt: input.prompt,
    outputSummary: input.outputSummary,
    snapshot,
  };
}

/** Convenience helper for MCP / agents: summarize without requiring a prior chat update. */
export function summarizePromptOutput(
  prompt: string,
  outputSummary: string,
): { prompt: string; outputSummary: string } {
  return {
    prompt: prompt.trim(),
    outputSummary: outputSummary.trim(),
  };
}
