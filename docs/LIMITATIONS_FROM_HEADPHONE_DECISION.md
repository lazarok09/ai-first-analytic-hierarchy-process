# CLI limitations from a real decision

**Exercise:** pick a Brazilian studio headphone for use with an audio interface, budget **R$200–400**.  
**Workspace:** `workspaces/br-headphones-audio-iface`  
**Tool:** `ahp` 0.3.0 (`go install` / local `bin/ahp`)  
**Date:** 2026-09-16

This note is not a product wishlist. It lists **gaps that forced work or math outside `ahp`**, framed as **What / Which / Why**, plus **how to address** (or a qualitative discussion when the fix is not a single verb).

---

## Decision outcome (for context)

**Price rule (corrected):** Pix/boleto à vista from Brazilian music shops or `br.*` official sites only — **no USD→BRL conversion**, no US Amazon, no marketplace 3P outliers.

| Alt | Ref. price (BRL) | Source |
|-----|------------------|--------|
| Superlux HD681 | **289** | Cheda's SP Pix (R$288.90) |
| AKG K414P | **315** | King Musical boleto (R$314.99) |
| AKG K52 | **338** | Serenata BH à vista |
| Samson SR850 | **369** | King Musical boleto (R$368.99) |

**Dropped:** Shure SRH240A — Magalu/Buscapé ~**R$505** (out of R$200–400). Earlier ~R$359 KaBuM 3P was discarded.

After recompute with BR prices + K414P replacing Shure:

| Rank | Alternative | Global weight |
|------|-------------|----------------|
| 1 | **AKG K52** (R$338) | 0.375 |
| 2 | Samson SR850 (R$369) | 0.229 |
| 3 | AKG K414P (R$315) | 0.200 |
| 4 | Superlux HD681 (R$289) | 0.196 |

Criteria weights (committed): isolation **0.41**, accuracy **0.26**, value **0.19**, comfort **0.11**, drive **0.03**. All matrices CR ≤ 0.10.

**Interpretation:** with isolation prioritized for mic tracking, the closed over-ear K52 leads clearly. Samson stays #2 on accuracy despite semi-open leak. K414P and HD681 are nearly tied for 3rd. Prices and stock move; treat as method demo, not purchasing advice.

---

## Work done *outside* the CLI

| Outside work | Why it was outside |
|--------------|--------------------|
| Market research (Amazon BR, Kabum, AKG BR, Megasom, reviews) | No discovery / shortlist / constraint filter |
| Budget hard filter (drop ATH-M20x at ~R$499+) | Attributes are display-only; no eligibility rules |
| Manual Saaty conversion from BRL / closed vs semi-open / ohms | No attribute→priority / absolute measurement path |
| Shell loop for 40 `pair set` calls | No batch / import / propose-from-attributes |
| Qualitative comfort & accuracy judgments from reviews | No structured rating scale tied to compute |
| Mental sensitivity (“if isolation softens, Samson wins”) | No `explain` / `sensitivity` |
| This limitations write-up | Expected; agents still need a durable gap log |

What *did* work well: `init` → `status`/`next` → structure verbs → `pair set` (proposal) → `plan` → `apply` → `compute` → `doctor` / HTML report. Orientation verbs and plan/apply staging are already useful.

---

## Limitations (What / Which / Why + how to address)

### L1 — Attributes never enter the ranking math

- **What:** `attributes.csv` / `ahp set-attribute` store facts (price, ohms, closed/semi-open) but **do not influence weights**. Ranking only comes from Saaty pairwise matrices.
- **Which:** Affects every decision with measurable criteria. In this run, prices 293 / 359 / 387 / 399 and design types were typed into attributes, then **re-judged by hand** into `alt:value` and `alt:isolation`.
- **Why:** Vision intentionally keeps objective data as human-informing context (avoid silent AI weight faking). That is sound — but without an *opt-in* bridge, agents reinvent ratio→Saaty math in chat/shell, which is exactly the inconsistency risk AHP was meant to prevent.
- **How to address:** Phase C `ahp rate` / attribute ratings (absolute measurement), opt-in only; optionally `ahp pair suggest-from-attributes --criterion value --scale saaty` that writes **proposals** with notes citing attribute values. Never auto-commit.

### L2 — No hard constraints or shortlist filtering

- **What:** Budget “R$200–400” and “must work with audio interface” lived in the goal text and agent brain, not in the model.
- **Which:** ATH-M20x (often ~R$499 in BR) was excluded manually; Behringer options under R$200 were skipped manually. Nothing in `doctor`/`status` would fail if an out-of-band alternative were added.
- **Why:** AHP assumes the alternative set is already valid. Real shopping starts with constraints.
- **How to address:** Lightweight eligibility: `ahp constrain value --min 200 --max 400 --unit BRL` reading attributes, or `must_pass` flags on alternatives. Out-of-range alts show in `doctor` as warnings and are excluded from synthesis until overridden. Qualitative discussion: keep constraints on attributes (objective), not on pairwise (subjective).

