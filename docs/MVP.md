# MVP

1. `go build -o bin/ahp ./cmd/ahp` (or `go install`) yields `ahp`
2. `ahp init` + CSV/MCP fill
3. `ahp compute` writes report + weight/ranking CSVs
4. Incomplete matrices and CR > 0.10 visible with repair hints
5. `ahp mcp` stdio for Cursor
6. Example: `examples/vendor-selection`
7. `ahp status` / `ahp open` / `ahp commit-proposals`

## Agent rules

- Follow [architecture.md](./architecture.md).
- Do not add a web app, Auth, or HTTP MCP unless asked.
- Pairwise from agents is `proposal` unless the user asked to commit.
- After math changes, keep `go test ./...` green.

## Post-MVP

- Sensitivity / robustness plots in the report
- Group AHP / multiple judges
- Optional `ahp serve` HTTP viewer
