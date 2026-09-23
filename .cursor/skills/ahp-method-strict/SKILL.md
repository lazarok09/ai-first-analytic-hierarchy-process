---
name: ahp-method-strict
description: >-
  Enforce AHP method procedure before defending a ranking: structure complete,
  mode chosen (saaty|gaussian|hybrid|compare), CR gates for Saaty, attribute
  decision-matrix + prefer for absolute/Gaussian, never claim CR for Gaussian
  scores. Use when running ahp compute/plan/gaussian/absolute, defending a
  ranking, or when the user mentions method integrity, AHP-Gaussian, or absolute
  measurement. Distinct from ahp-data-integrity (prices).
---

# AHP method integrity (procedure)

**Data integrity ≠ method integrity.** For purchase-like decisions run
`.cursor/skills/ahp-data-integrity/SKILL.md` **first**, then this checklist.

Saaty pairwise ranking is **canonical**. Absolute / AHP-Gaussian paths are
**comparative only** (Santos-style sum-norm + optional σ/μ reweight).

## Checklist before defending a ranking

Copy and track:

```
Method:
- [ ] Goal + criteria + alternatives declared
- [ ] Mode: saaty | gaussian | hybrid | compare
- [ ] Numeric leaves have prefer (ahp constrain <c> --prefer higher|lower)
- [ ] Decision matrix complete for absolute/gaussian (all eligible alts × leaves)
- [ ] If saaty: matrices complete, CR ≤ 0.10, proposals applied or rejected
- [ ] If gaussian: factor table reviewed; NO Saaty CR claimed
- [ ] If hybrid: criteria CR ≤ 0.10 + complete; alts from attributes; `--method=hybrid` fails if matrix-only
- [ ] Purchase-like → ahp-data-integrity already passed
- [ ] Never defend equal-weight fallback peek as a decision
```

## Commands

```bash
ahp doctor --method          # attributes + prefer readiness
ahp constrain quality --prefer higher   # direction-only OK
ahp absolute                 # sum-norm (+ hybrid only if criteria CR OK)
ahp gaussian                 # σ/μ weights + ranking → output/gaussian.json
ahp compute --method=compare # Saaty + comparative sections in report
ahp rate --mode=saaty …      # lossy Saaty proposals (label as lossy)
ahp rate --mode=sum-norm …   # sum-norm preview for one criterion
```

## Hard rules

1. **Dispersion ≠ importance** — high Gaussian \(f=\sigma/\mu\) means the shortlist differs a lot on that criterion, not that the DM values it.
2. **No CR on Gaussian/absolute scores** — CR applies only to Saaty pairwise matrices.
3. **Benefit zero OK; cost zero fails** — `prefer=lower` requires positive attributes; `prefer=higher` rejects negatives.
4. **Do not auto-commit** attribute-derived or Gaussian results.
5. **Default `ahp compute` stays Saaty** — comparative methods are opt-in flags/verbs.
6. **Hybrid uses criteria weights only** — incomplete/`alt:*` pairwise does not block hybrid; hot criteria CR does.

## When to run

- Before treating any ranking as final
- When attributes cover all alternatives and Saaty alt-pairwise feels forced
- After `ahp rate` (ensure proposals went through `plan`/`apply` + CR if defending Saaty)
- Anytime the user asks “is this method sound?”
