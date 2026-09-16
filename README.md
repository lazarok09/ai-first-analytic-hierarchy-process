# AHP method — Go CLI + MCP

Local Analytic Hierarchy Process: a single Go binary an agent can install, CSV as the source of truth, real AHP math (eigenvector + consistency ratio + repair hints), and a self-contained HTML report so humans can see the method.

Agents do not pick the winner. They fill **proposal** judgments; humans commit in CSV (or via `ahp commit-proposals`).

## Install

Go 1.22+:

```bash
go install github.com/lazarok/ahp-method/cmd/ahp@latest
# or from this repo:
go build -o bin/ahp ./cmd/ahp
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
ahp validate
ahp open
ahp mcp          # stdio MCP for Cursor / Claude
```

Example:

```bash
./bin/ahp compute -w examples/vendor-selection
# open examples/vendor-selection/output/report.html
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

See [docs/README.md](docs/README.md).
