# Agent notes

This is a **Go** CLI (`ahp`), not Next.js or Python.

- Read `docs/VISION.md` and `docs/architecture.md` before large changes.
- Domain math: `internal/engine`; CSV I/O: `internal/workspace`.
- MCP tools wrap the same workspace functions as the CLI. Do not fork logic.
- `propose_pairwise` must keep `status=proposal`.
- Run `go test ./...` after engine changes.
- Prefer `ahp status` / `suggest_repairs` over inventing consistency fixes by hand.
- When choosing alternatives/prices for a purchase-like decision, follow `.cursor/skills/ahp-data-integrity/SKILL.md` (local street prices, budget band, attribute↔pairwise direction).
