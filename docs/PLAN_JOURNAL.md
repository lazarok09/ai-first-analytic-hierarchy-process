# Plan: Workspace journal (per-run CSV + results snapshots)

**Status:** Phase A implemented (default off)  
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

## Layout

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
        results/
          weights.csv
          ranking.csv
          compute.json
          report.html          # optional / configurable
          sensitivity.json     # if present at write time
        summary.json    # small ranking + CR peek for list UIs
```

**Run id:** `YYYYMMDDTHHMMSSZ_<command>_<short-hash>` (first 8 hex of data content hash).

---

## When to write

Write after a successful `Workspace.WriteOutputs` (`ahp compute` / `render` / `open`, MCP `compute`).

### Flags / config

```toml
[journal]
enabled = true          # default false in Phase A
dir = "tmp/journal"
keep = 50
include_html = false
include_sensitivity = true
```

- `--journal` / `--no-journal` on compute-family commands
- `AHP_JOURNAL=0` kill-switch; `AHP_JOURNAL=1` enables when config is off

---

## CLI (Phase A)

```text
ahp journal list [--limit N] [--json]
ahp journal show <id|latest> [--json]
ahp journal path <id|latest>
ahp journal prune [--keep N]
```

```bash
ahp compute --journal
ahp journal list
ahp journal show latest
```

---

## MCP

| Tool | Maps to |
|------|---------|
| `journal_list` | `Workspace.JournalList` |
| `journal_show` | `Workspace.JournalShow` |
| `compute` + `journal` | returns `journal_id` / `journal_path` when enabled |

---

## Phased delivery

### Phase A — MVP

- [x] `tmp/journal/<id>/{meta.json,data/,results/,summary.json}` + `index.jsonl`
- [x] Hook from `WriteOutputs`
- [x] `ahp journal list|show|path|prune`
- [x] gitignore `tmp/`
- [x] Tests + one line on compute success
- [x] MCP: return journal path from compute; `journal_list` / `journal_show`

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

## Open questions (Phase A answers)

1. **Default on or off?** **Off** until dogfooded; enable with `--journal`, `AHP_JOURNAL=1`, or `[journal] enabled = true`.
2. **`include-proposals`:** tagged in `meta.include_proposals`; list filters deferred.
3. **Restore scope:** deferred to Phase B (lean data-only).
4. **Retention:** count of runs (`keep = 50`).
