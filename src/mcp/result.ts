import { isAppError } from "@/lib/ahp/errors";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

/** Serialize tool payloads (Dates → ISO) as MCP text content. */
export function toolOk(data: unknown): CallToolResult {
  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(data, (_key, value) =>
          value instanceof Date ? value.toISOString() : value,
        ),
      },
    ],
  };
}

export function toolErr(err: unknown): CallToolResult {
  if (isAppError(err)) {
    return {
      isError: true,
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: {
              code: err.code,
              message: err.message,
              details: err.details,
            },
          }),
        },
      ],
    };
  }
  if (err && typeof err === "object" && "issues" in err) {
    return {
      isError: true,
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: {
              code: "VALIDATION_ERROR",
              message: "Invalid tool arguments",
              details: (err as { issues: unknown }).issues,
            },
          }),
        },
      ],
    };
  }
  console.error("[mcp] tool error", err);
  return {
    isError: true,
    content: [
      {
        type: "text",
        text: JSON.stringify({
          error: { code: "INTERNAL_ERROR", message: "Internal server error" },
        }),
      },
    ],
  };
}
