# MCP

Stdio server: `ahp mcp` (Go). No HTTP, no API keys.

## Cursor example

Copy [mcp-cursor.example.json](./mcp-cursor.example.json) into Cursor MCP settings.

Set `AHP_WORKSPACE` to an absolute path, or pass `workspace` on every tool call.

## Tools

| Tool | Writes | Notes |
|------|--------|--------|
| `init_workspace` | ahp.toml + empty CSVs | First call |
| `set_goal` | ahp.toml | |
| `upsert_criterion` | criteria.csv | |
| `upsert_alternative` | alternatives.csv | |
| `upsert_attribute` | attributes.csv | Facts only |
| `propose_pairwise` | pairwise.csv | **Always** `proposal` |
| `commit_pairwise` | pairwise.csv | Prefer human CSV edits |
| `commit_proposals` | pairwise.csv | Bulk flip |
| `get_state` | — | Ranking, CR, missing, repairs |
| `workspace_status` | — | Compact counts |
| `missing_pairs` | — | Gaps to propose |
| `suggest_repairs` | — | CR > 0.10 pairs |
| `compute` | output/* | Math + HTML |
| `render_report` | output/* | Alias of compute |

## Agent loop

1. `init_workspace` or detect `ahp.toml`
2. Upsert criteria, alternatives, attributes
3. `missing_pairs` → `propose_pairwise`
4. Stop for human commit unless explicitly asked
5. `compute` → open `output/report.html`
6. If inconsistent → `suggest_repairs` → propose revisions
