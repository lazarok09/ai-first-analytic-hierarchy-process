# Contributing

Thanks for helping improve `ahp` — a local Go CLI + MCP server for Analytic Hierarchy Process workspaces.

## Development

Requirements: **Go 1.25.5+** (see `go.mod`).

```bash
git clone https://github.com/lazarok09/ahp-method.git
cd ahp-method
go test ./...
go build -o bin/ahp ./cmd/ahp
./bin/ahp compute -w examples/vendor-selection
```

## Layout

| Path | Role |
|------|------|
| `cmd/ahp` | CLI entrypoints |
| `internal/engine` | AHP math (eigenvector, CR, repairs) |
| `internal/workspace` | CSV / TOML I/O and orchestration |
| `internal/mcp` | MCP tools — must call the same workspace functions as the CLI |
| `internal/render` | HTML report |
| `examples/vendor-selection` | Demo workspace |

**Do not fork logic** between CLI and MCP. Add behavior in `internal/workspace` (or `engine`), then expose it from both surfaces.

## Product invariants

- Agents write pairwise judgments with `status=proposal` only (`propose_pairwise` / `ahp set-pairwise --status proposal`).
- Humans commit judgments in CSV or via `ahp commit-proposals`.
- Prefer `ahp status`, `ahp doctor`, and `suggest_repairs` over inventing consistency fixes by hand.
- See [AGENTS.md](./AGENTS.md), [docs/VISION.md](./docs/VISION.md), and [docs/architecture.md](./docs/architecture.md).

## Pull requests

1. Keep changes focused; update tests for engine/workspace behavior.
2. Run `go test ./...` and `go vet ./...` locally.
3. Smoke: `go build -o ahp ./cmd/ahp && ./ahp compute -w examples/vendor-selection`.
4. Document user-facing CLI/MCP changes in `CHANGELOG.md` under Unreleased.

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).