### L3 — Pairwise entry does not scale (N(N−1)/2 per matrix)

- **What:** Five criteria + four alternatives → **40 judgments**, each a separate `ahp pair set …` invocation (or CSV edit).
- **Which:** Agent workflow required a bash helper; humans would need `pair ask` (TTY) or spreadsheet import.
- **Why:** Classic AHP cost. CLI today is correct but tedious; friction pushes people to invent incomplete matrices or equal weights.
- **How to address:** `ahp pair import judgments.csv`; `ahp pair set --from-json`; MCP bulk propose; interactive `pair ask` for humans (exists) + agent-friendly batch propose. Longer term: incomplete solvers (Harker/RGMM) so sparse judgments still compute with honest incompleteness flags.

### L4 — Proposal-vs-committed view confuses “missing”

- **What:** After writing 40 **proposals**, `ahp pair missing` (default) still listed all 40 pairs as missing. Completeness only appeared with `--include-proposals` (or after `apply`).
- **Which:** Agents that follow `Next: ahp pair missing` blindly may re-propose or think the workspace is empty of judgments.
- **Why:** Committed-only completeness is a good human-ownership rule, but the default “missing” verb does not explain that proposals already cover the graph.
- **How to address:** `pair missing` should report `{missing_committed, covered_by_proposals, uncovered}` by default; `status` without `--include-proposals` should still say “40 proposals fill all gaps — run `ahp plan`” instead of looking incomplete. Align `next` with that message.

### L5 — Empty / incomplete workspaces report misleading readiness fields

- **What:** Fresh `ahp init` with **0 criteria / 0 alternatives** returned `complete: true` (and `consistent: true`) in JSON status, while `exit_code` was 2 and `doctor` correctly errored.
- **Which:** Any agent that branches on `complete` alone will skip structure steps.
- **Why:** Completeness is defined on pairwise matrices; with no matrices to fill, “vacuously complete” is mathematically easy and operationally wrong.
- **How to address:** Define readiness: `complete` requires ≥1 criterion, ≥2 alternatives (or ≥1), and filled required matrices. Mirror `doctor` structure errors into `status.complete=false`.

### L6 — Equal-weight fallback ranks look like real results

- **What:** Before judgments, ranking showed four alternatives at **0.25** each. `plan` compared against that flat baseline, producing large Δrank noise.
- **Which:** Agents/humans glancing at `status` ranking peek may treat ties as “already solved.”
- **Why:** Incomplete matrices currently fall back to equal weights (documented roadmap debt).
- **How to address:** Hide ranking peek when incomplete; or label `ranking_mode=equal_fallback`; Phase C `--incomplete=harker|rgmm|none` with `none` refusing to rank.

### L7 — No explain / sensitivity (critical when top scores are close)

- **What:** Commands `ahp explain` and `ahp sensitivity` do not exist yet (Phase B in ROADMAP). Top three weights: 0.296 / 0.285 / 0.274.
- **Which:** Outside the tool we reasoned: “isolation dominates → closed-backs; if use-case is mix-only, Samson likely flips to #1.” That is core decision quality, done in prose.
- **Why:** AHP without contribution breakdown and weight perturbation is a black box for close races.
- **How to address:** Ship Phase B: `ahp explain [alt]` (local × criterion weight contributions); `ahp sensitivity --delta 0.05` (tornado / rank-reversal thresholds). Emit `output/sensitivity.json` and a report section. This is high leverage relative to new pairwise UX.

### L8 — No structured bridge from qualitative evidence

- **What:** Comfort and monitoring accuracy came from reviews and reputation, not numbers. Notes on pairs help humans, but there is no evidence object the engine can cite.
- **Which:** Risk of untraceable agent judgment, especially on `alt:accuracy` and `alt:comfort`.
- **Why:** CSV notes are free text; no schema for sources, scores, or confidence.
- **How to address:** Optional `evidence.csv` or richer attribute types (`ordinal:1-5`, `source_url`). `pair set` could require `--note` when `--as proposal` and matrix is subjective. Qualitative: keep Saaty human-owned; improve audit trail, not auto-scoring of reviews.

### L9 — No alternative discovery / catalog helpers

- **What:** Choosing which four SKUs to compare was 100% external search.
- **Which:** Entire “Brazilian market in band” problem is pre-AHP.
- **Why:** Out of scope for a local math CLI (correct boundary), but agents still need a documented handoff: “AHP starts after shortlist.”
- **How to address (qualitative):** Do **not** scrape shops into the core binary. Document a workflow: shortlist → `add-alternative` → attributes → rate/pair. Optional later: `ahp import` from a user-supplied CSV shortlist template (`id,name,price_brl,design,…`).

