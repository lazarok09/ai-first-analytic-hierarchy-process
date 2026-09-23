# Roadmap

Make `ahp` feel like a **great local CLI** first — the kind people reach for like `gh`, `git`, `kubectl`, or `terraform` — while staying true to [VISION.md](./VISION.md): CSV on disk, human-owned judgment, MCP for agents, no hosted app.

Math upgrades still matter; they land *behind* a surface that teaches the next step, scripts cleanly, and never surprises.

**Baseline:** v0.3.0 — flat Cobra verbs, `Short`-only help, text dumps, equal-weight incomplete fallback. MVP done; the product is not yet delightful.

---

## Inspiration (steal the feeling, not the domain)

| Tool | Steal this |
|------|------------|
| **git** | Upward root find; `status` that teaches; meaningful exit codes; porcelain vs plumbing |
| **gh** | Next-step hints; `--json` + field selection; interactive prompts when TTY; errors that name the fix command |
| **kubectl** | Noun/verb consistency (`get` / `set` / `describe`); `-o json\|table\|yaml`; stable resource shapes |
| **terraform** | `plan` before `apply`; dry-run; show ranking/CR delta before committing proposals |
| **cargo / brew** | `doctor`; one command that says “here’s what’s wrong” |
| **ripgrep / jq** | Quiet when piped; machine output is the default for agents; humans get tables |
| **gum / survey** | Progressive disclosure — interactive only when stdin is a TTY |

Non-goals stay: no Auth, no app DB, no web SPA unless asked. Optional local `serve` for the HTML report is fine later.

---

## Design principles for the new surface

1. **Status is the home screen.** Like `git status` / `gh pr status`: completeness, CR, proposals, ranking peek, and **exactly one recommended next command**.
2. **Plan before apply.** Proposals are the staging area; `ahp plan` shows what `apply` would do to CR and ranks (terraform muscle memory).
3. **One output contract.** Human table when TTY; JSON when `--json`, `--format json`, or stdout is not a TTY. Every read command supports it.
4. **Errors are actionable.** Prefer `incomplete: 3 pairs — run: ahp pair missing` over opaque failures.
5. **Exit codes are API.** Scripts and CI can branch without scraping text (see below).
6. **Agents and humans share verbs; differ on defaults.** Humans may commit; agents default to proposal / never silent-commit.
7. **Catalog is how help stays honest.** Cobra `Long`/`Example` and MCP descriptions import one registry — docs don’t drift.

### Exit code contract (proposed)

| Code | Meaning |
|------|---------|
| 0 | OK / ready |
| 1 | Usage / flag error |
| 2 | Workspace incomplete (missing pairs or structure) |
| 3 | Inconsistent (CR or GCI above threshold) |
| 4 | Proposals pending (blocked for “clean” checks) |
| 5 | I/O or engine failure |

`validate` / `doctor` / CI wrappers use these; humans still see prose.

---

## Target surface (north-star CLI)

Grouped like a tool people memorize — not a bag of `set-*` helpers.

```text
Orientation     status · doctor · tree · next
Structure       init · goal · criterion · alternative · attribute
Judgment        pair · plan · apply · reject
Solve           compute · explain · sensitivity · diff
Inspect         get · describe · open · docs · catalog
Agent           mcp
Meta            completion · version
```

Aliases keep old muscle memory for a while (`set-pairwise` → `pair set`, `commit-proposals` → `apply`, `validate` → `doctor`).

### Command sketches (what “better” looks like)

**`ahp status`** (git + gh)

```text
Choose a vendor  ·  examples/vendor-selection
Matrices  complete ✓   Consistency  CR≤0.10 ✓   Proposals  2 pending
Ranking   1. Acme 0.41   2. Globex 0.33   3. Initech 0.26

Next:  ahp plan          # preview committing 2 proposals
       ahp pair missing  # or fill remaining gaps
```

**`ahp doctor`** (cargo/brew) — schema orphans, unknown ids, reciprocal gaps, empty matrices, CR hotspots; exit codes as above.

**`ahp tree`** — ASCII hierarchy (criteria → children → alt matrices), like `kubectl` / `tree`.

**`ahp next`** — print the single best next action (scriptable: `ahp next --json`).

**`ahp pair`** (judgment UX)

| Subcommand | Role |
|------------|------|
| `pair missing` | List gaps (MCP `missing_pairs`) |
| `pair set …` | Write one judgment (`--as proposal\|committed`) |
| `pair ask` | Interactive Saaty walk over missing pairs (TTY only; verbal scale prompts) |
| `pair repairs` | CR repair suggestions (MCP `suggest_repairs`) |

Default for non-TTY / `AHP_AGENT=1`: `--as proposal`. Default for human TTY can stay committed *after* an explicit confirm or `pair ask`.

**`ahp plan` / `ahp apply`** (terraform)

- `plan` — recompute *as if* proposals were committed; show Δrank, ΔCR per matrix; no writes to `status`.
- `apply` — flip proposals → committed (filters: matrix / pair); supports `--dry-run` (= plan) and `-y`.

**`ahp get` / `ahp describe`** (kubectl)

```bash
ahp get ranking -o json
ahp get pairs --status proposal
ahp describe matrix criteria
ahp get criteria -o table
```

