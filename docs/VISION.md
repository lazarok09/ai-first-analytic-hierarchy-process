# Product vision — AHP as a local Go CLI

Cognitive AHP without a hosted web app: a Go CLI you install on a machine, MCP tools for agents, CSV for weights and data, HTML so people can inspect the method.

## Problem

1. AHP math is easy to fake in chat or spreadsheets.
2. Hosted “AI decision apps” outsource judgment.
3. Agents need a real installable tool that mutates files and shows a report.

## Solution

- **`ahp` Go binary**: init workspace, compute eigenvector + CR, render HTML.
- **MCP stdio** (`ahp mcp`): the same operations as tools.
- **CSV + TOML** as durable state. AI fills data; humans own commits.
- **`output/report.html`**: method explainer, matrices, weights, ranking, repairs.
- **Proposals vs committed.** Agents write `status=proposal`.

## Design principles

| Principle | Meaning |
|-----------|---------|
| Human owns judgment | MCP `propose_pairwise` cannot commit |
| Files over servers | No Auth, no app DB, no ChatKit |
| Show the method | HTML is a canvas of AHP |
| CR is core | Incomplete or CR > 0.10 is first-class |
| Objective data is data | `attributes.csv` informs people |

## Success

- [x] `go build` / `go install` yields `ahp`
- [x] Agent can init, propose pairs, compute, point human at `report.html`
- [x] Human can edit CSV, recompute, see CR + ranking change
- [x] Example vendor-selection workspace demos consistently
- [x] CR repair hints + `ahp status` / `ahp open`
- [x] Implementation in Go (CLI framework for AI-supplied data)
