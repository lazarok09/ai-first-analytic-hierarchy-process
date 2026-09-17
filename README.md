# AHP method — Go CLI + MCP

[![CI](https://github.com/lazarok09/ahp-method/actions/workflows/ci.yml/badge.svg)](https://github.com/lazarok09/ahp-method/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25.5+-00ADD8?logo=go&logoColor=white)](./go.mod)

Local Analytic Hierarchy Process: a single Go binary an agent can install, CSV as the source of truth, real AHP math (eigenvector + consistency ratio + repair hints), and a self-contained HTML report so humans can see the method.

Agents do not pick the winner. They fill **proposal** judgments; humans commit in CSV (or via `ahp commit-proposals`).

## Install

Requires **Go 1.25.5+** (see `go.mod`; `mcp-go` sets the floor).

```bash
go install github.com/lazarok09/ahp-method/cmd/ahp@latest
```

Or from this repo / a release binary:

```bash
go build -o bin/ahp ./cmd/ahp
# or download from GitHub Releases after the first v* tag
```

## 30-second demo

```bash
go build -o bin/ahp ./cmd/ahp
./bin/ahp compute -w examples/vendor-selection
./bin/ahp status -w examples/vendor-selection
# open examples/vendor-selection/output/report.html
```

## Workspace layout

```
my-decision/
  ahp.toml
  data/
    criteria.csv
    alternatives.csv
    pairwise.csv      # Saaty 1–9 or 1/n; status proposal|committed
    attributes.csv    # objective facts (not automatic weights)
  output/             # generated
    weights.csv
    ranking.csv
    compute.json
    report.html
```

`pairwise.csv` `matrix` column:

- `criteria` — root criteria comparisons
- `criteria:<parent_id>` — sub-criteria
- `alt:<criterion_id>` — alternatives under a leaf criterion

## Commands (the function surface)

```bash
ahp init ./my-decision --title "Choose a vendor"
ahp set-goal "Choose a vendor" --description "..."
ahp add-criterion cost "Total cost"
ahp add-alternative acme "Acme"
ahp set-attribute acme cost 180000 --unit USD --source RFP
ahp set-pairwise criteria cost quality 3 --status proposal
ahp commit-proposals
ahp compute
ahp status
ahp doctor
ahp open
ahp mcp          # stdio MCP for Cursor / Claude
```

## MCP (agents)

```json
{
  "mcpServers": {
    "ahp": {
      "command": "ahp",
      "args": ["mcp"],
      "env": { "AHP_WORKSPACE": "/absolute/path/to/workspace" }
    }
  }
}
```

Tools: `init_workspace`, `set_goal`, `upsert_criterion`, `upsert_alternative`, `upsert_attribute`, `propose_pairwise`, `commit_pairwise`, `commit_proposals`, `get_state`, `workspace_status`, `missing_pairs`, `suggest_repairs`, `compute`, `render_report`.

## What “good AHP” means here

- Reciprocal Saaty matrices
- Principal eigenvector local weights
- Consistency index / ratio; CR ≤ 0.10 flagged
- Repair hints when inconsistent
- Hierarchical synthesis to a global ranking
- Incomplete matrices: equal fallback + explicit missing pairs
- Objective CSV data is provenance, not a silent scoring model

## Docs

- [docs/](./docs/README.md) — vision, architecture, schema, MCP, roadmap
- [docs/QUICKSTART.md](./docs/QUICKSTART.md) — vendor-selection walkthrough
- [CONTRIBUTING.md](./CONTRIBUTING.md) — build, test, PR expectations
- [SECURITY.md](./SECURITY.md) — vulnerability reporting
- [CHANGELOG.md](./CHANGELOG.md)

## License

[MIT](./LICENSE) © 2026 Lazaro Souza
