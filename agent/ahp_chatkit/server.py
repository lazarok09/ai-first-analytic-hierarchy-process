"""ChatKit server wiring for AHP Studio."""

from __future__ import annotations

from collections.abc import AsyncIterator
from typing import Any

from agents import Agent, Runner, StopAtTools
from chatkit.agents import AgentContext, simple_to_agent_input, stream_agent_response
from chatkit.server import ChatKitServer
from chatkit.types import ThreadMetadata, ThreadStreamEvent, UserMessageItem

from ahp_chatkit.memory_store import MemoryStore
from ahp_chatkit.tools import AHP_INSTRUCTIONS, AHP_TOOLS, refresh_decision_ui


class AhpChatKitServer(ChatKitServer[dict[str, Any]]):
    def __init__(self, store: MemoryStore) -> None:
        super().__init__(store)
        self.agent = Agent[AgentContext[dict[str, Any]]](
            model="gpt-4.1-mini",
            name="AHP Studio",
            instructions=AHP_INSTRUCTIONS,
            tools=AHP_TOOLS,
            tool_use_behavior=StopAtTools(
                stop_at_tool_names=[refresh_decision_ui.name]
            ),
        )

    async def respond(
        self,
        thread: ThreadMetadata,
        input_user_message: UserMessageItem | None,
        context: dict[str, Any],
    ) -> AsyncIterator[ThreadStreamEvent]:
        items_page = await self.store.load_thread_items(
            thread.id,
            after=None,
            limit=30,
            order="desc",
            context=context,
        )
        input_items = await simple_to_agent_input(list(reversed(items_page.data)))
        agent_context = AgentContext(
            thread=thread,
            store=self.store,
            request_context=context,
        )
        result = Runner.run_streamed(
            self.agent,
            input_items,
            context=agent_context,
        )
        async for event in stream_agent_response(agent_context, result):
            yield event
