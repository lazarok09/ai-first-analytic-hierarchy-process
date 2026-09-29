# Agent notes

This is a **Go** CLI (`ahp`), not Next.js or Python.

- Read `docs/VISION.md` and `docs/architecture.md` before large changes.
- **UI is one surface, never both.** A UI request is either the workspace report HTML (`internal/render` → `output/report.html`, via `ahp compute` / `ahp render` / `ahp open`) or the landing page (`web/`). Infer which one from the prompt. If it is not clear, ask the prompter before editing. Do not change the report and the landing page in the same task.
- Domain math: `internal/engine`; CSV I/O: `internal/workspace`.
- MCP tools wrap the same workspace functions as the CLI. Do not fork logic.
- `propose_pairwise` must keep `status=proposal`.
- Run `go test ./...` after engine changes.
- Prefer `ahp status` / `suggest_repairs` over inventing consistency fixes by hand.
- **Data integrity ≠ method integrity:** purchase prices → `.cursor/skills/ahp-data-integrity/SKILL.md`; defending a ranking / absolute / Gaussian → `.cursor/skills/ahp-method-strict/SKILL.md` (see `docs/PLAN_SANTOS_AHP_METHOD_STRICT.md`).
- When choosing alternatives/prices for a purchase-like decision, follow `ahp-data-integrity`, then `ahp constrain` + `ahp doctor --purchase`; use `ahp rate --prefer lower [--refresh] [--stretch]` for `alt:value`.
- Facility filters (parking etc.): `ahp constrain parking --must-have` with truthy attributes (`1`/`true`/`yes`).
- Direction for absolute/Gaussian: `ahp constrain <c> --prefer higher|lower` (min/max optional), then `ahp doctor --method` / `ahp gaussian` / `ahp absolute`. Saaty remains default; Gaussian scores have **no CR**.
- Multi-source prices: `ahp quote add …` then `ahp quote pick value --prefer lower` (writes attributes + audit note).
- Drop shortlist noise with `ahp remove-alternative <id>` (cleans attrs + pairs).
- Pasteable ranking: `ahp share --whatsapp`.
- Run history (opt-in): `ahp compute --journal` then `ahp journal list|show` (snapshots under `tmp/journal/`; see `docs/PLAN_JOURNAL.md`).
