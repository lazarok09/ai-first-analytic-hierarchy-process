"""AHP domain tools for the ChatKit agent (REST-backed)."""

from __future__ import annotations

from typing import Any, Literal

from agents import RunContextWrapper, function_tool
from chatkit.agents import AgentContext, ClientToolCall
from pydantic import BaseModel, Field

from ahp_chatkit.rest import AhpRestClient


class PairwiseJudgmentInput(BaseModel):
    leftId: str
    rightId: str
    value: float = Field(ge=1 / 9, le=9)
    level: Literal["criteria", "alternatives"] = "criteria"
    parentCriterionId: str | None = None


def _client(ctx: RunContextWrapper[AgentContext[dict[str, Any]]]) -> AhpRestClient:
    request_context = ctx.context.request_context
    token = request_context.get("agent_token")
    if not token:
        raise RuntimeError("Missing agent_token in ChatKit request context")
    return AhpRestClient(token)


@function_tool(
    description_override=(
        "Create a new AHP decision (comparison). Prefer this when the user names "
        "alternatives to compare (e.g. iPhone vs Pixel)."
    )
)
async def create_decision(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    title: str,
    description: str | None = None,
) -> dict[str, Any]:
    client = _client(ctx)
    return await client.request(
        "POST",
        "/api/v1/decisions",
        json={"title": title, "description": description, "status": "draft"},
    )


@function_tool(
    description_override=(
        "Load aggregated decision state: decision + criteria + alternatives + pairwise."
    )
)
async def get_decision_state(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str,
) -> dict[str, Any]:
    client = _client(ctx)
    return await client.request("GET", f"/api/v1/decisions/{decision_id}/state")


@function_tool(description_override="List criteria for a decision.")
async def list_criteria(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str,
) -> dict[str, Any]:
    client = _client(ctx)
    return await client.request("GET", f"/api/v1/decisions/{decision_id}/criteria")


@function_tool(
    description_override=(
        "Create a criterion (decision_id + name) or update one (criterion_id). "
        "Set source to agent when you invent criteria from research."
    )
)
async def upsert_criterion(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    name: str | None = None,
    decision_id: str | None = None,
    criterion_id: str | None = None,
    description: str | None = None,
    parent_id: str | None = None,
    sort_order: int | None = None,
) -> dict[str, Any]:
    client = _client(ctx)
    if criterion_id:
        body: dict[str, Any] = {}
        if name is not None:
            body["name"] = name
        if description is not None:
            body["description"] = description
        if parent_id is not None:
            body["parentId"] = parent_id
        if sort_order is not None:
            body["sortOrder"] = sort_order
        return await client.request("PATCH", f"/api/v1/criteria/{criterion_id}", json=body)
    if not decision_id or not name:
        raise ValueError("decision_id and name are required when creating a criterion")
    return await client.request(
        "POST",
        f"/api/v1/decisions/{decision_id}/criteria",
        json={
            "name": name,
            "description": description,
            "parentId": parent_id,
            "sortOrder": sort_order,
            "source": "agent",
        },
    )


@function_tool(
    description_override="Create an alternative (option being compared) under a decision."
)
async def create_alternative(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str,
    name: str,
    description: str | None = None,
    sort_order: int | None = None,
) -> dict[str, Any]:
    client = _client(ctx)
    return await client.request(
        "POST",
        f"/api/v1/decisions/{decision_id}/alternatives",
        json={
            "name": name,
            "description": description,
            "sortOrder": sort_order,
            "source": "agent",
        },
    )


@function_tool(
    description_override=(
        "Propose pairwise Saaty judgments (always status=proposal). "
        "Never mark committed — humans own subjective judgment."
    )
)
async def propose_pairwise(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str,
    judgments: list[PairwiseJudgmentInput],
) -> dict[str, Any]:
    client = _client(ctx)
    normalized = [
        {
            "parentCriterionId": j.parentCriterionId,
            "level": j.level,
            "leftId": j.leftId,
            "rightId": j.rightId,
            "value": j.value,
            "status": "proposal",
        }
        for j in judgments
    ]
    return await client.request(
        "PUT",
        f"/api/v1/decisions/{decision_id}/pairwise",
        json={"judgments": normalized},
    )


@function_tool(
    description_override=(
        "Create an immutable snapshot with prompt + outputSummary for audit lineage."
    )
)
async def create_snapshot(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str,
    label: str,
    prompt: str,
    output_summary: str,
    chat_id: str | None = None,
) -> dict[str, Any]:
    client = _client(ctx)
    return await client.request(
        "POST",
        f"/api/v1/decisions/{decision_id}/snapshots",
        json={
            "label": label,
            "prompt": prompt,
            "outputSummary": output_summary,
            "chatId": chat_id,
        },
    )


@function_tool(
    description_override=(
        "Ensure a local chats row exists for this ChatKit thread "
        "(externalSessionId = thread id). Optionally link a decisionId."
    )
)
async def ensure_chat_lineage(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    thread_id: str,
    title: str | None = None,
    decision_id: str | None = None,
) -> dict[str, Any]:
    client = _client(ctx)
    body: dict[str, Any] = {"externalSessionId": thread_id}
    if title:
        body["title"] = title
    if decision_id:
        body["decisionId"] = decision_id
    return await client.request("POST", "/api/v1/chats/ensure", json=body)


@function_tool(
    description_override=(
        "Ask the web UI to refresh so criteria / pairwise panels pick up agent writes."
    )
)
async def refresh_decision_ui(
    ctx: RunContextWrapper[AgentContext[dict[str, Any]]],
    decision_id: str | None = None,
) -> dict[str, Any]:
    ctx.context.client_tool_call = ClientToolCall(
        name="refresh_decision_ui",
        arguments={"decisionId": decision_id},
    )
    return {"ok": True, "decisionId": decision_id}


AHP_TOOLS = [
    create_decision,
    get_decision_state,
    list_criteria,
    upsert_criterion,
    create_alternative,
    propose_pairwise,
    create_snapshot,
    ensure_chat_lineage,
    refresh_decision_ui,
]

AHP_INSTRUCTIONS = """
You are the in-app AHP Studio agent. Users start with a natural-language prompt
(e.g. "compare iPhone 16 vs Pixel 9"). Your job is to assemble the decision model;
humans own subjective judgment.

Rules:
1. Prefer create_decision + create_alternative + upsert_criterion from the prompt.
2. Call get_decision_state before mutating an existing decision.
3. Pairwise writes MUST use propose_pairwise (status proposal only). Never claim
   judgments are final — invite the user to review/commit in the UI.
4. After meaningful model changes, call ensure_chat_lineage with the ChatKit
   thread id, then create_snapshot with the user prompt + a short outputSummary,
   then refresh_decision_ui.
5. Ask clarifying questions when alternatives or criteria are ambiguous.
6. Be concise. Do not dump raw JSON unless the user asks.
""".strip()
