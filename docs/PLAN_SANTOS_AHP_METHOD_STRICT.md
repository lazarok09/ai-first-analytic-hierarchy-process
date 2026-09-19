# Plan: Strict AHP method (Santos AHP-Gaussian + classical Saaty)

**Status:** proposal (not implemented)  
**Date:** 2026-09-18  
**Paper:** Marcos dos Santos, Igor Pinheiro de Araújo Costa, Carlos Francisco Simões Gomes — *Multicriteria Decision-Making in the Selection of Warships: A New Approach to the AHP Method*, IJAHP 13(1), 2021.  
**DOI:** [10.13033/ijahp.v13i1.833](https://doi.org/10.13033/ijahp.v13i1.833)  
**Related:** [ROADMAP.md](./ROADMAP.md) Phase C (“Optional Gaussian view”), [VISION.md](./VISION.md), skill `.cursor/skills/ahp-data-integrity/`

---

## Verdict

| Layer | Aligned with Santos 2021? | Notes |
|-------|---------------------------|--------|
| **CLI / engine** | Partial (classical Saaty only) | Full pairwise + CR; not AHP-Gaussian |
| **`ahp-data-integrity` skill** | No (orthogonal) | Purchase/price hygiene, not method procedure |
| **ROADMAP** | Intentional gap | Gaussian listed as comparative, Saaty canonical |

The CLI is classical Saaty AHP. The skill is data integrity. Neither implements the paper’s **AHP-Gaussian** path. Do **not** silently replace Saaty — dual-mode: **Saaty (canonical)** + **Gaussian (comparative / absolute-measurement)**.

---

## What the paper does

Two layers:

1. **Classical-ish AHP (paper §5.1–5.3)**  
   - Experts define alternatives and criteria.  
   - **Alternatives:** decision matrix of objective values → **sum-normalize** per criterion (cost/min inverted so cheaper scores higher). **No** alternative pairwise Saaty.  
   - **Criteria:** expert **Saaty pairwise** + consistency (CR ≤ 0.1).  
   - Ranking = normalized decision matrix × criteria priority vector.

2. **“New approach” / sensitivity (paper §5.4) — AHP-Gaussian**  
   - On each criterion’s normalized column: mean \(\mu\), std \(\sigma\), **Gaussian factor** \(f_c = \sigma_c / \mu_c\).  
   - Renormalize factors → new criterion weights.  
   - Re-rank; discrimination increases (e.g. Model 3 from ~0.39 → ~0.51).  
   - Conclusion: ordering possible **without** pairwise on alternatives (and, in the pure Gaussian path, without expert criteria pairwise).

Reference implementation: R package [AHPGaussian](https://github.com/cidedson/ahpgaussian) (`min → 1/value`, sum-norm, `sd/mean`, score = Σ factor × norm).

---

## Alignment detail (paper vs `ahp` today)

| Step | Santos et al. | Our CLI today |
|------|---------------|---------------|
| Structure | Goal → criteria → alternatives | `init` / criteria / alternatives |
| Alt scores from facts | Decision matrix → **sum-normalize** | Facts in `attributes.csv`; ranking from **Saaty pairwise** |
| Criteria weights | Expert Saaty + CR ≤ 0.1 | Same (criteria matrix + CR) |
| Alt pairwise | **Not used** | Required (`alt:<c>` matrices) |
| Sensitivity | Gaussian \(f_c = \sigma/\mu\) reweight | `ahp sensitivity` = ±δ **tornado** |
| Attribute → judgment | Direct weighted sum of norms | `ahp rate` = ratio → **nearest Saaty 1–9** (lossy) |

**Stricter than the paper on one point:** we demand CR on **every** matrix (including alts). The paper CR-focused the **criteria** matrix and skipped alt pairwise.

**Skill:** correctly hardens inputs (local price, band, provenance, direction vs `alt:value`). It does **not** encode hierarchy completeness, decision-matrix fill, min/max semantics, CR gates, or Gaussian sensitivity.

---

## Goals

1. Make method procedure explicit and agent-enforceable (separate from price integrity).  
2. Implement true AHP-Gaussian as an **opt-in, comparative** path.  
3. Stop treating `ahp rate` (lossy Saaty quantization) as equivalent to the paper’s absolute measurement.  
4. Harden the classical Saaty “ready to decide” gate without changing default ranking math.

---

## Proposed work

### 1. New skill: `ahp-method-strict`

**Path:** `.cursor/skills/ahp-method-strict/SKILL.md`  
**Role:** method procedure (not purchase prices).  
**Relationship:** run **after** `ahp-data-integrity` for purchase-like decisions; alone for non-purchase.

Checklist before defending a ranking:

1. Goal + criteria + alternatives declared  
2. Every numeric criterion has `prefer` / `min|max` locked (`constrain` or schema)  
3. Decision matrix complete (all alt×criterion attributes numeric, or explicit NA policy)  
4. Mode chosen: `saaty` | `gaussian` | `both`  
5. If `saaty`: all matrices complete, CR ≤ 0.10, proposals applied or rejected  
6. If `gaussian`: report Gaussian ranking + factor table; **do not** claim Saaty CR for that ranking  
7. Purchase decisions still run `ahp-data-integrity` first  

Also update `AGENTS.md`: **data integrity ≠ method integrity**.

### 2. Schema / CLI: objective criteria first-class

- Persist per criterion: `direction=higher|lower` (paper min/max) — extend `constrain` / schema as needed.  
- **Doctor gate:** if a criterion has numeric attributes for all alts, require direction before `rate` / `gaussian`.  
- Optional: `ahp doctor --method` fails when the decision matrix is incomplete for the selected mode.

### 3. Engine: AHP-Gaussian (Phase C, opt-in)

```text
ahp compute --method=saaty|gaussian|compare
# or
ahp gaussian [--write-report-section]
```

Pipeline (match paper + R package):

1. Build decision matrix from `attributes.csv`  
2. Cost/min: \(v' = 1/v\) (document zeros — paper AAW=0 case; may need affine shift like `rate`)  
3. Sum-normalize per criterion  
4. \(f_c = \sigma / \mu\); \(w_c = f_c / \sum f\)  
5. \(\mathrm{score}_a = \sum_c w_c \cdot n_{c,a}\); emit ranking + factor table  

**Invariant:** Saaty ranking stays canonical; Gaussian is comparative only (same as ROADMAP).

**Tests:** fixture from paper Tables 9–10 / 15–17 (warship numbers); `go test ./...`.

**MCP:** mirror CLI (`compute` flags or `gaussian` tool); no forked math.

### 4. Stricten `ahp rate` (honesty vs paper)

| Change | Why |
|--------|-----|
| Add `--mode=sum-norm` (or `ahp absolute`) writing **local priorities** without Saaty quantization | Matches paper §5.1 |
| Keep `--mode=saaty` as today’s bridge; label **lossy / proposal-only** | Honesty |
| Skill: forbid defending ranking that used `rate` without `plan`/`apply` + CR | Procedure |
| When attributes cover all alts, `doctor --method` prefer sum-norm or Gaussian over vibes Saaty | Close headphone L1 gap |

### 5. Stricten classical Saaty “ready” gate

- `doctor --strict` / `status --ready`: incomplete → exit 2, CR > 0.10 → 3, proposals → 4  
- Keep ranking peek hidden under equal-weight fallback (hard rule in method skill)  
- Optional hybrid: expert criteria weights vs Gaussian weights, show both rankings (paper §5.2 + §5.4)

### 6. Docs / report

- Method section: cite Santos 2021 for Gaussian; Saaty 1980 for pairwise  
- Dual-mode report: side-by-side ranking + factor table when `--method=compare`  
- Point ROADMAP Phase C item at this plan

---

## Priority

| Order | Item | Risk | Value |
|-------|------|------|-------|
| 1 | Skill `ahp-method-strict` + `AGENTS.md` pointer | Low | Immediate agent discipline |
| 2 | `ahp gaussian` / `compute --method=compare` + engine tests | Medium | Real paper fidelity |
| 3 | `rate --mode=sum-norm` (or `ahp absolute`) | Medium | Stop forcing attributes through Saaty 1–9 |
| 4 | `doctor --method` / ready defaults | Low–medium | Procedural hardness |

---

## Non-goals

- Replacing Saaty as the default ranking  
- Auto-committing Gaussian or attribute-derived judgments  
- Dropping CR checks on Saaty matrices  
- Folding price/FX rules into the method skill (keep `ahp-data-integrity`)

---

## Success criteria

- [ ] Agent can run a documented method checklist before defending a ranking  
- [ ] `ahp compute --method=compare` (or equivalent) reproduces paper-style Gaussian ranking on a warship fixture within tolerance  
- [ ] Report clearly separates Saaty vs Gaussian and never claims CR for Gaussian-only scores  
- [ ] `ahp rate --mode=saaty` remains proposal-only; sum-norm path available without Saaty quantization  
- [ ] `go test ./...` green; MCP mirrors CLI  

---

## References

- Santos, M. dos, Costa, I. P. de A., & Gomes, C. F. S. (2021). Multicriteria decision-making in the selection of warships: a new approach to the AHP method. *International Journal of the Analytic Hierarchy Process*, 13(1). https://doi.org/10.13033/ijahp.v13i1.833  
- Saaty, T. L. (1980). *The Analytic Hierarchy Process*. McGraw-Hill.  
- R package AHPGaussian: https://github.com/cidedson/ahpgaussian  
- In-repo: `docs/ROADMAP.md` Phase C; `internal/engine/from_attributes.go`; `internal/engine/sensitivity.go`; `.cursor/skills/ahp-data-integrity/`
