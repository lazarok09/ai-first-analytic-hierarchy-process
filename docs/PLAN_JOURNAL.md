# Plan: Workspace journal (per-run CSV + results snapshots)

**Status:** proposal (not implemented)  
**Date:** 2026-09-18  
**Related:** [architecture.md](./architecture.md), [VISION.md](./VISION.md), [ROADMAP.md](./ROADMAP.md), `internal/workspace.WriteOutputs`

---

## Problem

Today each `ahp compute` (and `render` / `open`) **overwrites** `output/`:

| Path | Contents |
|------|----------|
| `output/weights.csv` | Per-matrix weights, λmax, CI, CR |
| `output/ranking.csv` | Global synthesis ranking |
| `output/compute.json` | Full `ComputeResult` (matrices, missing, repairs, …) |
| `output/report.html` | Human report |
| `output/sensitivity.json` | Optional, when sensitivity writes |

`data/*.csv` is the durable source of truth, but there is **no run history**. Agents and humans cannot answer:

- What did the ranking look like before we changed pairwise / prices?
- Which compute produced CR = 0.12 vs 0.08?
- Can we recover the exact CSV + numbers from an earlier CLI run without relying on git?

Dogfood sessions (headphones, hotel, monitor) already hit this: rankings move; `output/` only keeps the latest.

---

## Goals

1. **Journal** each meaningful run under a workspace-local **tmp** tree so snapshots are cheap, local, and not the source of truth.
2. Capture **input CSV snapshot** + **numeric results** (weights, ranking, matrix values, CR / completeness flags).
3. Keep logic in `internal/workspace` so CLI and MCP share one implementation.
4. Stay true to vision: **files over servers**, CSV/JSON on disk, no app DB.

### Non-goals (v1)

- Full time-travel / `git`-like branching of the workspace.
- Automatic restore that mutates live `data/` without an explicit command.
- Journaling every read-only command (`get`, `status`, `doctor` peek).
- Cloud sync or encrypted audit trails.

---

## Proposed layout

Under the workspace root (next to `data/` and `output/`):

```text
workspace/
  ahp.toml
  data/                 # live source of truth (unchanged)
  output/               # latest compute only (unchanged)
  tmp/
    journal/
      index.jsonl       # append-only run index (one JSON object per line)
      20260918T232215Z_compute_a1b2c3/
        meta.json       # run metadata
        data/           # copy of data/*.csv (+ optional constraints/quotes)
          criteria.csv
          alternatives.csv
          pairwise.csv
          attributes.csv
          constraints.csv
          quotes.csv
        results/
          weights.csv
          ranking.csv
          compute.json
          report.html          # optional / configurable
          sensitivity.json     # if present at write time
        summary.json    # small ranking + CR peek for list UIs
```

**Why `tmp/journal/`**

- Matches the request for a **tmp** folder: ephemeral, regenerable, not canonical state.
- Keeps `data/` and `output/` semantics intact (agents already know those paths).
- Easy to gitignore (`tmp/`) without ignoring example `data/`.

**Run id**

- Suggested: `YYYYMMDDTHHMMSSZ_<command>_<short-hash>`
- Short hash = first 6–8 hex of a content hash of snapshotted `data/` (or of `compute.json`) so identical successive computes can be detected / deduped later.

---

## When to write a journal entry

### Recommended default (v1)

Write a journal entry **after a successful write of compute outputs**, i.e. inside or immediately after `Workspace.WriteOutputs`:

| Trigger | Command / MCP | Why |
|---------|---------------|-----|
| Compute outputs written | `ahp compute`, `ahp render`, `ahp open` (unless `--no-compute`), MCP `compute` | Results exist; numbers are meaningful |

Optional later (not default):

| Trigger | Note |
|---------|------|
| Explicit `ahp journal snapshot` | Manual checkpoint without recompute |
| After `pair apply` / `commit` | Judgment milestone; may only snapshot `data/` if no recompute |
| After `rate` / `constrain` / `quote pick` | Noisy; prefer opt-in |

**Do not** journal by default on pure inspection (`get`, `status`, `describe`, `doctor` without `--fix`).

### Flags / config

```toml
# ahp.toml (proposed extension)
[journal]
enabled = true          # default true once shipped, or false until stable
dir = "tmp/journal"     # relative to workspace root
keep = 50               # max runs; prune oldest
include_html = false    # report.html is large; off by default
include_sensitivity = true
```

CLI overrides:

- `--journal` / `--no-journal` on compute-family commands
- `AHP_JOURNAL=0` env kill-switch for CI / scripts

---

## What each entry stores

### `meta.json`

```json
{
  "id": "20260918T232215Z_compute_a1b2c3",
  "created_at": "2026-09-18T23:22:15Z",
  "command": "compute",
  "argv": ["ahp", "compute", "-w", ".", "--include-proposals"],
  "include_proposals": false,
  "ahp_version": "0.3.0",
  "workspace_title": "Choose a document-processing vendor",
  "data_hash": "a1b2c3d4",
  "complete": true,
  "consistent": true,
  "leader_id": "vendor_a",
  "leader_weight": 0.4123
}
```

### `data/`

Byte-copy (or CSV re-serialize) of live workspace data files present at run time. Same schemas as [schema.md](./schema.md).

### `results/`

Reuse the same writers as `WriteOutputs` so journal artifacts match `output/` formats (no second encoding of AHP math):

- `weights.csv`, `ranking.csv`, `compute.json` — **required**
- `report.html` — optional (`include_html`)
- `sensitivity.json` — copy if already on disk / if this run produced it

`compute.json` already embeds matrix values, local weights, λmax / CI / CR, missing pairs, repairs — that covers “the matrix and the weights and the numbers.”