**`ahp compute`** — still the solve+write verb; gains `--incomplete`, `--synthesis`, `--priority` later. `render` remains an alias.

**`ahp explain [alt]`** — contribution by criterion (decision quality).

**`ahp diff`** — last two `compute.json` rankings (or plan vs HEAD).

**`ahp sensitivity`** — ±δ on weights; tornado in report + `output/sensitivity.json`.

**`ahp docs` / `ahp catalog`** — command-as-doc; agents prefer these over inventing flags.

**`ahp completion`** — bash/zsh/fish/powershell (table stakes for “serious CLI”).

**Later / stretch UX**

| Idea | Inspiration |
|------|-------------|
| `ahp watch` | recompute + refresh report on CSV change |
| `ahp import` / `export` | spreadsheet ↔ workspace |
| `ahp snapshot` | tagged copies of `data/` + `output/` |
| `ahp serve` | local static viewer for `report.html` (not a product app) |
| Group AHP / BOCR | only if users ask |

---

## Phases

### Phase A — CLI craft (ship the feeling)

No new AHP math. Make the binary feel intentional.

| Deliverable | Detail |
|-------------|--------|
| Output layer | Shared printer: table / json / quiet; detect TTY; `--json` on reads |
| Exit codes | Implement contract on `status`, `doctor`/`validate`, `compute` |
| `status` v2 | Next-step block; ranking peek; proposal count |
| `doctor` | Replace/narrow `validate`; schema + graph + CR |
| `tree`, `next` | Hierarchy view; single recommended action |
| `pair` + `plan`/`apply` | Staging-area workflow; deprecate flat names gradually |
| `get` / `describe` | Resource-oriented inspect; MCP maps to same payloads |
| Completions | `ahp completion …` |
| Catalog seed | `internal/catalog` feeds `Long`/`Example` + MCP + `docs`/`catalog` |
| Agent defaults | Document + enforce proposal-default when agent-ish (env / non-TTY) |

**Exit:** A new user can run `init` → `status` → follow `Next:` without reading the README. An agent can drive the loop with `--json` only.

### Phase B — Decision quality (still surface-led)

| Deliverable | Detail |
|-------------|--------|
| `explain` | Criterion contributions for an alternative |
| `diff` | Ranking / weight deltas |
| `sensitivity` | Robustness of #1; report section |
| Report polish | Verbal Saaty legend; clear proposal banner; link “what to do next” |
| Checklist mode | `doctor --strict` or `status --ready` for “human may decide” |

**Exit:** You can defend a ranking in the terminal without opening a spreadsheet.

### Phase C — SOTA engine (behind stable flags)

Defaults unchanged so old workspaces don’t silently shift.

| Capability | Flag / command | Notes |
|------------|----------------|-------|
| Incomplete pairwise | `compute --incomplete=harker\|rgmm\|none` | Replace equal-weight fallback |
| RGMM priorities | `compute --priority=eigen\|rgmm` | Emit GCI with CR |
| Ideal synthesis | `compute --synthesis=distributive\|ideal` | |
| Attribute ratings | `ahp rate` (opt-in) | `--mode=saaty` lossy proposals; `--mode=sum-norm` preview |
| Optional Gaussian view | `ahp gaussian` / `compute --method=compare` | Comparative only; Saaty stays canonical — see [PLAN_SANTOS_AHP_METHOD_STRICT.md](./PLAN_SANTOS_AHP_METHOD_STRICT.md) |

**Tests:** fixtures in `internal/engine`; `go test ./...` green.

### Phase D — Stretch

- Group AHP (`aggregate --method=aij|aip`)
- Snapshots, import/export, `watch`, optional `serve`
- Fuzzy AHP / ANP: **out of scope** unless requested

---

## CLI ↔ MCP

MCP remains a thin mirror of workspace functions. After Phase A:

- Tool names can stay snake_case; catalog holds the CLI↔MCP map.
- New MCP tools only when CLI gains a real operation (`plan`, `explain`, …).
- `propose_pairwise` **never** commits — unchanged invariant.

---

## Implementation order

1. Shared output + exit-code helpers (`internal/cliout` or similar)
2. `status` v2 + `doctor` + `next` + `tree`
3. `pair` / `plan` / `apply` (+ aliases from old verbs)
4. `get` / `describe` + universal `--json`
5. `internal/catalog` + `docs` / `catalog` + completions
6. `explain` / `diff` / `sensitivity` + report upgrades
7. Engine: incomplete → Ideal/GCI → `rate`

---

## Success criteria

- [ ] `ahp status` alone tells a human what to do next
- [ ] `ahp plan` before `apply` is the normal commit path
- [ ] Every inspect command speaks JSON for agents and tables for people
- [ ] Exit codes distinguish incomplete vs inconsistent vs pending proposals
- [ ] Interactive `pair ask` exists for humans; agents never hit it by accident
- [ ] Help/MCP/docs cannot drift (`catalog` is source of truth)
- [ ] Incomplete matrices and Ideal/sensitivity exist *without* breaking default eigenvector behavior
- [ ] `go test ./...` covers output contracts and new engine paths

## How to use this doc

Prefer shipping **Phase A** slices end-to-end (one verb that feels great) over scattering half-done math flags. When choosing work: if it doesn’t improve `status` → `Next:` or agent JSON, it’s probably Phase C+.
