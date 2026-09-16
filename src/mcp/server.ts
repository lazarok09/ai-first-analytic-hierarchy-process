import { z } from "zod";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import * as criteriaService from "@/lib/ahp/criteria";
import * as decisionsService from "@/lib/ahp/decisions";
import { AppError } from "@/lib/ahp/errors";
import * as pairwiseService from "@/lib/ahp/pairwise";
import * as snapshotsService from "@/lib/ahp/snapshots";
import {
  createCriterionSchema,
  createDecisionSchema,
  pairwiseLevelSchema,
  saatyValueSchema,
  updateCriterionSchema,
} from "@/lib/ahp/validators";
import { toolErr, toolOk } from "@/mcp/result";

/**
 * Stateless MCP server bound to a single authenticated user.
 * Tools call domain services directly (no HTTP loopback).
 */
export function createAhpMcpServer(userId: string): McpServer {
  const server = new McpServer(
    {
      name: "ai-first-ahp",
      version: "0.1.0",
    },
    {
      instructions:
        "AHP decision modeling tools. Pairwise writes are always proposals — never silently commit subjective judgments. Prefer get_decision_state before mutating.",
    },
  );

  server.registerTool(
    "create_decision",
    {
      title: "Create decision",
      description: "Create a new AHP decision owned by the authenticated user.",
      inputSchema: {
        title: z.string().min(1).max(500).describe("Decision title / question"),
        description: z
          .string()
          .max(5000)
          .optional()
          .nullable()
          .describe("Optional longer description"),
        status: z
          .enum(["draft", "active", "archived"])
          .optional()
          .describe("Initial status (default: draft)"),
      },
    },
    async (args) => {
      try {
        const input = createDecisionSchema.parse(args);
        const decision = await decisionsService.createDecision(userId, input);
        return toolOk({ decision });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  server.registerTool(
    "get_decision_state",
    {
      title: "Get decision state",
      description:
        "Aggregated agent context: decision + criteria + alternatives + pairwise judgments.",
      inputSchema: {
        decisionId: z.string().uuid().describe("Decision id"),
      },
    },
    async ({ decisionId }) => {
      try {
        const decision = await decisionsService.getDecision(userId, decisionId);
        const state = await snapshotsService.serializeDecisionState(decisionId);
        return toolOk({
          decision,
          criteria: state.criteria,
          alternatives: state.alternatives,
          pairwiseJudgments: state.pairwiseJudgments,
          serializedAt: state.serializedAt,
        });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  server.registerTool(
    "list_criteria",
    {
      title: "List criteria",
      description: "List criteria for a decision (ordered).",
      inputSchema: {
        decisionId: z.string().uuid().describe("Decision id"),
      },
    },
    async ({ decisionId }) => {
      try {
        const criteria = await criteriaService.listCriteria(userId, decisionId);
        return toolOk({ criteria });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  server.registerTool(
    "upsert_criterion",
    {
      title: "Upsert criterion",
      description:
        "Create a criterion (provide decisionId + name) or update an existing one (provide criterionId).",
      inputSchema: {
        decisionId: z
          .string()
          .uuid()
          .optional()
          .describe("Required when creating"),
        criterionId: z
          .string()
          .uuid()
          .optional()
          .describe("Required when updating"),
        name: z.string().min(1).max(300).optional().describe("Criterion name"),
        description: z.string().max(5000).optional().nullable(),
        parentId: z.string().uuid().optional().nullable(),
        sortOrder: z.number().int().optional(),
        source: z.enum(["user", "agent", "import"]).optional(),
      },
    },
    async (args) => {
      try {
        if (args.criterionId) {
          const patch = updateCriterionSchema.parse({
            name: args.name,
            description: args.description,
            parentId: args.parentId,
            sortOrder: args.sortOrder,
            source: args.source,
          });
          const criterion = await criteriaService.updateCriterion(
            userId,
            args.criterionId,
            patch,
          );
          return toolOk({ criterion, action: "updated" });
        }

        if (!args.decisionId) {
          throw new AppError(
            "VALIDATION_ERROR",
            "decisionId is required when creating a criterion",
            422,
          );
        }

        const input = createCriterionSchema.parse({
          name: args.name,
          description: args.description,
          parentId: args.parentId,
          sortOrder: args.sortOrder,
          source: args.source ?? "agent",
        });
        const criterion = await criteriaService.createCriterion(
          userId,
          args.decisionId,
          input,
        );
        return toolOk({ criterion, action: "created" });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  server.registerTool(
    "propose_pairwise",
    {
      title: "Propose pairwise judgments",
      description:
        "Upsert pairwise Saaty comparisons as proposals only (status is always forced to proposal; never commits subjective judgments).",
      inputSchema: {
        decisionId: z.string().uuid().describe("Decision id"),
        judgments: z
          .array(
            z.object({
              parentCriterionId: z.string().uuid().nullable().optional(),
              level: pairwiseLevelSchema,
              leftId: z.string().uuid(),
              rightId: z.string().uuid(),
              value: saatyValueSchema.describe("Saaty scale 1/9 … 9"),
            }),
          )
          .min(1),
      },
    },
    async ({ decisionId, judgments }) => {
      try {
        // Always proposal — ignore any client attempt to commit.
        const proposed = judgments.map((j) => ({
          parentCriterionId: j.parentCriterionId ?? null,
          level: j.level,
          leftId: j.leftId,
          rightId: j.rightId,
          value: j.value,
          status: "proposal" as const,
        }));
        const result = await pairwiseService.upsertPairwise(
          userId,
          decisionId,
          proposed,
        );
        return toolOk({ judgments: result, status: "proposal" });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  server.registerTool(
    "create_snapshot",
    {
      title: "Create snapshot",
      description:
        "Immutable point-in-time copy of the decision model. prompt and outputSummary are required for MCP (agent audit trail).",
      inputSchema: {
        decisionId: z.string().uuid().describe("Decision id"),
        label: z.string().min(1).max(300).describe("Human-readable label"),
        prompt: z
          .string()
          .min(1)
          .describe("Prompt / agent context that led to this snapshot"),
        outputSummary: z
          .string()
          .min(1)
          .describe("Short summary of agent output / rationale"),
        chatId: z.string().uuid().optional().nullable(),
      },
    },
    async ({ decisionId, label, prompt, outputSummary, chatId }) => {
      try {
        const snapshot = await snapshotsService.createSnapshot(
          userId,
          decisionId,
          { label, prompt, outputSummary, chatId },
        );
        return toolOk({ snapshot });
      } catch (err) {
        return toolErr(err);
      }
    },
  );

  return server;
}
