# Plan: README for daily use

**Status:** shipped. `README.md` matches the draft below.  
**Date:** 2026-10-16  
**Related:** [QUICKSTART.md](./QUICKSTART.md), [mcp.md](./mcp.md), [VISION.md](./VISION.md)

---

## Problem

`README.md` opens as a product pitch: agents, MCP, the principal eigenvector, a principles table, then an architecture diagram. The path a person actually needs — name a decision, compare the options, open the report — shows up halfway down, mixed with purchase-integrity flags.

Someone choosing a laptop or a vendor this week should see that path first.

## Goals

1. The first screen says what you do and how to install.
2. One complete decision fits in a single sitting, with commands that already exist.
3. Buying something (prices, sources, prefer-lower) is a second recipe, not the introduction.
4. Agents are a short closing section that points at `docs/mcp.md`.

### Non-goals

- New CLI behavior.
- Editing `internal/render` or the workspace report in the same change.
- Moving method notes out of `docs/architecture.md` and `docs/VISION.md`. Those stay the deep docs.
- Inventing commands. Every snippet below is already in the current README.

## Voice

- Say what you do, then the command.
- One idea per sentence.
- Name pairwise comparison and consistency when they matter. Skip slogans.
- Keep the three GIFs only with plain captions.

## Shape

1. What it is.
2. Install.
3. A decision in one sitting (the laptop example, without proposals, `ahp rate`, or budget bands).
4. What the report is showing.
5. Buying something (short; link `docs/QUICKSTART.md` for the long walk).
6. If you use an agent.
7. Go further (`ahp status`, `ahp explain`, `ahp sensitivity`, schema, contributing).

Drop from the front page: the ASCII pipeline, the “good AHP means” checklist, and the principles table. They already live in `docs/architecture.md` and `docs/VISION.md`.

## Draft

Copy this over `README.md` only after review. Badge URLs and GIF paths match the current file.

````markdown
# ahp

You name a goal, the things that matter, and the options. You say which criterion matters more, and which option wins on each one. `ahp` ranks them and writes an HTML report on your machine. The files stay in a folder you own.

```bash
go install github.com/lazarok09/ahp-method/cmd/ahp@latest
ahp version
```

Needs Go 1.25.5+.

## A decision in one sitting

```bash
ahp init ./laptop-buy --title "Choose a laptop"
cd ./laptop-buy
ahp add-criterion battery "Battery life"
ahp add-criterion value "Price"
ahp add-alternative framework "Framework 13"
ahp add-alternative thinkpad "ThinkPad T14"

ahp pair set criteria battery value 3 --note "battery matters a bit more than price"
ahp compute
ahp open
```

`ahp open` writes `output/report.html`. The ranking is in that file and in `output/ranking.csv`.

Fill the other comparisons the same way. `alt:battery` is “which laptop is better on battery.” `criteria` is “which criterion matters more.” `ahp status` tells you what is still missing.

## What the report is showing

The pictures are the committed `examples/vendor-selection` workspace, the same numbers `ahp compute` writes.

<table>
  <tr>
    <td align="center" width="33%">
      <img src="docs/assets/hierarchy.gif" width="100%" alt="Goal Choose a vendor splits into Cost 0.64, Quality 0.26, and Risk 0.10, then Acme, Initech, and Globex">
      <br><b>Structure</b>
      <br><sub>Goal, how much each criterion weighs, one shortlist</sub>
    </td>
    <td align="center" width="33%">
      <img src="docs/assets/pairwise.gif" width="100%" alt="Cost matters 3 times Quality and 5 times Risk. Consistency ratio 0.03.">
      <br><b>Comparisons</b>
      <br><sub>How much more one side matters. CR 0.03</sub>
    </td>
    <td align="center" width="33%">
      <img src="docs/assets/ranking.gif" width="100%" alt="Ranking: Acme 0.41, Initech 0.33, Globex 0.26">
      <br><b>Ranking</b>
      <br><sub>One score per option</sub>
    </td>
  </tr>
</table>

A consistency ratio above 0.10 means two comparisons disagree. `ahp status` points at which one to fix. The ranking is still computed, and the report says the matrix is inconsistent.

Try the example:

```bash
ahp compute -w examples/vendor-selection
ahp open    -w examples/vendor-selection
```

## Buying something

Record the price and where it came from, then let the tool turn those numbers into comparisons. You still accept them.

```bash
ahp set-attribute framework value 8990 --unit BRL --source "loja oficial Pix"
ahp set-attribute thinkpad  value 7490 --unit BRL --source "loja oficial Pix"
ahp constrain value --min 5000 --max 10000 --unit BRL --prefer lower
ahp rate --criterion value --prefer lower
ahp doctor --purchase
ahp compute
```

The longer walk is [QUICKSTART](./docs/QUICKSTART.md).

## If you use an agent

The same binary speaks MCP. The agent writes comparisons as drafts (`status=proposal`). You run `ahp apply` when you agree. Details: [mcp.md](./docs/mcp.md).

## Go further

| Command | What it answers |
|---------|-----------------|
| `ahp status` | What is filled in, and what to do next |
| `ahp explain` | Why the winner scored what it scored |
| `ahp sensitivity` | Whether a small weight change flips the rank |
| `ahp docs <command>` | Help for one command |

Build from this repo with `go build -o bin/ahp ./cmd/ahp`. Tests: `go test ./...`.

[Schema](./docs/schema.md) · [Architecture](./docs/architecture.md) · [Contributing](./CONTRIBUTING.md) · [License](./LICENSE)
````

## Done when

- `README.md` matches the accepted draft.
- Every command in the new README still exists (`ahp catalog` or `ahp docs <command>`).
- MCP setup JSON is not on the front page. `docs/mcp.md` still has it.
