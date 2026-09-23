# Plan: Strict AHP method (Santos AHP-Gaussian + classical Saaty)

**Status:** implemented (opt-in comparative path; Saaty default unchanged)  
**Date:** 2026-09-23  
**Paper (math source):** Marcos dos Santos, Igor Pinheiro de Araújo Costa, Carlos Francisco Simões Gomes — *Multicriteria Decision-Making in the Selection of Warships…*, IJAHP 13(1), 2021. [DOI](https://doi.org/10.13033/ijahp.v13i1.833) · [PDF](https://ijahp.org/index.php/IJAHP/article/download/833/748)  
**Related:** [ROADMAP.md](./ROADMAP.md) Phase C, [VISION.md](./VISION.md), [schema.md](./schema.md), [architecture.md](./architecture.md), `.cursor/skills/ahp-data-integrity/`  
**Reference code:** R [AHPGaussian](https://github.com/cidedson/ahpgaussian) (algorithm parity; not a product domain)

---

## Product framing (read this first)

`ahp` is a **generic** decision CLI. Users rank vendors, headphones, apartments, cameras, jobs — any goal → criteria → alternatives workspace. The Santos paper is a **math and procedure reference**, not a domain we ship features for.

| Role of the paper | Role of the CLI |
|-------------------|-----------------|
| Absolute measurement (decision matrix → sum-norm) | Works on any numeric `attributes.csv` |
| Gaussian factor \(f=\sigma/\mu\) criterion reweight | Opt-in comparative ranking for any shortlist |
| Warship Tables 9–17 | **Optional** engine fixture for numeric tolerance only |

**Non-goal:** a warship product mode, naval templates, or UX copy about ships. One regression fixture may reuse paper numbers under a neutral name (e.g. `testdata/gaussian_paper/`); it must not appear in QUICKSTART as “how to use ahp.”

---

## Verdict

| Layer | Aligned with Santos math? | Notes |
|-------|---------------------------|--------|
| **CLI / engine** | Yes (opt-in) | Absolute + AHP-Gaussian implemented; Saaty remains default |
| **`ahp-data-integrity`** | Orthogonal | Purchase/price hygiene only |
| **ROADMAP** | Linked | Gaussian = comparative; Saaty canonical |

Do **not** silently replace Saaty. Modes: **`saaty` (default)** + **`absolute` / `gaussian` / `hybrid` / `compare` (opt-in)**.

---

## Generic problem shape (any subject)

```text
Goal (ahp.toml)
  └─ Criteria (criteria.csv) — may have parent_id hierarchy
       └─ Alternatives (alternatives.csv)
            └─ Objective facts (attributes.csv: alt × criterion → number)
            └─ Optional bands (constraints.csv: prefer higher|lower, min/max, unit)
            └─ Optional Saaty judgments (pairwise.csv)
```

**Absolute / Gaussian eligibility (domain-agnostic):**

A leaf criterion \(c\) is **attribute-ready** when every *included* alternative has a parseable numeric attribute for \(c\), and \(c\) has an explicit **direction** (`prefer=higher|lower` from `constraints.csv` or a future criteria field).

Examples of the *same* math on different subjects:

| Subject | Benefit (higher) | Cost (lower) |
|---------|------------------|--------------|
| Headphones | guest_score, isolation_db | price_brl, weight_g |
| Apartment | area_m2, bedrooms | rent, commute_min |
| Vendor | sla_score, features | annual_cost |
| Paper warship (fixture only) | range, endurance | cost, build_years |

Qualitative-only criteria (feel, brand trust) stay on the **Saaty** path (`alt:<c>` pairwise). Mixed workspaces are normal: absolute/Gaussian on numeric leaves; Saaty elsewhere (or full Saaty default).

---

## CLI implementation plan (primary deliverable)

### End-to-end user/agent flow (any workspace)

```text
1. ahp init / structure          # goal, criteria, alternatives
2. ahp set-attribute …           # fill decision matrix (facts)
3. ahp constrain <c> --prefer …  # direction (+ optional band)
4. [purchase?] ahp-data-integrity + doctor --purchase
5. Choose mode:
     saaty     → pair fill → doctor → compute → report
     absolute  → sum-norm locals × expert criteria w (needs criteria matrix)
     gaussian  → sum-norm + σ/μ weights (no criteria pairwise required)
     hybrid    → absolute with expert w, then optional gaussian compare
     compare   → side-by-side outputs; Saaty still canonical if complete
6. ahp doctor --method           # mode-specific gates
7. ahp compute --method=… / ahp gaussian / ahp absolute
8. Defend ranking only if checklist in ahp-method-strict passes
```

### Suggested verbs / flags

Keep catalog-style names; MCP mirrors CLI with no forked math.

| Surface | Behavior |
|---------|----------|
| `ahp compute --method=saaty` | Today’s default (unchanged) |
| `ahp compute --method=gaussian` | Attribute matrix → invert costs → sum-norm → \(f=\sigma/\mu\) → ranking + factor table |
| `ahp compute --method=hybrid` | Same matrix × **committed criteria** priority vector (eigenvector + CR) |
| `ahp compute --method=compare` | Emit Saaty (if ready) and/or hybrid + gaussian; never claim CR for gaussian-only |
| `ahp gaussian` | Thin alias of gaussian compute + human factor table |
| `ahp absolute` | Preview sum-norm local priorities / hybrid score without writing pairwise |
| `ahp rate --mode=saaty` | Today’s lossy ratio→Saaty **proposals** (label as lossy) |
| `ahp rate --mode=sum-norm` | Write or preview absolute locals (no Saaty 1–9); never auto-commit subjective pairs |
| `ahp doctor --method` | Incomplete decision matrix, missing prefer, cost≤0, mode mismatch |
| `ahp sensitivity` | **Unchanged** — tornado ±δ; different from Gaussian |

Outputs (additive, domain-neutral):

- `output/gaussian.json` — \(\mu_c,\sigma_c,f_c,w_c^{\mathrm{G}}\), scores, ranking  
- `output/absolute.json` — normalized decision matrix + hybrid scores when requested  
- Report HTML section: “Absolute / Gaussian (comparative)” with **dispersion ≠ importance** caveat  

### Data contracts (reuse existing CSV; extend carefully)

| Need | Prefer using | Avoid |
|------|--------------|--------|
| Direction higher\|lower | `constraints.csv` `prefer` (already exists) | Hard-coding criterion ids like `value` |
| Eligibility bands | same file `min`/`max`/`must_have` | Separate “gaussian.toml” domain packs |
| Facts | `attributes.csv` | Embedding paper ship columns in schema |
| Expert criteria weights | committed `pairwise.csv` matrix `criteria` | Shipping published warship \(w\) as defaults |

**Schema tweak (if needed):** allow `prefer` without requiring min/max (direction-only constrain). Document in `schema.md`. Optional later: `criteria.csv` column `direction` — only if constrain proves awkward for non-purchase domains.

### Engine placement

| Package | Responsibility |
|---------|----------------|
| `internal/engine` | Pure functions: build matrix, invert, sum-norm, gaussian factors, score. **No** workspace I/O, **no** domain types |
| `internal/workspace` | Load attrs + constraints + criteria weights; call engine; write outputs |
| `cmd/ahp` | Flags, human/JSON printing |
| `internal/mcp` | Same workspace entrypoints |

Pseudocode (generic):

```text
alts = included alternatives (after constraints filter)
leaves = leaf criteria that are attribute-ready
for c in leaves:
  prefer = Prefer(c)  # higher|lower — required
  for a in alts: raw[a,c] = ParseFloat(attr[a,c])
  col = InvertIfLower(raw[:,c], prefer)  # error if lower and v<=0
  norm[:,c] = col / sum(col)
if method == gaussian:
  w[c] = normalize(sd(norm[:,c]) / mean(norm[:,c]))  # sample SD default
elif method == hybrid:
  w = EigenvectorWeights(criteria matrix)  # existing engine; CR gated
score[a] = sum_c w[c] * norm[a,c]
```

Hierarchy rule: run absolute/Gaussian on **leaf** criteria with full attributes; parent criteria still use Saaty aggregation of children (or require flat leaves-only mode — decide in open questions, default = leaves-only).

### Agent skill: `ahp-method-strict`

Domain-agnostic checklist (not purchase-specific):

1. Structure present  
2. Every attribute-ready criterion has `prefer`  
3. Decision matrix complete for chosen mode (or explicit NA policy)  
4. Mode declared  
5. `saaty`: matrices complete, CR ≤ 0.10, proposals resolved  
6. `gaussian`: factor table shown; **no CR claim**  
7. `hybrid`: criteria CR OK; alts from attributes not from vibes pairwise  
8. Purchase-like → also `ahp-data-integrity`  
9. Never defend equal-weight fallback  

Update `AGENTS.md`: **data integrity ≠ method integrity**.

---

## Paper math (generic extraction)

Ignore naval narrative. Keep only transferable procedure.

### Layer A — Absolute + expert criteria (paper §5.1–5.3)

1. Decision matrix of objective values (any units, per criterion).  
2. Per criterion sum-normalize; **min/lower**: \(v'=1/v\) then sum-norm.  
3. Criteria weights from expert Saaty (+ theory CR ≤ 0.10).  
4. \(score_a = \sum_c w_c^{\mathrm{expert}} n_{a,c}\).

Our `hybrid` mode: step 3 uses **our** principal eigenvector (canonical), not the paper’s average-of-normalized-columns. Fixture may embed published weights only to check arithmetic — not as product behavior.

### Layer B — AHP-Gaussian (paper §5.4 / Fig. 2)

On the **normalized** columns:

\[
\mu_c=\mathrm{mean}(n_{\cdot,c}),\ 
\sigma_c=\mathrm{SD}(n_{\cdot,c}),\ 
f_c=\sigma_c/\mu_c,\ 
w_c^{\mathrm{G}}=f_c/\sum f
\]

Then \(score_a=\sum_c w_c^{\mathrm{G}} n_{a,c}\).

Interpretation to put in CLI help/report: higher \(f_c\) ⇒ more **dispersion / discrimination** across the current shortlist — **not** “more important to the decision maker.”

Pure gaussian ⇒ no criteria pairwise (explicit opt-in). Matches paper §6 claim without making it the default story.

### Zeros and SD (product rules)

| Case | Policy |
|------|--------|
| `prefer=higher`, value `0` | Allowed (capability absent → local score 0) |
| `prefer=lower`, value ≤ 0 | Fail closed (no silent `1/0`) |
| Sample vs population SD | Default **sample** (\(n-1\)) to match R `AHPGaussian`; document |

Optional affine stretch remains a **documented extension** for `rate --mode=saaty`, not the paper-faithful absolute default.

### What Gaussian is not

- Not CI/CR  
- Not `ahp sensitivity` tornado  
- Not `ahp rate` Saaty quantization  
- Not a domain pack for ships or anything else  

---

## Alignment (paper math vs CLI today)

| Capability | Paper math | CLI | Notes |
|------------|------------|-----|--------|
| Alt scores from facts | Sum-norm decision matrix | `absolute` / `hybrid` / `gaussian` | Opt-in; Saaty default unchanged |
| Criteria weights | Expert Saaty | Eigenvector + CR | Used by `saaty` + `hybrid` (CR ≤ 0.10 gated) |
| Alt pairwise | Unused in absolute/Gaussian | Required for default compute | Still required for `saaty` only |
| Sensitivity | \(\sigma/\mu\) reweight | ±δ tornado | Both exist; different commands |
| Attr → judgment | Direct weighted sum | `rate --mode=saaty\|sum-norm` | Saaty path stays lossy bridge |

---

## Limits (must show in skill + report)

| Limit | Why it matters for any subject |
|-------|--------------------------------|
| Dispersion ≠ importance | Price may dominate expert \(w\) but vanish under Gaussian if all quotes are close |
| Shortlist-dependent | Add/remove one alternative → new \(w^{\mathrm{G}}\) and possible reorder |
| Compensatory | No veto criteria (use `constrain --must-have` / bands for hard filters) |
| Numeric leaves only | Brand vibe etc. stay Saaty |
| No CR on Gaussian scores | UI must not imply consistency ratio |

---

## Optional paper fixture (not product)

**Purpose:** lock arithmetic against Tables 10 / 15–17 (and R `ahpgaussian`) within tolerance.  
**Location:** `internal/engine/testdata/…` or similar — **tests only**.  
**Naming:** prefer neutral (`gaussian_paper.json`) over `warships` in user-facing paths.  
**Not:** example workspace in QUICKSTART, catalog demos, or MCP instructions.

If hybrid Table 14 parity is desired in tests, embed published \(w^{\mathrm{expert}}\) in the fixture; do not change default eigenvector behavior.

---

## Proposed work (ordered)

| # | Item | Notes |
|---|------|--------|
| 1 | Skill `ahp-method-strict` + `AGENTS.md` | Generic checklist |
| 2 | Engine absolute + gaussian (pure functions) + unit tests | Include optional paper numeric fixture |
| 3 | Workspace wiring + `compute --method` / `ahp gaussian` | Any workspace with attrs + prefer |
| 4 | `doctor --method` + direction-only constrain | Generic gates |
| 5 | `rate --mode=sum-norm` / `ahp absolute` | Honesty vs lossy Saaty |
| 6 | Report + MCP + catalog entries | Comparative section + caveat |
| 7 | ROADMAP Phase C link to this plan | |

---

## Non-goals

- Replacing Saaty as default  
- Auto-committing attribute-derived or Gaussian results  
- Domain templates (ships, phones, …) as product surface  
- Folding FX/price rules into method skill  
- Changing tornado `sensitivity` into Gaussian  
- Shipping bibliometrics / military narrative  

---

## Success criteria

- [x] Any attribute-complete workspace with `prefer` per numeric leaf can run `gaussian` / `hybrid` without domain-specific code  
- [x] Default `ahp compute` remains classical Saaty  
- [x] Doctor/method skill block defending Gaussian scores as if they had Saaty CR  
- [x] Report states dispersion ≠ importance  
- [x] `rate --mode=saaty` stays proposal-only; sum-norm path exists without Saaty quantization  
- [x] Benefit-zero OK; cost-zero fails closed  
- [x] Engine tests include generic cases (e.g. 3 alts × mixed higher/lower) **and** optional paper-number tolerance test  
- [x] MCP mirrors CLI; `go test ./...` green  
- [x] No user-facing “warship mode”; paper appears only in this plan + test comments/citations  
- [x] ROADMAP Phase C row links here

## Paper map (reference only)

| Paper § | Keep for implementers | Skip for product |
|---------|----------------------|------------------|
| §1–2 Intro / lit / bibliometrics | Why compensatory hierarchical AHP | BN force structure, Scopus tables |
| §3 AHP theory | Saaty scale, CR ≤ 0.10 | — |
| §4 Methodology | Experts lock structure + criteria weights | 10 officers, ship models as UX |
| §5.1–5.3 | Decision matrix, sum-norm, expert \(w\), score | Table 8 technical sheet |
| §5.4 | Fig. 2 Gaussian steps | “Warship” labels in UI |
| §6 | Pure Gaussian possible; sharper discrimination | Call to apply only in military settings |

---

## Open questions

1. Sample vs population SD — **decided:** sample (\(n-1\)), R-compatible.
2. Leaves-only Gaussian vs error if non-leaf lacks attributes — **decided:** leaves-only; parents stay Saaty.
3. Direction: constrain-only `prefer` vs `criteria.csv` column — **decided for now:** constrain-only (direction-only rows OK).
4. `compare` default contents when Saaty matrix incomplete — **decided:** gaussian required; absolute soft-fails with warning.
5. Whether `hybrid` should require criteria CR ≤ 0.10 — **decided:** yes; incomplete/hot criteria → matrix-only (or hard fail for `--method=hybrid`).  

---

## References

- Santos, M. dos, Costa, I. P. de A., & Gomes, C. F. S. (2021). IJAHP 13(1). https://doi.org/10.13033/ijahp.v13i1.833  
- Saaty, T. L. (1980). *The Analytic Hierarchy Process*.  
- R AHPGaussian: https://github.com/cidedson/ahpgaussian  
- In-repo: `docs/schema.md`, `internal/engine/from_attributes.go`, `internal/engine/sensitivity.go`, `.cursor/skills/ahp-data-integrity/`
