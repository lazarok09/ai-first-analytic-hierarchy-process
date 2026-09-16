import { z } from "zod";

export const decisionStatusSchema = z.enum(["draft", "active", "archived"]);
export const criterionSourceSchema = z.enum(["user", "agent", "import"]);
export const pairwiseLevelSchema = z.enum(["criteria", "alternatives"]);
export const pairwiseStatusSchema = z.enum(["proposal", "committed"]);

/** Saaty 1–9 or reciprocal (≈1/9 … 9). */
export const saatyValueSchema = z
  .number()
  .refine((v) => v >= 1 / 9 && v <= 9, {
    message: "Saaty value must be between 1/9 and 9",
  });

export const createDecisionSchema = z.object({
  title: z.string().min(1).max(500),
  description: z.string().max(5000).optional().nullable(),
  status: decisionStatusSchema.optional(),
});

export const updateDecisionSchema = z
  .object({
    title: z.string().min(1).max(500).optional(),
    description: z.string().max(5000).optional().nullable(),
    status: decisionStatusSchema.optional(),
  })
  .refine((v) => Object.keys(v).length > 0, {
    message: "At least one field is required",
  });

export const createCriterionSchema = z.object({
  name: z.string().min(1).max(300),
  description: z.string().max(5000).optional().nullable(),
  parentId: z.string().uuid().optional().nullable(),
  sortOrder: z.number().int().optional(),
  source: criterionSourceSchema.optional(),
});

export const updateCriterionSchema = z
  .object({
    name: z.string().min(1).max(300).optional(),
    description: z.string().max(5000).optional().nullable(),
    parentId: z.string().uuid().optional().nullable(),
    sortOrder: z.number().int().optional(),
    source: criterionSourceSchema.optional(),
  })
  .refine((v) => Object.keys(v).length > 0, {
    message: "At least one field is required",
  });

export const upsertPairwiseSchema = z.object({
  judgments: z
    .array(
      z.object({
        parentCriterionId: z.string().uuid().nullable().optional(),
        level: pairwiseLevelSchema,
        leftId: z.string().uuid(),
        rightId: z.string().uuid(),
        value: saatyValueSchema,
        /** Agent writes should use `proposal` (default). */
        status: pairwiseStatusSchema.default("proposal"),
      }),
    )
    .min(1),
});

export const createSnapshotSchema = z.object({
  label: z.string().min(1).max(300),
  chatId: z.string().uuid().optional().nullable(),
  prompt: z.string().optional().nullable(),
  outputSummary: z.string().optional().nullable(),
});

export const createChatSchema = z.object({
  title: z.string().min(1).max(300),
  decisionId: z.string().uuid().optional().nullable(),
  externalSessionId: z.string().max(500).optional().nullable(),
});

export const ensureChatSchema = z.object({
  externalSessionId: z.string().min(1).max(500),
  title: z.string().min(1).max(300).optional(),
  decisionId: z.string().uuid().optional().nullable(),
});

export const updateChatSchema = z
  .object({
    title: z.string().min(1).max(300).optional(),
    decisionId: z.string().uuid().optional().nullable(),
  })
  .refine((v) => Object.keys(v).length > 0, {
    message: "At least one field is required",
  });

export const attachChatMessageSchema = z.object({
  prompt: z.string().min(1),
  outputSummary: z.string().min(1),
  label: z.string().min(1).max(300).optional(),
});

export const createAlternativeSchema = z.object({
  name: z.string().min(1).max(300),
  description: z.string().max(5000).optional().nullable(),
  sortOrder: z.number().int().optional(),
  source: criterionSourceSchema.optional(),
});

export const createApiKeySchema = z.object({
  name: z.string().min(1).max(200),
  scopes: z.array(z.string().min(1).max(100)).max(20).optional(),
});