### `summary.json`

Small porcelain for `ahp journal list` without parsing full `compute.json`:

```json
{
  "complete": true,
  "consistent": true,
  "top": [
    {"rank": 1, "id": "vendor_a", "name": "Acme", "weight": 0.4123},
    {"rank": 2, "id": "vendor_b", "name": "Beta", "weight": 0.3310}
  ],
  "worst_cr": {"matrix": "alt:cost", "cr": 0.087}
}
```

### `index.jsonl`

Append one line per run (path, timestamp, command, complete/consistent, leader). Enables fast listing without scanning every folder.

---

## CLI surface (proposed)

```text
ahp journal list [--limit N] [--json]
ahp journal show <id|latest> [--json]
ahp journal path <id|latest>          # print directory path
ahp journal diff <idA> <idB>          # ranking + CR delta (v1 text/json)
ahp journal prune [--keep N]          # enforce retention
ahp journal snapshot                  # checkpoint data (+ optional compute)
ahp journal clear                     # delete tmp/journal (confirm / --yes)
```

Examples:

```bash
ahp compute                 # also writes tmp/journal/<id>/
ahp journal list
ahp journal show latest
ahp journal diff 20260918T120000Z_compute_abc 20260918T180000Z_compute_def
```

Exit / print: after compute, optionally one line:

```text
journal: tmp/journal/20260918T232215Z_compute_a1b2c3
```

(quiet when non-TTY or `--no-journal`).

---

## MCP

Mirror workspace functions (no forked logic):

| Tool | Maps to |
|------|---------|
| `journal_list` | `Workspace.JournalList` |
| `journal_show` | `Workspace.JournalShow` |
| `journal_diff` | `Workspace.JournalDiff` (later OK) |

Compute / write-outputs path journals automatically when enabled; tools return `journal_id` / `journal_path` in the JSON payload.

Catalog: add entries under `internal/catalog` + `ahp docs journal`.

---

## Implementation sketch

1. **`internal/workspace/journal.go`**
   - `JournalDir()`, `WriteJournalEntry(meta, result, opts)`
   - Copy data files; write results via existing CSV/JSON helpers used by `WriteOutputs`
   - Append `index.jsonl`; prune by `keep`
2. **Hook** `WriteOutputs` (or callers of it) so every successful output write can journal once.
3. **`cmd/ahp/journal.go`** — Cobra subcommands.
4. **`.gitignore`** — add `tmp/` (and/or `**/tmp/journal/`).
5. **`EnsureLayout`** — do **not** create `tmp/` until first journal write (lazy).
6. **Tests** — temp workspace: compute twice → two entries; prune keeps N; disabled flag writes nothing; `compute.json` in journal matches `output/compute.json`.

Package boundary: journal I/O stays in `workspace`; engine unchanged.

---

## Design choices / trade-offs

| Choice | Recommendation | Rationale |
|--------|----------------|-----------|
| Where | `tmp/journal/` under workspace | Local, disposable, next to the decision |
| Trigger | After `WriteOutputs` | Numbers exist; one hook covers compute/render/open |
| HTML in journal | Off by default | Size; HTML is rebuildable from `compute.json` |
| Dedup identical runs | Optional v1.1 | Same `data_hash` + same include flag → skip or soft-link |
| Restore | Explicit later: `ahp journal restore <id>` copies `data/` back | Dangerous; needs confirm + `--yes` |
| vs git | Journal is for **run artifacts**, not replacing VCS | Git tracks intentional commits; journal tracks every compute |

**Rejected alternative:** overwrite a single `tmp/last-run/`. That loses history, which is the point of a journal.

**Rejected alternative:** store only under OS `/tmp`. Harder for agents to find; dies across reboots; not tied to the workspace.

---

## Phased delivery

### Phase A — MVP

- [ ] `tmp/journal/<id>/{meta.json,data/,results/,summary.json}` + `index.jsonl`
- [ ] Hook from `WriteOutputs`
- [ ] `ahp journal list|show|path|prune`
- [ ] gitignore `tmp/`
- [ ] Tests + one line on compute success
- [ ] MCP: return journal path from compute; `journal_list` / `journal_show`

### Phase B — Compare & restore

- [ ] `ahp journal diff` (ranking delta, CR per matrix)
- [ ] `ahp journal snapshot` without requiring compute
- [ ] `ahp journal restore <id>` (copies `data/` only; then user runs compute)
- [ ] Dedup / `--force` re-snapshot

### Phase C — Niceties

- [ ] Embed “previous run” link in `report.html`
- [ ] `ahp status` shows latest journal id
- [ ] Compression (`results/compute.json.gz`) if keep is large

---

## Success criteria

1. Two successive `ahp compute` runs with a pairwise edit between them leave **two** journal folders with different rankings.
2. `results/weights.csv` and `results/ranking.csv` in a journal entry match what `output/` held at that moment.
3. Disabling journal (`--no-journal` / config) leaves behavior identical to today.
4. `go test ./...` green; MCP compute payload includes journal path when enabled.
5. Docs: short section in README / QUICKSTART pointing at `ahp journal list`.

---

## Open questions

1. **Default on or off?** Safer ship: default **off** until Phase A is dogfooded, then default **on** with `keep = 50`.
2. **Should `include-proposals` runs be tagged / separated** from committed-only journals in `list` filters?
3. **Restore scope:** data-only vs data+output?
4. **Retention unit:** count of runs vs age (e.g. 7 days)?

---

## Suggested first PR

Implement Phase A only: journal writer + list/show/path/prune + gitignore + tests + compute hook. No restore, no diff UI.
