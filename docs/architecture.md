# Architecture

```
Agent (Cursor / Claude / …)
    │  MCP stdio: `ahp mcp`
    ▼
ahp (Go)  ── reads/writes ──►  workspace/
    │                           ahp.toml
    │                           data/*.csv
    ├── engine (eigenvector, CR, repairs, synthesis)
    └── render ──────────────►  output/report.html
                                output/weights.csv
                                output/ranking.csv
```

Humans can skip MCP and use the CLI or edit CSV directly.

## Stack

| Layer | Choice |
|-------|--------|
| Language | Go |
| CLI | Cobra (`ahp`) |
| Math | Power-method principal eigenvector |
| HTML | Self-contained report (`internal/render`) |
| MCP | `mark3labs/mcp-go`, **stdio** |
| State | TOML + CSV on disk |

## Package layout

```
cmd/ahp/                 CLI entrypoint
internal/engine/         Saaty math
internal/workspace/      CSV/TOML I/O + compute orchestration
internal/render/         HTML report
internal/mcp/            MCP tool surface
examples/vendor-selection/
docs/
```

## Conventions

1. MCP tools wrap the same workspace functions as the CLI — do not fork logic.
2. `propose_pairwise` always writes `status=proposal`.
3. Compute ignores proposals unless `--include-proposals` / `include_proposals`.
4. Run `go test ./...` after engine changes.
