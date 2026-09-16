# Roadmap — SOTA AHP CLI, catalog, and command-as-doc

Treat **each CLI command’s Cobra help as the source of truth**, mirror it 1:1 in MCP, then deepen the **engine** toward literature-grade AHP (incomplete matrices, Ideal synthesis, sensitivity, attribute→priority). Docs become generated from that catalog—not hand-duplicated.

## Current baseline (gaps)

| Area | Today | Gap |
|------|--------|-----|
| Help/docs | `Short` only | No `Long` / `Example`; README lists commands once |
| Catalog | Split CLI vs MCP | MCP has `missing_pairs` / `suggest_repairs`; CLI does not |
| Incomplete matrices | Equal-weight fallback | Not SOTA (Harker / RGMM) |
| Priorities | Eigenvector only | No RGMM / Ideal mode |
| Attributes | Display-only | No absolute measurement path |
| Robustness | Repair hints only | No sensitivity (listed in [MVP.md](./MVP.md) post-MVP) |

## Principle: command-as-doc

1. Add `internal/catalog` — one entry per operation: id, CLI name, MCP name, summary, long prose, examples, flags, writes?, agent rules.
2. Cobra `Long` / `Example` and MCP `WithDescription` **import from catalog** (no fork).
3. New commands:
   - `ahp catalog` — machine-readable JSON of the surface
   - `ahp docs [command]` — print markdown for one or all commands
   - optional `ahp docs --write docs/commands/` for CI/docs sync
4. Thin human docs: `README` + [VISION.md](./VISION.md) stay narrative; [mcp.md](./mcp.md) / command pages **generated or linked** from catalog.
5. Agent rule in `AGENTS.md`: prefer `ahp docs <cmd>` / `ahp catalog` over inventing flags.

## Phase 0 — Catalog & docs parity (no new math)

**Goal:** every operation documented once; CLI ↔ MCP aligned.

| Command | Doc focus | Work |
|---------|-----------|------|
| `ahp` (root) | Workflow: init → structure → propose → human commit → compute → status | Rich root `Long` + agent loop |
| `init` | Layout contract (`ahp.toml`, CSVs) | Examples; refuse overwrite unless `--force` |
| `set-goal` | Goal text owns the decision narrative | |
| `add-criterion` / `add-alternative` | Hierarchy (`parent_id`), id stability | Document matrix naming |
| `set-attribute` | Facts ≠ weights | Clear “never enters eigenvector” |
| `set-pairwise` | Saaty scale, reciprocity, `proposal` vs `committed` | Verbal scale table in `Long` |
| `commit-proposals` | Human gate | Warn if CR would worsen |
| `compute` / `render` | What files write; `--include-proposals` | Deprecate duplicate or make `render` alias explicitly |
| `validate` | Schema + graph sanity | Expand checks (orphan parents, unknown ids) |
| `status` | Compact readiness | JSON via `--json` |
| `open` | Opens report | |
| `mcp` | Points to tool catalog | |
| **new** `missing-pairs` | Gaps to fill | Wrap `Status.Missing` (MCP already has) |
| **new** `suggest-repairs` | CR repair loop | Wrap existing engine (MCP already has) |
| **new** `show` / `get-state` | Read-only hierarchy + matrices | Align with MCP `get_state` |
| **new** `catalog` / `docs` | Meta-docs | Generate from `internal/catalog` |

**Exit:** `ahp docs` covers all commands; [mcp.md](./mcp.md) matches `ahp catalog --format mcp`.

## Phase 1 — SOTA math (engine first, then commands)

Keep CSV as source of truth; add options in `ahp.toml` + flags.

| Capability | Literature | Command / flag | Notes |
|------------|------------|----------------|-------|
| Incomplete pairwise | Harker method and/or RGMM on observed pairs | `compute --incomplete=harker\|rgmm\|none` | Replace equal-weight fallback; still list `missing` |
| Alternate local priorities | Row geometric mean (RGMM) vs eigenvector | `compute --priority=eigen\|rgmm` | Report both CR and GCI when RGMM |
| Geometric consistency | GCI | Always in matrix payload + report | Complements Saaty CR |
| Synthesis mode | Distributive vs Ideal | `compute --synthesis=distributive\|ideal` | Ideal reduces some rank-reversal artifacts |
| Sensitivity | ±δ on criterion weights; rank stability | **new** `ahp sensitivity` | Write `output/sensitivity.json` + report section |
| Attribute → local priorities | Absolute measurement / ratings | **new** `ahp rate` or `derive-from-attributes` | Explicit, opt-in; never silent |
| Optional Gaussian view | Comparative insight | Report section only | Saaty remains the human judgment path |

**Tests:** extend `internal/engine` with fixtures for incomplete, Ideal vs distributive, GCI.

## Phase 2 — Decision quality UX

| Command | Purpose |
|---------|---------|
| `ahp explain [alt]` | Why rank: contribution by criterion (from synthesis) |
| `ahp diff` | Compare last two `compute.json` rankings |
| `ahp checklist` | “Ready for human commit?” (complete, CR, proposals pending) |
| Report upgrades | Verbal Saaty legend, Ideal/distributive toggle note, sensitivity tornado |

Still no web app / Auth (per [VISION.md](./VISION.md)).

## Phase 3 — Stretch (only if needed)

- **Group AHP:** `judges/` or `judge` column + `aggregate --method=aij|aip`
- **Snapshots:** `ahp snapshot save|list` copying `data/` + `output/`
- **BOCR / benefits-costs:** structured criterion types in schema
- Fuzzy AHP / ANP: **out of scope** unless explicitly requested

## Command catalog target shape

```text
Lifecycle:   init → set-goal → add-* → set-attribute → set-pairwise
Gate:        missing-pairs → suggest-repairs → commit-proposals → validate
Solve:       compute / render → status → open → explain → sensitivity
Meta:        catalog / docs / version / mcp
```

Agents use the same ids via MCP (`propose_pairwise` still **never** commits).

## Implementation order

1. `internal/catalog` + enrich Cobra/MCP help + `ahp docs` / `ahp catalog`
2. CLI parity: `missing-pairs`, `suggest-repairs`, `show --json`
3. Engine: incomplete (Harker) + Ideal synthesis + GCI in report
4. `sensitivity` + report section
5. Opt-in `derive-from-attributes` + example workspace update
6. Generate `docs/commands/*.md` in CI; slim narrative docs

## Success criteria

- `ahp docs compute` alone teaches Saaty, CR, proposals, outputs
- CLI and MCP lists stay identical via catalog
- Incomplete matrices produce usable priorities without pretending completeness
- Report shows Ideal vs distributive and sensitivity of #1 rank
- `go test ./...` covers new engine paths
