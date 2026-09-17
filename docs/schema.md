# File schema

## `ahp.toml`

```toml
[decision]
title = "Choose a vendor"
description = "Optional longer prompt / context"
```

## `data/criteria.csv`

| column | notes |
|--------|--------|
| id | Stable slug used in pairwise matrices |
| name | Display name |
| parent_id | Empty for root criteria; set for sub-criteria |
| description | Free text |

## `data/alternatives.csv`

| column | notes |
|--------|--------|
| id | Stable slug |
| name | Display name |
| description | Free text |

## `data/pairwise.csv`

| column | notes |
|--------|--------|
| matrix | `criteria`, `criteria:<parent_id>`, or `alt:<criterion_id>` |
| left, right | Node ids (one row per unordered pair) |
| value | `1`–`9` or `1/n` |
| status | `proposal` (agent) or `committed` (counts in compute) |
| note | Why this intensity |

## `data/attributes.csv`

Objective facts keyed by alternative × criterion. Never enter the eigenvector directly
(use `ahp rate` to propose Saaty pairs).

## `data/constraints.csv`

Hard eligibility bands for numeric attributes (optional):

| column | notes |
|--------|--------|
| criterion_id | e.g. `value` |
| min, max | Inclusive band (empty = unbound) |
| unit | Required attribute unit (e.g. `BRL`) |
| prefer | `higher` \| `lower` — used by purchase integrity |
| note | Free text |

Set with `ahp constrain value --min 200 --max 400 --unit BRL --prefer lower`.
Out-of-band alternatives are **excluded from synthesis** and flagged by `ahp doctor`.

## Generated `output/`

- `weights.csv` — local weights + CR per matrix
- `ranking.csv` — global alternative order
- `compute.json` — full payload
- `report.html` — method + ranking + contribution breakdown + matrices + CSV dump + repairs
- `sensitivity.json` — written by `ahp sensitivity` (tornado / rank-reversal)

## Attribute → proposals (opt-in)

`ahp rate --criterion <id> --prefer higher|lower` (alias: `ahp pair suggest-from-attributes`)
reads numeric `attributes.csv` rows and writes **proposal** Saaty pairs on `alt:<criterion>`.
Never auto-commits. Default skips committed pairs; `--refresh` demotes them back to proposals
so price updates can re-enter `plan` / `apply`.

## Purchase integrity

`ahp doctor --purchase` (also auto-runs when `constraints.csv` has rows) checks local source,
unit, FX/foreign smells, budget band, and that `alt:<criterion>` direction matches attributes.
