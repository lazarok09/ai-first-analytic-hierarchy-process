"""In-memory ChatKit Store for local MVP (threads + items)."""

from __future__ import annotations

from typing import Any

from chatkit.store import NotFoundError, Store
from chatkit.types import Attachment, Page, ThreadItem, ThreadMetadata


class MemoryStore(Store[dict[str, Any]]):
    def __init__(self) -> None:
        self._threads: dict[str, ThreadMetadata] = {}
        self._items: dict[str, list[ThreadItem]] = {}
        self._attachments: dict[str, Attachment] = {}

    async def load_thread(self, thread_id: str, context: dict[str, Any]) -> ThreadMetadata:
        thread = self._threads.get(thread_id)
        if thread is None:
            raise NotFoundError(f"Thread {thread_id} not found")
        return thread

    async def save_thread(self, thread: ThreadMetadata, context: dict[str, Any]) -> None:
        self._threads[thread.id] = thread

    async def load_threads(
        self,
        limit: int,
        after: str | None,
        order: str,
        context: dict[str, Any],
    ) -> Page[ThreadMetadata]:
        threads = list(self._threads.values())
        threads.sort(
            key=lambda t: getattr(t, "created_at", None) or t.id,
            reverse=(order == "desc"),
        )
        start = 0
        if after:
            for i, t in enumerate(threads):
                if t.id == after:
                    start = i + 1
                    break
        slice_ = threads[start : start + limit]
        has_more = start + limit < len(threads)
        return Page(
            data=slice_,
            has_more=has_more,
            after=slice_[-1].id if slice_ and has_more else None,
        )

    async def load_thread_items(
        self,
        thread_id: str,
        after: str | None,
        limit: int,
        order: str,
        context: dict[str, Any],
    ) -> Page[ThreadItem]:
        items = list(self._items.get(thread_id, []))
        items.sort(key=lambda i: i.id, reverse=(order == "desc"))
        start = 0
        if after:
            for i, item in enumerate(items):
                if item.id == after:
                    start = i + 1
                    break
        slice_ = items[start : start + limit]
        has_more = start + limit < len(items)
        return Page(
            data=slice_,
            has_more=has_more,
            after=slice_[-1].id if slice_ and has_more else None,
        )

    async def add_thread_item(
        self, thread_id: str, item: ThreadItem, context: dict[str, Any]
    ) -> None:
        self._items.setdefault(thread_id, []).append(item)

    async def save_item(
        self, thread_id: str, item: ThreadItem, context: dict[str, Any]
    ) -> None:
        items = self._items.setdefault(thread_id, [])
        for i, existing in enumerate(items):
            if existing.id == item.id:
                items[i] = item
                return
        items.append(item)

    async def load_item(
        self, thread_id: str, item_id: str, context: dict[str, Any]
    ) -> ThreadItem:
        for item in self._items.get(thread_id, []):
            if item.id == item_id:
                return item
        raise NotFoundError(f"Item {item_id} not found in thread {thread_id}")

    async def delete_thread(self, thread_id: str, context: dict[str, Any]) -> None:
        self._threads.pop(thread_id, None)
        self._items.pop(thread_id, None)

    async def delete_thread_item(
        self, thread_id: str, item_id: str, context: dict[str, Any]
    ) -> None:
        items = self._items.get(thread_id, [])
        self._items[thread_id] = [i for i in items if i.id != item_id]

    async def save_attachment(self, attachment: Attachment, context: dict[str, Any]) -> None:
        self._attachments[attachment.id] = attachment

    async def load_attachment(
        self, attachment_id: str, context: dict[str, Any]
    ) -> Attachment:
        attachment = self._attachments.get(attachment_id)
        if attachment is None:
            raise NotFoundError(f"Attachment {attachment_id} not found")
        return attachment

    async def delete_attachment(self, attachment_id: str, context: dict[str, Any]) -> None:
        self._attachments.pop(attachment_id, None)
