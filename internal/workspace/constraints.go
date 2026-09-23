package workspace

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// Constraint is an attribute eligibility rule (hard filter before ranking).
type Constraint struct {
	CriterionID string   `json:"criterion_id"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Unit        string   `json:"unit"`
	Prefer      string   `json:"prefer,omitempty"` // higher|lower — used by purchase integrity
	// MustHave requires a truthy attribute (1/true/yes/sim or numeric > 0).
	// Used for facility filters like parking without inventing a fake min/max band.
	MustHave bool   `json:"must_have,omitempty"`
	Note     string `json:"note,omitempty"`
}

// ConstraintViolation is one alternative failing a constraint.
type ConstraintViolation struct {
	AlternativeID string  `json:"alternative_id"`
	CriterionID   string  `json:"criterion_id"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	Reason        string  `json:"reason"`
}

func (w *Workspace) Constraints() ([]Constraint, error) {
	rows, err := readCSV(w.dataPath("constraints.csv"))
	if err != nil {
		return nil, err
	}
	var out []Constraint
	for _, r := range rows {
		id := strings.TrimSpace(r["criterion_id"])
		if id == "" {
			continue
		}
		c := Constraint{
			CriterionID: id,
			Unit:        strings.TrimSpace(r["unit"]),
			Prefer:      strings.TrimSpace(r["prefer"]),
			MustHave:    parseTruthyFlag(r["must_have"]),
			Note:        strings.TrimSpace(r["note"]),
		}
		if v, ok, err := parseOptFloat(r["min"]); err != nil {
			return nil, fmt.Errorf("constraints %s min: %w", id, err)
		} else if ok {
			c.Min = &v
		}
		if v, ok, err := parseOptFloat(r["max"]); err != nil {
			return nil, fmt.Errorf("constraints %s max: %w", id, err)
		} else if ok {
			c.Max = &v
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CriterionID < out[j].CriterionID })
	return out, nil
}

func (w *Workspace) WriteConstraints(items []Constraint) error {
	rows := make([]map[string]string, 0, len(items))
	for _, c := range items {
		row := map[string]string{
			"criterion_id": c.CriterionID,
			"unit":         c.Unit,
			"prefer":       c.Prefer,
			"note":         c.Note,
		}
		if c.MustHave {
			row["must_have"] = "true"
		}
		if c.Min != nil {
			row["min"] = strconv.FormatFloat(*c.Min, 'f', -1, 64)
		}
		if c.Max != nil {
			row["max"] = strconv.FormatFloat(*c.Max, 'f', -1, 64)
		}
		rows = append(rows, row)
	}
	return writeCSV(w.dataPath("constraints.csv"),
		[]string{"criterion_id", "min", "max", "unit", "prefer", "must_have", "note"}, rows)
}

// UpsertConstraint creates or replaces a constraint for a criterion.
func (w *Workspace) UpsertConstraint(item Constraint) error {
	if item.CriterionID == "" {
		return fmt.Errorf("criterion_id is required")
	}
	if item.Prefer != "" && item.Prefer != "higher" && item.Prefer != "lower" {
		return fmt.Errorf("prefer must be higher, lower, or empty")
	}
	if item.Min != nil && item.Max != nil && *item.Min > *item.Max {
		return fmt.Errorf("min %.4g > max %.4g", *item.Min, *item.Max)
	}
	if !item.MustHave && item.Min == nil && item.Max == nil && item.Unit == "" && item.Prefer == "" {
		return fmt.Errorf("provide --prefer, --must-have, and/or --min/--max/--unit")
	}
	items, err := w.Constraints()
	if err != nil {
		return err
	}
	kept := items[:0]
	for _, c := range items {
		if c.CriterionID == item.CriterionID {
			continue
		}
		kept = append(kept, c)
	}
	kept = append(kept, item)
	return w.WriteConstraints(kept)
}

