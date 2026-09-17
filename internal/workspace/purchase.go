package workspace

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// FX / foreign listing smells. Brazilian "R$123" is intentionally not matched
// (RE2 has no lookbehind/lookahead).
var (
	fxSmellRE  = regexp.MustCompile(`(?i)(usd|\beur\b|\bus\$|converted|convers[aã]o|sweetwater|thomann)`)
	amazonComRE = regexp.MustCompile(`(?i)amazon\.com`)
	dollarRE   = regexp.MustCompile(`(?i)(?:≈|~)?\s*\$\s*\d`)
	market3PRE = regexp.MustCompile(`(?i)(marketplace|3p|terceiro|\bseller\b|fulfilled by )`)
	reaisRE    = regexp.MustCompile(`(?i)R\$`)
)

func hasFXSmell(blob string) bool {
	if fxSmellRE.MatchString(blob) {
		return true
	}
	if amazonComRE.MatchString(blob) && !strings.Contains(strings.ToLower(blob), "amazon.com.br") {
		return true
	}
	// Strip Brazilian reais markers so "$59" still matches but "R$359" does not.
	stripped := reaisRE.ReplaceAllString(blob, " ")
	return dollarRE.MatchString(stripped)
}

// PurchaseIntegrityFindings audits attribute provenance and alt:<criterion>
// direction vs objective attributes for purchase-like criteria.
// When criterionFilter is empty, every constraint with Prefer set is audited;
// if no constraints exist, criterion "value" with prefer "lower" is used when
// that criterion has attributes.
func (w *Workspace) PurchaseIntegrityFindings() ([]Finding, error) {
	constraints, err := w.Constraints()
	if err != nil {
		return nil, err
	}
	attrs, err := w.Attributes()
	if err != nil {
		return nil, err
	}
	alts, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	pairs, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	altSet := map[string]bool{}
	for _, a := range alts {
		altSet[a.ID] = true
	}

	type auditSpec struct {
		Criterion string
		Unit      string
		Prefer    string
		Min, Max  *float64
	}
	var specs []auditSpec
	for _, c := range constraints {
		prefer := c.Prefer
		if prefer == "" {
			prefer = "lower"
		}
		specs = append(specs, auditSpec{
			Criterion: c.CriterionID, Unit: c.Unit, Prefer: prefer, Min: c.Min, Max: c.Max,
		})
	}
	if len(specs) == 0 {
		// Fallback: if value attributes exist, audit them as lower-better purchase facts.
		hasValue := false
		for _, a := range attrs {
			if a.CriterionID == "value" {
				hasValue = true
				break
			}
		}
		if hasValue {
			specs = append(specs, auditSpec{Criterion: "value", Prefer: "lower"})
		}
	}

	var findings []Finding
	for _, spec := range specs {
		findings = append(findings, auditPurchaseCriterion(spec.Criterion, spec.Unit, spec.Prefer, spec.Min, spec.Max, attrs, pairs, altSet)...)
	}
	return findings, nil
}

