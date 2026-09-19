# Agent notes

This is a **Go** CLI (`ahp`), not Next.js or Python.

- Read `docs/VISION.md` and `docs/architecture.md` before large changes.
- Domain math: `internal/engine`; CSV I/O: `internal/workspace`.
- MCP tools wrap the same workspace functions as the CLI. Do not fork logic.
- `propose_pairwise` must keep `status=proposal`.
- Run `go test ./...` after engine changes.
- Prefer `ahp status` / `suggest_repairs` over inventing consistency fixes by hand.
- When choosing alternatives/prices for a purchase-like decision, follow `.cursor/skills/ahp-data-integrity/SKILL.md`, then `ahp constrain` + `ahp doctor --purchase`; use `ahp rate --prefer lower [--refresh] [--stretch]` for `alt:value`.
- Facility filters (parking etc.): `ahp constrain parking --must-have` with truthy attributes (`1`/`true`/`yes`).
- Multi-source prices: `ahp quote add …` then `ahp quote pick value --prefer lower` (writes attributes + audit note).
- Drop shortlist noise with `ahp remove-alternative <id>` (cleans attrs + pairs).
- Pasteable ranking: `ahp share --whatsapp`.
