package workspace

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/engine"
)

// Finding is one actionable doctor diagnostic.
type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // error | warn | info
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
	Matrix   string `json:"matrix,omitempty"`
	Left     string `json:"left,omitempty"`
	Right    string `json:"right,omitempty"`
}

// DoctorOptions controls which judgments are included and whether pending
// proposals fail the run (--strict).
type DoctorOptions struct {
	IncludeProposals bool
	Strict           bool
	// Purchase enables attribute provenance / FX / pairwise-direction checks
	// (also runs automatically when constraints.csv has rows).
	Purchase bool
}

// DoctorReport is the structured result of Doctor / validate.
type DoctorReport struct {
	Workspace  string    `json:"workspace"`
	Title      string    `json:"title"`
	OK         bool      `json:"ok"`
	Complete   bool      `json:"complete"`
	Consistent bool      `json:"consistent"`
	Proposals  int       `json:"proposals"`
	ExitCode   int       `json:"exit_code"`
	Findings   []Finding `json:"findings"`
}

// Doctor runs schema, graph, and CR diagnostics. ExitCode follows the
// ROADMAP contract: incomplete (2) beats inconsistent (3); proposals
// pending only yield 4 when Strict and the workspace is otherwise ready.
func (w *Workspace) Doctor(opts DoctorOptions) (*DoctorReport, error) {
	report := &DoctorReport{
		Workspace: w.Root,
		Findings:  make([]Finding, 0),
	}

	if _, err := os.Stat(w.TomlPath()); err != nil {
		if os.IsNotExist(err) {
			report.Findings = append(report.Findings, Finding{
				Code:     "missing_toml",
				Severity: "error",
				Message:  fmt.Sprintf("missing ahp.toml under %s", w.Root),
				Fix:      "ahp init",
			})
			report.ExitCode = cliout.ExitIncomplete
			return report, nil
		}
		return nil, err
	}

	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	report.Title = meta.Title

	for _, name := range []string{"criteria.csv", "alternatives.csv", "pairwise.csv", "attributes.csv"} {
		path := w.dataPath(name)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				report.Findings = append(report.Findings, Finding{
					Code:     "missing_data_file",
					Severity: "error",
					Message:  fmt.Sprintf("missing data/%s", name),
					Fix:      "ahp init",
				})
				continue
			}
			return nil, err
		}
	}

	criteria, err := w.Criteria()
	if err != nil {
		return nil, err
	}
	alternatives, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	attributes, err := w.Attributes()
	if err != nil {
		return nil, err
	}

	critIDs := map[string]struct{}{}
	for _, c := range criteria {
		critIDs[c.ID] = struct{}{}
	}
	altIDs := map[string]struct{}{}
	for _, a := range alternatives {
		altIDs[a.ID] = struct{}{}
	}

	if len(criteria) == 0 {
		report.Findings = append(report.Findings, Finding{
			Code:     "no_criteria",
			Severity: "error",
			Message:  "no criteria defined",
			Fix:      "ahp add-criterion <id> <name>",
		})
	}
	if len(alternatives) == 0 {
		report.Findings = append(report.Findings, Finding{
			Code:     "no_alternatives",
			Severity: "error",
			Message:  "no alternatives defined",
			Fix:      "ahp add-alternative <id> <name>",
		})
	}

	// Orphan parent_id / unknown schema refs.
	for _, c := range criteria {
		if c.ParentID == "" {
			continue
		}
		if _, ok := critIDs[c.ParentID]; !ok {
			report.Findings = append(report.Findings, Finding{
				Code:     "orphan_parent",
				Severity: "error",
				Message:  fmt.Sprintf("criterion %q has unknown parent_id %q", c.ID, c.ParentID),
				Fix:      "ahp add-criterion " + c.ParentID + " <name>",
			})
		}
	}

	proposals := 0
	for _, p := range pairwise {
		if p.Status == "proposal" {
			proposals++
		}
		report.Findings = append(report.Findings, pairwiseIDFindings(p, critIDs, altIDs)...)
	}
	report.Proposals = proposals

	for _, a := range attributes {
		if _, ok := altIDs[a.AlternativeID]; !ok {
			report.Findings = append(report.Findings, Finding{
				Code:     "unknown_attribute_alt",
				Severity: "error",
				Message:  fmt.Sprintf("attributes: unknown alternative_id %q", a.AlternativeID),
				Fix:      "ahp add-alternative " + a.AlternativeID + " <name>",
			})
		}
		if _, ok := critIDs[a.CriterionID]; !ok {
			report.Findings = append(report.Findings, Finding{
				Code:     "unknown_attribute_crit",
				Severity: "error",
				Message:  fmt.Sprintf("attributes: unknown criterion_id %q", a.CriterionID),
				Fix:      "ahp add-criterion " + a.CriterionID + " <name>",
			})
		}
	}

	result, err := w.Compute(opts.IncludeProposals)
	if err != nil {
		return nil, err
	}
	report.Complete = result.Complete
	report.Consistent = result.Consistent

	// If committed-only view is incomplete but proposals alone would complete the
	// workspace, treat gaps as proposals_pending (not empty/missing errors).
	filledByProposals := false
	if proposals > 0 && !opts.IncludeProposals && !result.Complete {
		if withProp, err := w.Compute(true); err == nil && withProp.Complete {
			filledByProposals = true
		}
	}

	// Missing pairs / empty matrices (skip when proposals would fill the gaps).
	if !filledByProposals {
		for key, m := range result.Matrices {
			if len(m.IDs) < 2 || len(m.Missing) == 0 {
				continue
			}
			allMissing := len(m.Missing) == len(engine.OrderedPairs(m.IDs))
			if allMissing {
				report.Findings = append(report.Findings, Finding{
					Code:     "empty_matrix",
					Severity: "error",
					Message:  fmt.Sprintf("matrix %s has no pairwise judgments (%d pairs needed)", key, len(m.Missing)),
					Fix:      fmt.Sprintf("ahp pair set %s <left> <right> <value>", key),
					Matrix:   key,
				})
				continue
			}
			for _, p := range m.Missing {
				left, right := p["left"], p["right"]
				report.Findings = append(report.Findings, Finding{
					Code:     "missing_pair",
					Severity: "error",
					Message:  fmt.Sprintf("missing pair in %s: %s vs %s", key, left, right),
					Fix:      fmt.Sprintf("ahp pair set %s %s %s <saaty-value>", key, left, right),
					Matrix:   key,
					Left:     left,
					Right:    right,
				})
			}
		}
	}

	// Attributes present but alt:<criterion> never rated → agents often leave
	// equal-weight fallback after a failed rate. Point them at ahp rate.
	report.Findings = append(report.Findings, w.attrsUnratedFindings(result)...)

	// CR hotspots (only for complete matrices).
	for key, m := range result.Matrices {
		if !m.Complete || m.CR == nil || *m.CR <= engine.CRAccept {
			continue
		}
		report.Findings = append(report.Findings, Finding{
			Code:     "cr_hotspot",
			Severity: "error",
			Message:  fmt.Sprintf("matrix %s CR=%.3f exceeds 0.10", key, *m.CR),
			Fix:      "ahp pair repairs  # or ahp doctor",
			Matrix:   key,
		})
		for _, h := range m.Repairs {
			report.Findings = append(report.Findings, Finding{
				Code:     "cr_repair",
				Severity: "warn",
				Message: fmt.Sprintf("repair %s: %s vs %s %s → %s",
					key, h.Left, h.Right, engine.FormatSaaty(h.Current), engine.FormatSaaty(h.Suggested)),
				Fix: fmt.Sprintf("ahp pair set %s %s %s %s --as proposal",
					key, h.Left, h.Right, engine.FormatSaaty(h.Suggested)),
				Matrix: key,
				Left:   h.Left,
				Right:  h.Right,
			})
		}
	}

	if proposals > 0 {
		sev := "info"
		if opts.Strict {
			sev = "warn"
		}
		report.Findings = append(report.Findings, Finding{
			Code:     "proposals_pending",
			Severity: sev,
			Message:  fmt.Sprintf("%d proposal judgment(s) pending", proposals),
			Fix:      "ahp plan  # then ahp apply -y",
		})
	}

	constraints, err := w.Constraints()
	if err != nil {
		return nil, err
	}
	violations, err := w.EvaluateConstraints()
	if err != nil {
		return nil, err
	}
	for _, v := range violations {
		report.Findings = append(report.Findings, Finding{
			Code:     "constraint_violation",
			Severity: "error",
			Message:  fmt.Sprintf("%s fails constraint on %s: %s", v.AlternativeID, v.CriterionID, v.Reason),
			Fix:      "ahp constrain " + v.CriterionID + " --min/--max  # or remove/fix alternative",
		})
	}

	runPurchase := opts.Purchase || len(constraints) > 0
	if runPurchase {
		pf, err := w.PurchaseIntegrityFindings()
		if err != nil {
			return nil, err
		}
		// Avoid duplicating band violations already reported as constraint_violation.
		for _, f := range pf {
			if len(constraints) > 0 && (f.Code == "below_budget" || f.Code == "above_budget" || f.Code == "missing_price_attr") {
				continue
			}
			report.Findings = append(report.Findings, f)
		}
	}

	sort.SliceStable(report.Findings, func(i, j int) bool {
		return findingRank(report.Findings[i]) < findingRank(report.Findings[j]) ||
			(findingRank(report.Findings[i]) == findingRank(report.Findings[j]) &&
				report.Findings[i].Message < report.Findings[j].Message)
	})

	incomplete := hasCode(report.Findings, "missing_toml", "missing_data_file", "no_criteria", "no_alternatives",
		"orphan_parent", "unknown_pairwise_id", "unknown_matrix", "unknown_attribute_alt", "unknown_attribute_crit",
		"empty_matrix", "missing_pair") || (!result.Complete && !filledByProposals)
	inconsistent := hasCode(report.Findings, "cr_hotspot") || (result.Complete && !result.Consistent)
	purchaseBlocker := hasCode(report.Findings,
		"fx_or_foreign_source", "missing_source", "missing_unit", "unit_mismatch", "non_numeric",
		"pairwise_contradicts_attrs", "constraint_violation", "below_budget", "above_budget", "missing_price_attr")

	if incomplete {
		report.Complete = false
	}
	if filledByProposals {
		// Committed view is gappy, but proposals would complete — surface readiness honestly.
		report.Complete = false
	}

	switch {
	case incomplete:
		report.ExitCode = cliout.ExitIncomplete
	case inconsistent:
		report.ExitCode = cliout.ExitInconsistent
	case purchaseBlocker:
		report.ExitCode = cliout.ExitInconsistent
	case opts.Strict && proposals > 0:
		report.ExitCode = cliout.ExitProposals
	default:
		report.ExitCode = cliout.ExitOK
	}
	report.OK = report.ExitCode == cliout.ExitOK
	return report, nil
}