### L10 — Driveability criterion was nearly dead weight

- **What:** All candidates were ~32–38 Ω and easy for interface amps; criterion weight landed at **0.034**. Pairwise still required a full 6-pair matrix.
- **Which:** Extra agent work and CR surface for almost no discrimination.
- **Why:** Hierarchy does not support “park this criterion” or “skip alt matrix if σ≈0.”
- **How to address:** `ahp criterion park drive` (exclude from synthesis); or auto-hint in `doctor` when attribute variance is zero: “alt:drive judgments add little — consider removing criterion.” Qualitative: teach users to drop non-discriminating criteria early via `tree`/`doctor`.

### L11 — Agent JSON noise and apply verbosity

- **What:** Non-TTY defaults to JSON (good). `ahp apply -y` dumped all 40 committed rows into stdout — hard to skim in agent logs.
- **Which:** Slows review loops; obscures the ranking delta that `plan` already summarized well.
- **How to address:** `apply` default summary `{committed:40, ranking_after:[…], cr_ok:true}` with `--verbose` for full rows. Same for bulk `pair set`.

### L12 — Install / workspace hygiene for real use

- **What:** Install is fine (`go install` / `go build -o bin/ahp`). Decision workspace under `workspaces/` is useful but not mentioned in README quickstart; `output/` is gitignored globally so reports vanish from VCS (usually good).
- **Which:** Reproducing this exercise needs an explicit path and maybe a sample under `examples/`.
- **How to address:** Add `examples/br-headphones` (sanitized) or document `workspaces/` as local-only; keep outputs ignored.

---

## Priority map (what to build next from this exercise)

| Priority | Limitation | Fits roadmap | Status |
|----------|------------|--------------|--------|
| P0 | L5 empty `complete=true`; L4 missing-vs-proposals messaging | Phase A polish | **Done** (2026-09-16): structure-aware `complete`, `proposals_fill_gaps` / coverage fields, `next`→`plan` |
| P0 | L6 hide/label equal-weight ranking | Phase A / C incomplete flags | **Done**: `ranking_mode=equal_fallback\|ahp`; human status hides peek |
| P1 | L7 explain + sensitivity | Phase B | **Done** (2026-09-16): `ahp explain`, `ahp sensitivity` (+ MCP / report section / `output/sensitivity.json`) |
| P1 | L1 attribute→proposal / `rate` | Phase C | **Done** (partial): `ahp rate` / `pair suggest-from-attributes` → proposals only; absolute-measurement synthesis still open |
| P2 | L3 batch import / bulk propose | Phase A stretch / D import | Open |
| P2 | L2 constraints from attributes | Phase B checklist / doctor | Open |
| P3 | L8 evidence trail; L10 park criterion; L11 apply quiet | UX polish | Open |
| Out of core | L9 market discovery | Docs + CSV template only | Open |

---

## 3Whys summary table

| ID | What (gap) | Which (impact here) | Why (root) | Address |
|----|------------|---------------------|------------|---------|
| L1 | Attributes ≠ priorities | Manual price/isolation Saaty | Design: facts are advisory | Opt-in `rate` / suggest-from-attributes → proposals |
| L2 | No constraints | Manual budget filter | AHP assumes valid set | Attribute min/max eligibility in doctor/synthesis |
| L3 | One pair per command | 40 shell calls | No bulk API | import / bulk propose / incomplete solvers |
| L4 | Missing ignores proposals | False “still missing” | Committed-only default without explanation | Structured missing report + next hints |
| L5 | Vacuous complete | Empty WS `complete:true` | Matrix-only completeness | Structure-aware readiness |
| L6 | Equal-weight fake ranks | 0.25×4 before data | Incomplete fallback | Hide or flag; `--incomplete=` |
| L7 | No sensitivity/explain | 2pt race undecidable in-tool | Phase B not shipped | `explain` + `sensitivity` |
| L8 | Weak evidence model | Soft criteria from reviews | Notes only | evidence/ordinal attributes |
| L9 | No discovery | All shortlisting external | Correct product boundary | Docs + import template |
| L10 | Non-discriminating criteria | Full `alt:drive` matrix | No park/skip | park criterion + doctor hint |
| L11 | Noisy apply JSON | 40-row dump | Verbose default | Summary + `--verbose` |
| L12 | Example/docs gap | Hard to replay | Hygiene | example workspace or docs |

---

## Method note

Judgments were agent **proposals**, then `ahp apply -y` for this lab so compute reflected the full matrix. In production agent use, stop at `plan` and let a human `apply`. That workflow is already one of the CLI’s strengths; the gaps above are mostly **pre-judgment structure**, **attribute linkage**, and **post-ranking decision quality**.
