---
name: ahp-data-integrity
description: >-
  Verify AHP workspace data integrity for shortlists, prices, and pairwise
  comparisons before ranking. Enforces local-market price sources (no foreign
  currency conversion), budget band eligibility, attribute provenance, and
  directional consistency between attributes.csv and alt:value judgments. Use
  when choosing AHP alternatives, setting prices/attributes, filling pairwise
  value matrices, running ahp compute/plan, or when the user mentions preço,
  budget, shortlist, or data integrity for ahp.
---

# AHP data integrity (prices + comparisons)

Dogfood rule from real BR shopping decisions: **bad prices silently poison AHP**.
Run this skill **before** treating a ranking as defensible.

## Hard rules (user complaints → policy)

1. **Local street price only** — use the decision market’s currency as paid today (Pix/boleto/list in that country). **Never** convert USD/EUR → local and paste as `value`.
2. **Named local source required** — every price attribute needs `--source` naming a real shop/page in that market (e.g. King Musical, Serenata, `br.akg.com`, Cheda's). Empty source = fail.
3. **Reject marketplace 3P outliers** — KaBuM/Amazon marketplace third-party “too cheap” listings are not reference prices. Prefer music/specialty shops or official local storefronts. Cross-check Buscapé/Magalu when in doubt.
4. **Budget is a hard filter** — alternatives outside `--min/--max` must leave the shortlist (or stay only as `note` evidence of exclusion). Do not rank them and “hope value pairs fix it”.
5. **Attributes inform; pairwise must not contradict them** — if A is cheaper than B in attributes, `alt:value` must not say B is better value than A (for prefer-lower price).
6. **Provenance in notes** — record list vs Pix, date, and why a competing listing was discarded.

## When to run

- Before `ahp add-alternative` finalizes a shortlist
- After any `ahp set-attribute` on price/cost/value
- Before `ahp pair set` on `alt:value` (or any cost-like matrix)
- Before `ahp plan` / `apply` / `compute` when the decision is purchase-like
- Anytime the user disputes a price or source

## Workflow

Copy and track:

```
Integrity:
- [ ] 1. Constraints known (currency, min, max, market)
- [ ] 2. Shortlist eligibility (all alts inside band)
- [ ] 3. Attribute provenance (unit + source + note)
- [ ] 4. Script audit clean
- [ ] 5. Pairwise direction vs attributes (rate dry-run / manual)
- [ ] 6. ahp doctor ok
- [ ] 7. Only then compute / defend ranking
```

### 1. Lock constraints

From the user goal (example: Brazil, R$200–400):

| Field | Example |
|-------|---------|
| Market | BR |
| Currency / unit | BRL |
| Min | 200 |
| Max | 400 |
| Price criterion id | `value` (or `cost` / `price`) |
| Prefer | `lower` |

Pass the same numbers to the audit script.

### 2. Shortlist eligibility

For each candidate:

1. Fetch **≥1 local** quote (prefer specialty/official).
2. If only foreign MSRP exists → **do not add** until a local quote exists.
3. If local quote is outside band → exclude; keep a one-line note in the goal/docs, not as a ranked alternative.
4. If two local quotes disagree by a lot (e.g. >25%), keep the **specialty/official** one and note the discarded outlier.

### 3. Write attributes correctly

```bash
ahp set-attribute <alt> value <number> -w <ws> \
  --unit BRL \
  --source "King Musical boleto" \
  --note "R$368.99 boleto (lista 409.99). Amazon BR promo discarded as outlier."
```

Required: numeric `value`, `--unit`, `--source`. Notes should mention payment form (Pix/boleto) when relevant.

### 4. Run the audit script

From repo root (or any cwd; pass workspace path):

```bash
python3 .cursor/skills/ahp-data-integrity/scripts/audit_prices.py \
  --workspace <ws> \
  --criterion value \
  --unit BRL \
  --min 200 \
  --max 400 \
  --prefer lower
```

Exit `0` = pass. Non-zero = fix before ranking. JSON: add `--json`.

Banned patterns (script flags as errors/warnings):

- source/note mentioning USD/EUR/`$` conversion, `amazon.com` (non-`.br`), `converted`, `≈ $`
- missing unit/source
- non-numeric price
- price outside `--min/--max`
- pairwise on `alt:<criterion>` that **disagrees in direction** with attribute order

### 5. Align pairwise with attributes

Prefer CLI bridge when available (rebuild local binary if needed):

```bash
./ahp rate --criterion value --prefer lower --dry-run -w <ws>
# if prices changed after committed pairs:
./ahp rate --criterion value --prefer lower --refresh -w <ws>   # demotes→proposals
./ahp plan -w <ws>
```

Lock the budget band in the workspace (not only in chat):

```bash
./ahp constrain value --min 200 --max 400 --unit BRL --prefer lower -w <ws>
./ahp doctor --purchase -w <ws>
```

If `rate` is missing in the installed binary, rebuild from this repo (`go build -o ahp ./cmd/ahp`) or set pairs manually — still run the audit script / `doctor --purchase` so direction cannot silently invert.

**Never** invent Saaty values from foreign MSRP ratios.

### 6. Doctor + status

```bash
ahp doctor --purchase -w <ws>
ahp status -w <ws>
```

Fix doctor errors first (including constraint_violation / fx_or_foreign_source). Do not defend a ranking while integrity findings remain.

## Severity guide

| Severity | Meaning | Action |
|----------|---------|--------|
| **blocker** | Out of band, missing source/unit, FX conversion smell, pairwise contradicts prices | Remove alt or fix data; **no compute defense** |
| **warn** | Single thin source, large local price spread, promo vs list ambiguity | Second local quote or stronger note |
| **info** | Parked criterion, equal prices | Optional cleanup |

## Output to the user

Report briefly:

1. Constraints used (market, unit, min/max)
2. Blockers / warns (from script + doctor)
3. Shortlist after eligibility
4. Whether `alt:value` matches attribute order
5. Only then: ranking peek

## Anti-patterns

- Pasting US Amazon / Sweetwater / “$59 ≈ R$300” into attributes
- Keeping an over-budget closed-back “because isolation is important”
- Using marketplace 3P as the only price
- Filling `alt:value` from vibes while attributes say the opposite
- Calling the ranking final when the audit script failed

## Extra detail

- Failure examples from the headphone dogfood: see [examples.md](examples.md)