// ValidateGraph is an alias for Doctor without strict proposal failure.
func (w *Workspace) ValidateGraph(includeProposals bool) (*DoctorReport, error) {
	return w.Doctor(DoctorOptions{IncludeProposals: includeProposals})
}

func pairwiseIDFindings(p PairwiseRow, critIDs, altIDs map[string]struct{}) []Finding {
	var out []Finding
	matrix := p.Matrix
	switch {
	case matrix == "criteria":
		for _, id := range []string{p.Left, p.Right} {
			if _, ok := critIDs[id]; !ok {
				out = append(out, Finding{
					Code:     "unknown_pairwise_id",
					Severity: "error",
					Message:  fmt.Sprintf("pairwise criteria: unknown id %q", id),
					Fix:      "ahp add-criterion " + id + " <name>",
					Matrix:   matrix,
					Left:     p.Left,
					Right:    p.Right,
				})
			}
		}
	case strings.HasPrefix(matrix, "criteria:"):
		parent := strings.TrimPrefix(matrix, "criteria:")
		if _, ok := critIDs[parent]; !ok {
			out = append(out, Finding{
				Code:     "unknown_matrix",
				Severity: "error",
				Message:  fmt.Sprintf("pairwise matrix %q references unknown criterion %q", matrix, parent),
				Fix:      "ahp add-criterion " + parent + " <name>",
				Matrix:   matrix,
			})
		}
		for _, id := range []string{p.Left, p.Right} {
			if _, ok := critIDs[id]; !ok {
				out = append(out, Finding{
					Code:     "unknown_pairwise_id",
					Severity: "error",
					Message:  fmt.Sprintf("pairwise %s: unknown id %q", matrix, id),
					Fix:      "ahp add-criterion " + id + " <name>",
					Matrix:   matrix,
					Left:     p.Left,
					Right:    p.Right,
				})
			}
		}
	case strings.HasPrefix(matrix, "alt:"):
		crit := strings.TrimPrefix(matrix, "alt:")
		if _, ok := critIDs[crit]; !ok {
			out = append(out, Finding{
				Code:     "unknown_matrix",
				Severity: "error",
				Message:  fmt.Sprintf("pairwise matrix %q references unknown criterion %q", matrix, crit),
				Fix:      "ahp add-criterion " + crit + " <name>",
				Matrix:   matrix,
			})
		}
		for _, id := range []string{p.Left, p.Right} {
			if _, ok := altIDs[id]; !ok {
				out = append(out, Finding{
					Code:     "unknown_pairwise_id",
					Severity: "error",
					Message:  fmt.Sprintf("pairwise %s: unknown alternative id %q", matrix, id),
					Fix:      "ahp add-alternative " + id + " <name>",
					Matrix:   matrix,
					Left:     p.Left,
					Right:    p.Right,
				})
			}
		}
	default:
		out = append(out, Finding{
			Code:     "unknown_matrix",
			Severity: "error",
			Message:  fmt.Sprintf("pairwise matrix %q is not criteria, criteria:<id>, or alt:<id>", matrix),
			Matrix:   matrix,
			Left:     p.Left,
			Right:    p.Right,
		})
	}
	return out
}