func auditPurchaseCriterion(
	criterion, unit, prefer string,
	vmin, vmax *float64,
	attrs []AttributeRow,
	pairs []PairwiseRow,
	altSet map[string]bool,
) []Finding {
	prices := map[string]float64{}
	var findings []Finding

	for _, row := range attrs {
		if row.CriterionID != criterion {
			continue
		}
		alt := row.AlternativeID
		blob := row.Source + " " + row.Note + " " + row.Value
		if !altSet[alt] {
			findings = append(findings, Finding{
				Code: "unknown_alt", Severity: "error",
				Message: fmt.Sprintf("attribute %s/%s: alternative not in alternatives.csv", alt, criterion),
			})
			continue
		}
		u := strings.TrimSpace(row.Unit)
		if u == "" {
			findings = append(findings, Finding{
				Code: "missing_unit", Severity: "error",
				Message: fmt.Sprintf("%s/%s: missing unit", alt, criterion),
				Fix:     "ahp set-attribute " + alt + " " + criterion + " <n> --unit <UNIT> --source <shop>",
			})
		} else if unit != "" && !strings.EqualFold(u, unit) {
			findings = append(findings, Finding{
				Code: "unit_mismatch", Severity: "error",
				Message: fmt.Sprintf("%s/%s: unit %q != expected %q", alt, criterion, u, unit),
			})
		}
		if strings.TrimSpace(row.Source) == "" {
			findings = append(findings, Finding{
				Code: "missing_source", Severity: "error",
				Message: fmt.Sprintf("%s/%s: missing source (name a local shop/page)", alt, criterion),
				Fix:     "ahp set-attribute … --source \"King Musical boleto\"",
			})
		}
		if hasFXSmell(blob) {
			findings = append(findings, Finding{
				Code: "fx_or_foreign_source", Severity: "error",
				Message: fmt.Sprintf("%s/%s: foreign/FX smell in source/note — use local street price only", alt, criterion),
			})
		}
		if market3PRE.MatchString(blob) {
			findings = append(findings, Finding{
				Code: "marketplace_3p", Severity: "warn",
				Message: fmt.Sprintf("%s/%s: marketplace/3P wording — prefer specialty/official local quote", alt, criterion),
			})
		}
		num, err := engine.ParseNumericAttribute(row.Value)
		if err != nil {
			findings = append(findings, Finding{
				Code: "non_numeric", Severity: "error",
				Message: fmt.Sprintf("%s/%s: value %q is not numeric", alt, criterion, row.Value),
			})
			continue
		}
		prices[alt] = num
		if vmin != nil && num < *vmin {
			findings = append(findings, Finding{
				Code: "below_budget", Severity: "error",
				Message: fmt.Sprintf("%s: %.4g < min %.4g — remove from shortlist or widen band", alt, num, *vmin),
				Fix:     "ahp constrain " + criterion + " --min/--max  # or drop alternative",
			})
		}
		if vmax != nil && num > *vmax {
			findings = append(findings, Finding{
				Code: "above_budget", Severity: "error",
				Message: fmt.Sprintf("%s: %.4g > max %.4g — remove from shortlist (do not rank)", alt, num, *vmax),
				Fix:     "ahp constrain " + criterion + " --min/--max  # or drop alternative",
			})
		}
	}

	for alt := range altSet {
		if _, ok := prices[alt]; !ok {
			// Only require price when this alt appears among attributes for other criteria
			// or when any alt has this criterion — keep parity with audit script: every alt.
			findings = append(findings, Finding{
				Code: "missing_price_attr", Severity: "error",
				Message: fmt.Sprintf("%s: no numeric attribute for criterion %q", alt, criterion),
				Fix:     fmt.Sprintf("ahp set-attribute %s %s <n> --unit %s --source <shop>", alt, criterion, orDefault(unit, "BRL")),
			})
		}
	}

	matrix := "alt:" + criterion
	for _, row := range pairs {
		if row.Matrix != matrix {
			continue
		}
		lv, lok := prices[row.Left]
		rv, rok := prices[row.Right]
		if !lok || !rok {
			continue
		}
		expect := preferWinner(lv, rv, prefer)
		got := pairWinner(row.Value)
		if expect != "tie" && got != "tie" && expect != got {
			findings = append(findings, Finding{
				Code: "pairwise_contradicts_attrs", Severity: "error",
				Message: fmt.Sprintf("%s %s/%s=%s: pair prefers %s, attributes prefer %s (%s: %s=%.4g, %s=%.4g)",
					matrix, row.Left, row.Right, engine.FormatSaaty(row.Value), got, expect, prefer, row.Left, lv, row.Right, rv),
				Fix:    fmt.Sprintf("ahp rate --criterion %s --prefer %s --refresh", criterion, prefer),
				Matrix: matrix, Left: row.Left, Right: row.Right,
			})
		} else if expect != "tie" && got == "tie" {
			findings = append(findings, Finding{
				Code: "pairwise_flat_vs_attrs", Severity: "warn",
				Message: fmt.Sprintf("%s %s/%s=1 but attributes differ (%s=%.4g, %s=%.4g)",
					matrix, row.Left, row.Right, row.Left, lv, row.Right, rv),
				Matrix: matrix, Left: row.Left, Right: row.Right,
			})
		}
	}
	return findings
}

func preferWinner(left, right float64, prefer string) string {
	if math.Abs(left-right) < 1e-9 {
		return "tie"
	}
	if prefer == "lower" {
		if left < right {
			return "left"
		}
		return "right"
	}
	if left > right {
		return "left"
	}
	return "right"
}

func pairWinner(value float64) string {
	if math.Abs(value-1) < 1e-9 {
		return "tie"
	}
	if value > 1 {
		return "left"
	}
	return "right"
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
