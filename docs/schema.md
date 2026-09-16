# Data model (Drizzle)

Schemas must make “multiple data sources” and “AI-shaped app behavior” obvious. Prefer explicit tables over opaque JSON blobs; use JSON columns only for matrix payloads and snapshot documents.

## Entities (MVP)

### `users`
Auth.js-compatible user.

- `id` (uuid pk)
- `name`, `email` (unique), `emailVerified`, `image`
- `locale` (`en` | `pt-BR`, default `en`)
- `createdAt`, `updatedAt`

### `accounts` / `sessions` / `verification_tokens`
Standard Auth.js adapter tables.

### `api_keys`
Agent connect tokens (MVP bridge before full MCP OAuth AS).

- `id`, `userId` → users
- `name`, `keyHash`, `prefix` (for display)
- `scopes` (text/json)
- `createdAt`, `lastUsedAt`, `revokedAt`

### `chats`
One AI modeling session.

- `id` (uuid)
- `userId` → users
- `externalSessionId` (nullable text) — Cursor/ChatGPT session id when known
- `title`
- `decisionId` → decisions (nullable until decision created)
- `createdAt`, `updatedAt`

### `decisions`
The decision problem.

- `id`, `userId`
- `title`, `description`
- `status` (`draft` | `active` | `archived`)
- `createdAt`, `updatedAt`

### `criteria`
- `id`, `decisionId`
- `parentId` (nullable self-FK for hierarchy)
- `name`, `description`
- `sortOrder`
- `source` (`user` | `agent` | `import`)
- `createdAt`, `updatedAt`

### `alternatives`
- `id`, `decisionId`
- `name`, `description`
- `sortOrder`
- `source`
- `createdAt`, `updatedAt`

### `pairwise_judgments`
Saaty scale between two siblings under the same parent (or root).

- `id`, `decisionId`
- `parentCriterionId` (nullable = comparing under goal)
- `leftId`, `rightId` — criterion or alternative ids (discriminated by `level`: `criteria` | `alternatives`)
- `value` (real; Saaty 1–9 or reciprocal)
- `status` (`proposal` | `committed`) — **proposals are agent-writable; committed preferred via human UI**
- `createdAt`, `updatedAt`
- Unique: (`decisionId`, `parentCriterionId`, `level`, `leftId`, `rightId`)

### `snapshots`
Immutable history, FK to chat for prompt lineage.

- `id`, `decisionId`, `chatId` (nullable if manual snapshot)
- `label`
- `prompt` (text, full)
- `outputSummary` (text, AI-preprocessed summary)
- `state` (json) — full serialized model at that moment
- `createdAt`

### `data_source_records` (MVP stub ok)
Provenance for ingested facts.

- `id`, `decisionId`, `entityType`, `entityId`
- `url`, `provider`, `rawExcerpt`
- `qualityScore` (0–1), `discrepancyFlags` (json)
- `createdAt`

## Snapshot policy

- Creating a snapshot copies current criteria, alternatives, and judgments into `state`.
- Chat-driven agent turns that mutate the model should call `create_snapshot` with full prompt + short summary.
- Listing snapshots: `ORDER BY created_at DESC` filtered by `chat_id` or `decision_id`.

## Migrations

- Drizzle Kit; SQLite file `data/ahp.db` for local MVP
- Keep schema Postgres-compatible (no SQLite-only types in public API)
