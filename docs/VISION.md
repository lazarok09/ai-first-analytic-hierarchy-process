# Product vision — AI-first Analytic Hierarchy Process (AHP)

Cognitive AHP, reified for the web: multimodal data fusion, live hierarchical matrices, MCP-driven orchestration, and chronotopic (time-aware) snapshots.

## Problem

Two failure modes show up when people use AI for multi-criteria decisions:

1. **Tedious criterion intake** — For common comparisons (e.g. choosing a phone), criteria and factual attributes already exist on the public web, but wiring them into an AHP model by hand is slow.
2. **Outsourced subjectivity** — An LLM can pre-fill everything and “pick a winner.” That may be mathematically consistent (weights, CR), but it is *interpretively wrong* when the human should own 100% of the subjective judgments.

The product separates **objective data assembly** (AI-assisted, reviewable) from **subjective judgment** (human-owned, never silently decided by the model).

## Solution (high level)

- A web app where criteria, alternatives, and pairwise judgments are first-class, editable data.
- Users start with a **prompt** in the in-app **ChatKit** agent; tools mutate the model via REST (MCP remains optional for Cursor).
- **Snapshots** freeze model state so the user can rewind during modeling.
- UI components **appear as data arrives** (matrices materialize as comparisons are filled).

## Core product loop (MVP target)

1. Authenticate (human via web SSO — see [sso.md](./sso.md)).
2. Ask in ChatKit: “What are we deciding today?” (see [chatkit.md](./chatkit.md)).
3. Agent gathers candidate criteria/alternatives, proposes pairwise importance (**proposal** only).
4. Create the **first snapshot** of the comparison (prompt + summary).
5. Human reviews criteria, commits importance / pairwise judgments in the UI.
6. Matrices update live; history stays tied to chat sessions (`externalSessionId` = ChatKit thread).

## Chat ↔ snapshot model

- Each chat stores the **AI session id** and is the foreign key parent of snapshots (ordered by `created_at`).
- Every snapshot persists:
  - **Full prompt** (exact text sent)
  - **Summarized agent output** (preprocessed summary for history; not a dump of raw tool noise)
  - Full structured model state (criteria, alternatives, judgments, metadata)

This keeps a prompt history without bloating the UI with every tool trace.

## Design principles

| Principle | Meaning |
|-----------|---------|
| Human owns judgment | AI may propose; only the user commits subjective weights |
| Schema-first intuition | Clear relational model so “multiple data sources” is visible |
| MCP-first editing | Agents mutate state via typed tools (ChatKit in-app; MCP optional) backed by REST |
| Chronotopic snapshots | Time-travel during modeling; never lose a good intermediate state |
| Progressive disclosure UI | Show matrices/components only when underlying data exists |
| i18n from day one | English + Portuguese (Brazil) first |

## Non-goals (MVP)

- Fully automated “AI chooses the phone for you”
- GraphQL API (see [api.md](./api.md))
- Production multi-tenant billing
- Perfect multi-source fusion ML (quality gates + review UI are enough)

## Success criteria for MVP

- [ ] Sign in as a human (OAuth / Dev Login)
- [ ] Prompt in ChatKit creates a decision + criteria/alternatives
- [ ] Agent proposes pairwise judgments; human commits in the UI
- [ ] First snapshot created with prompt + output summary (ChatKit thread linked)
- [ ] (Optional) Connect Cursor MCP to the same account
- [ ] UI strings work in `en` and `pt-BR`