// EvaluateConstraints returns alternatives that fail any numeric band/unit rule.
func (w *Workspace) EvaluateConstraints() ([]ConstraintViolation, error) {
	constraints, err := w.Constraints()
	if err != nil {
		return nil, err
	}
	if len(constraints) == 0 {
		return nil, nil
	}
	alts, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	attrs, err := w.Attributes()
	if err != nil {
		return nil, err
	}
	attrBy := map[string]AttributeRow{}
	for _, a := range attrs {
		attrBy[a.AlternativeID+"|"+a.CriterionID] = a
	}

	var out []ConstraintViolation
	for _, c := range constraints {
		for _, alt := range alts {
			at, ok := attrBy[alt.ID+"|"+c.CriterionID]
			if !ok || strings.TrimSpace(at.Value) == "" {
				out = append(out, ConstraintViolation{
					AlternativeID: alt.ID, CriterionID: c.CriterionID,
					Reason: "missing attribute for constrained criterion",
				})
				continue
			}
			if c.MustHave {
				if !AttributeIsTruthy(at.Value) {
					out = append(out, ConstraintViolation{
						AlternativeID: alt.ID, CriterionID: c.CriterionID,
						Reason: fmt.Sprintf("must-have failed (value %q is falsy)", at.Value),
					})
				}
				// must-have alone: skip numeric band checks unless min/max/unit also set
				if c.Min == nil && c.Max == nil && c.Unit == "" {
					continue
				}
			}
			v, err := engine.ParseNumericAttribute(at.Value)
			if err != nil {
				out = append(out, ConstraintViolation{
					AlternativeID: alt.ID, CriterionID: c.CriterionID,
					Reason: fmt.Sprintf("non-numeric attribute: %v", err),
				})
				continue
			}
			unit := strings.TrimSpace(at.Unit)
			if c.Unit != "" && !strings.EqualFold(unit, c.Unit) {
				out = append(out, ConstraintViolation{
					AlternativeID: alt.ID, CriterionID: c.CriterionID, Value: v, Unit: unit,
					Reason: fmt.Sprintf("unit %q != required %q", unit, c.Unit),
				})
				continue
			}
			if c.Min != nil && v < *c.Min {
				out = append(out, ConstraintViolation{
					AlternativeID: alt.ID, CriterionID: c.CriterionID, Value: v, Unit: unit,
					Reason: fmt.Sprintf("value %.4g < min %.4g", v, *c.Min),
				})
				continue
			}
			if c.Max != nil && v > *c.Max {
				out = append(out, ConstraintViolation{
					AlternativeID: alt.ID, CriterionID: c.CriterionID, Value: v, Unit: unit,
					Reason: fmt.Sprintf("value %.4g > max %.4g", v, *c.Max),
				})
			}
		}
	}
	return out, nil
}

// EligibleAlternativeIDs returns alt ids that pass all constraints (or all alts if none).
func (w *Workspace) EligibleAlternativeIDs() ([]string, []ConstraintViolation, error) {
	alts, err := w.Alternatives()
	if err != nil {
		return nil, nil, err
	}
	violations, err := w.EvaluateConstraints()
	if err != nil {
		return nil, nil, err
	}
	bad := map[string]bool{}
	for _, v := range violations {
		bad[v.AlternativeID] = true
	}
	var ok []string
	for _, a := range alts {
		if !bad[a.ID] {
			ok = append(ok, a.ID)
		}
	}
	return ok, violations, nil
}

// AttributeIsTruthy reports whether an attribute value counts as present/true
// for must-have constraints (parking, breakfast, etc.).
func AttributeIsTruthy(raw string) bool {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "", "0", "false", "no", "n", "não", "nao", "off", "none", "null":
		return false
	case "1", "true", "yes", "y", "sim", "on":
		return true
	}
	if v, err := engine.ParseNumericAttribute(raw); err == nil {
		return v > 0
	}
	// Non-empty free text (e.g. "gratis", "privado") counts as present.
	return s != ""
}

func parseTruthyFlag(raw string) bool {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "1", "true", "yes", "y", "sim":
		return true
	default:
		return false
	}
}

func parseOptFloat(raw string) (float64, bool, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false, nil
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return 0, false, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false, fmt.Errorf("invalid number %q", raw)
	}
	return v, true, nil
}