// attrsUnratedFindings warns when ≥2 alternatives have numeric attributes for a
// criterion but matrix alt:<criterion> has no judgments at all.
func (w *Workspace) attrsUnratedFindings(_ *ComputeResult) []Finding {
	attrs, err := w.Attributes()
	if err != nil || len(attrs) == 0 {
		return nil
	}
	byCrit := map[string]map[string]bool{}
	for _, a := range attrs {
		if strings.TrimSpace(a.Value) == "" {
			continue
		}
		if _, err := engine.ParseNumericAttribute(a.Value); err != nil {
			continue
		}
		if byCrit[a.CriterionID] == nil {
			byCrit[a.CriterionID] = map[string]bool{}
		}
		byCrit[a.CriterionID][a.AlternativeID] = true
	}
	pairs, err := w.Pairwise()
	if err != nil {
		return nil
	}
	hasPair := map[string]bool{}
	for _, p := range pairs {
		if strings.HasPrefix(p.Matrix, "alt:") {
			hasPair[p.Matrix] = true
		}
	}
	var out []Finding
	for crit, alts := range byCrit {
		if len(alts) < 2 {
			continue
		}
		matrix := "alt:" + crit
		if hasPair[matrix] {
			continue
		}
		// Prefer pointing at rate; stretch helps tight bands from hotel dogfood.
		prefer := "higher"
		if cons, err := w.Constraints(); err == nil {
			for _, c := range cons {
				if c.CriterionID == crit && c.Prefer != "" {
					prefer = c.Prefer
					break
				}
			}
		}
		out = append(out, Finding{
			Code:     "attrs_unrated",
			Severity: "warn",
			Message:  fmt.Sprintf("%d alternatives have numeric attributes on %q but matrix %s has no judgments — ranking may use equal-weight fallback", len(alts), crit, matrix),
			Fix:      fmt.Sprintf("ahp rate --criterion %s --prefer %s [--stretch] && ahp plan", crit, prefer),
			Matrix:   matrix,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Matrix < out[j].Matrix })
	return out
}

func hasCode(findings []Finding, codes ...string) bool {
	set := map[string]struct{}{}
	for _, c := range codes {
		set[c] = struct{}{}
	}
	for _, f := range findings {
		if _, ok := set[f.Code]; ok {
			return true
		}
	}
	return false
}

func findingRank(f Finding) int {
	switch f.Severity {
	case "error":
		return 0
	case "warn":
		return 1
	default:
		return 2
	}
}
