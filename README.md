# ahp

**Local Analytic Hierarchy Process for humans and agents.**

One Go binary. CSV is the source of truth. Real AHP math — principal eigenvector, consistency ratio, repair hints, hierarchical synthesis. Agents propose judgments; humans commit. An HTML report shows the method, not just a ranking.

[![CI](https://github.com/lazarok09/ai-first-analytic-hierarchy-process/actions/workflows/ci.yml/badge.svg)](https://github.com/lazarok09/ai-first-analytic-hierarchy-process/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25.5+-00ADD8?logo=go&logoColor=white)](./go.mod)
[![Release](https://img.shields.io/github/v/release/lazarok09/ai-first-analytic-hierarchy-process?include_prereleases&label=release)](https://github.com/lazarok09/ai-first-analytic-hierarchy-process/releases)

```bash
go install github.com/lazarok09/ahp-method/cmd/ahp@latest
ahp compute -w examples/vendor-selection
ahp open    -w examples/vendor-selection   # → output/report.html
```

---

## Why this exists

AHP is easy to fake in a chat window or a spreadsheet. Hosted “AI decision” apps hide the math and own your judgment.

`ahp` is the opposite:

| Principle | What you get |
|-----------|----------------|
| **Files over servers** | Workspace = `ahp.toml` + CSVs. No account, no cloud DB. |
| **Humans own judgment** | Agents write `status=proposal`. Commit is explicit (`ahp apply` / CSV edit). |
| **Show the method** | HTML report with matrices, weights, CR, repairs, explain. |
| **CR is first-class** | Incomplete or CR > 0.10 is diagnosed, not papered over. |
| **Attributes ≠ scores** | Prices and ohms inform people; they enter ranking only via opt-in `ahp rate`. |

Built for Cursor / Claude-style agents via MCP — same functions as the CLI, no forked logic.

---

## Install

Requires **Go 1.25.5+**.

```bash
go install github.com/lazarok09/ahp-method/cmd/ahp@latest
ahp version
```

From source:

```bash
git clone https://github.com/lazarok09/ai-first-analytic-hierarchy-process.git
cd ai-first-analytic-hierarchy-process
go build -o bin/ahp ./cmd/ahp
./bin/ahp version
```

Shell completions: `ahp completion bash|zsh|fish`.

---

## 30-second demo

```bash
go build -o bin/ahp ./cmd/ahp
./bin/ahp compute -w examples/vendor-selection
./bin/ahp status  -w examples/vendor-selection
./bin/ahp explain -w examples/vendor-selection
./bin/ahp open    -w examples/vendor-selection
```

You get ranking + CR, a contribution breakdown, and `examples/vendor-selection/output/report.html`.

---

## How it works

```
┌─────────────┐     propose      ┌──────────────────┐
│  Agent/MCP  │ ───────────────► │  pairwise.csv    │
│  or human   │   status=proposal│  (durable state) │
└─────────────┘                  └────────┬─────────┘
                                          │ ahp apply
                                          ▼
                                 status=committed
                                          │
                                          ▼
                              ┌───────────────────────┐
                              │ ahp compute           │
                              │ eigenvector · CR      │
                              │ synthesis · report    │
                              └───────────────────────┘
```

**Workspace layout**

```
my-decision/
  ahp.toml
  data/
    criteria.csv
    alternatives.csv
    pairwise.csv       # Saaty 1–9 or 1/n · proposal|committed
    attributes.csv     # objective facts (unit, source, note)
    constraints.csv    # optional eligibility bands (min/max/unit)
  output/              # generated — do not hand-edit
    weights.csv
    ranking.csv
    compute.json
    report.html
    sensitivity.json   # from ahp sensitivity
```

**Matrix keys** in `pairwise.csv`:

- `criteria` — root criteria
- `criteria:<parent_id>` — sub-criteria under a parent
- `alt:<criterion_id>` — alternatives under a leaf criterion

---

## Everyday workflow

```bash
# 1. Structure
ahp init ./laptop-buy --title "Choose a laptop"
cd ./laptop-buy
ahp add-criterion battery "Battery life"
ahp add-criterion value "Price / value"
ahp add-alternative framework "Framework 13"
ahp add-alternative thinkpad "ThinkPad T14"

# 2. Objective facts (provenance required for purchase-like criteria)
ahp set-attribute framework value 8990 --unit BRL --source "loja oficial Pix"
ahp set-attribute thinkpad  value 7490 --unit BRL --source "loja oficial Pix"
ahp constrain value --min 5000 --max 10000 --unit BRL --prefer lower

# 3. Judgments — agents default to proposals
ahp pair set criteria battery value 3 --as proposal --note "battery slightly over price"
ahp rate --criterion value --prefer lower          # attributes → Saaty proposals
# prices changed later?
ahp rate --criterion value --prefer lower --refresh

# 4. Preview → commit → compute
ahp plan
ahp apply -y
ahp doctor --purchase
ahp compute
ahp status
ahp open
```

**Orientation verbs** (start here when lost):

| Command | Role |
|---------|------|
| `ahp status` | Completeness, CR, proposals, ranking peek, next step |
| `ahp next` | Single best follow-up command |
| `ahp doctor` | Schema, gaps, CR hotspots; `--purchase` for price integrity |
| `ahp tree` | Goal → criteria → alt matrices |
| `ahp plan` | Ranking if proposals were committed (no writes) |
| `ahp explain` | Why an alternative won (criterion contributions) |
| `ahp sensitivity` | ±δ weight tornado / rank-reversal |

Full catalog: `ahp catalog` · `ahp docs <command>`.

---

## MCP (agents)

Point Cursor (or any MCP client) at the same binary:

```json
{
  "mcpServers": {
    "ahp": {
      "command": "ahp",
      "args": ["mcp"],
      "env": {
        "AHP_WORKSPACE": "/absolute/path/to/workspace"
      }
    }
  }
}
```

See [`docs/mcp-cursor.example.json`](./docs/mcp-cursor.example.json) and [`docs/mcp.md`](./docs/mcp.md).

**Contract:** `propose_pairwise` / `suggest_from_attributes` always write **proposals**. Never silently commit subjective judgments. Use `plan` → human `apply` (or `commit_proposals` with explicit intent).

---

## What “good AHP” means here

- Reciprocal Saaty matrices (1–9 and reciprocals)
- Principal eigenvector local weights
- Consistency index / ratio; **CR ≤ 0.10** accepted, hotspots + repair hints above
- Hierarchical synthesis → global ranking
- Incomplete matrices: equal-weight fallback labeled `ranking_mode=equal_fallback` (not a fake solved decision)
- Opt-in attribute → Saaty bridge (`ahp rate`) — never auto-commits
- Hard eligibility bands (`ahp constrain`) exclude out-of-band alternatives from synthesis
- Purchase integrity (`ahp doctor --purchase`): local currency/source, FX smell, pairwise direction vs attributes

---

## Project layout

```
cmd/ahp/           CLI entrypoints
internal/
  engine/          AHP math (eigenvector, CR, synthesis, rate mapping)
  workspace/       CSV/TOML I/O, doctor, plan/apply, constraints
  mcp/             stdio MCP — wraps workspace (no forked logic)
  render/          HTML report
  catalog/         command-as-doc registry (CLI ↔ MCP)
examples/
  vendor-selection/
docs/              vision, architecture, schema, roadmap
```

---

## Docs

| Doc | Contents |
|-----|----------|
| [QUICKSTART](./docs/QUICKSTART.md) | Vendor-selection walkthrough |
| [VISION](./docs/VISION.md) | Product principles |
| [architecture](./docs/architecture.md) | CLI ↔ workspace ↔ MCP |
| [schema](./docs/schema.md) | TOML + CSV contracts |
| [mcp](./docs/mcp.md) | Tool list |
| [ROADMAP](./docs/ROADMAP.md) | What’s next |
| [LIMITATIONS…](./docs/LIMITATIONS_FROM_HEADPHONE_DECISION.md) | Gaps found in a real BR purchase dogfood |
| [CONTRIBUTING](./CONTRIBUTING.md) | Build, test, PRs |
| [CHANGELOG](./CHANGELOG.md) | Releases |
| [SECURITY](./SECURITY.md) | Vulnerability reporting |

---

## Develop

```bash
go test ./...
go vet ./...
go build -o bin/ahp ./cmd/ahp
./bin/ahp compute -w examples/vendor-selection
```

PRs welcome. Keep MCP tools wrapping `internal/workspace` — do not fork domain logic into the MCP layer.

---

## License

[MIT](./LICENSE) © 2026 Lazaro Souza
