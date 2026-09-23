# Quickstart — vendor selection demo

End-to-end walkthrough using [`examples/vendor-selection`](../examples/vendor-selection).

## Build

```bash
go build -o bin/ahp ./cmd/ahp
```

Requires Go **1.25.5+**.

## Compute and inspect

```bash
./bin/ahp compute -w examples/vendor-selection
./bin/ahp status -w examples/vendor-selection
./bin/ahp doctor -w examples/vendor-selection
./bin/ahp tree -w examples/vendor-selection
```

Optional run history (off by default):

```bash
./bin/ahp compute -w examples/vendor-selection --journal
./bin/ahp journal list -w examples/vendor-selection
./bin/ahp journal show latest -w examples/vendor-selection
```

Open the HTML report:

```bash
./bin/ahp open -w examples/vendor-selection
# or open examples/vendor-selection/output/report.html in a browser
```

## Agent path (MCP)

Point Cursor (or another MCP client) at the binary:

```json
{
  "mcpServers": {
    "ahp": {
      "command": "/absolute/path/to/ahp",
      "args": ["mcp"],
      "env": {
        "AHP_WORKSPACE": "/absolute/path/to/examples/vendor-selection"
      }
    }
  }
}
```

Agents should call `propose_pairwise` (always `status=proposal`). Humans commit with `ahp commit-proposals` or by editing `data/pairwise.csv`.

## Next reading

- [schema.md](./schema.md) — CSV / TOML formats
- [mcp.md](./mcp.md) — tool list
- [architecture.md](./architecture.md) — CLI ↔ workspace ↔ MCP
