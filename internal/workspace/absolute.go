package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// AbsoluteMethodResult is hybrid absolute measurement (sum-norm × criterion weights).
type AbsoluteMethodResult struct {
	Workspace        string                 `json:"workspace"`
	Title            string                 `json:"title"`
	Method           string                 `json:"method"` // absolute|hybrid
	Caveat           string                 `json:"caveat"`
	AlternativeIDs   []string               `json:"alternative_ids"`
	Matrix           *engine.AbsoluteMatrix `json:"matrix"`
	CriterionWeights map[string]float64     `json:"criterion_weights,omitempty"`
	Scores           map[string]float64     `json:"scores,omitempty"`
	Ranking          []RankRow              `json:"ranking,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
}

// GaussianMethodResult is Santos-style AHP-Gaussian comparative ranking.
type GaussianMethodResult struct {
	Workspace string                 `json:"workspace"`
	Title     string                 `json:"title"`
	Method    string                 `json:"method"` // gaussian
	Caveat    string                 `json:"caveat"`
	Result    *engine.GaussianResult `json:"result"`
	Ranking   []RankRow              `json:"ranking"`
	Warnings  []string               `json:"warnings,omitempty"`
}

const (
	absoluteMatrixCaveat = "Comparative only. Sum-normalized attributes (not Saaty pairwise). These local scores have no Saaty consistency ratio (CR)."
	hybridAbsoluteCaveat = "Comparative hybrid: attribute sum-norm × Saaty criteria leaf weights. Alternative pairwise unused. Attribute-derived scores have no Saaty CR."
	gaussianCaveat       = "Comparative only. Gaussian criterion weights reflect dispersion across the shortlist (σ/μ), not decision-maker importance. Do not treat these scores as having a Saaty consistency ratio (CR)."
)

// AbsoluteReadinessIssue is a typed doctor/readiness message for absolute/Gaussian.
type AbsoluteReadinessIssue struct {
	Code    string // method_missing_prefer | method_incomplete_matrix | method_not_ready
	Message string
}

// LeafCriterionIDs returns criteria that have no children (attribute/Gaussian leaves).
func LeafCriterionIDs(criteria []Criterion) []string {
	hasChild := map[string]bool{}
	for _, c := range criteria {
		if c.ParentID != "" {
			hasChild[c.ParentID] = true
		}
	}
	var leaves []string
	for _, c := range criteria {
		if !hasChild[c.ID] {
			leaves = append(leaves, c.ID)
		}
	}
	sort.Strings(leaves)
	return leaves
}

// PreferForCriterion returns prefer from constraints.csv, or empty if unset.
func (w *Workspace) PreferForCriterion(criterionID string) (engine.PreferDirection, error) {
	items, err := w.Constraints()
	if err != nil {
		return "", err
	}
	for _, c := range items {
		if c.CriterionID == criterionID && (c.Prefer == "higher" || c.Prefer == "lower") {
			return engine.PreferDirection(c.Prefer), nil
		}
	}
	return "", nil
}

// BuildAbsoluteColumnInputs collects attribute-ready leaf columns for eligible alts.
// Returns columns plus readiness issues (empty prefer, missing attrs, etc.).
func (w *Workspace) BuildAbsoluteColumnInputs() (altIDs []string, cols []engine.AbsoluteColumnInput, issues []AbsoluteReadinessIssue, err error) {
	criteria, err := w.Criteria()
	if err != nil {
		return nil, nil, nil, err
	}
	attrs, err := w.Attributes()
	if err != nil {
		return nil, nil, nil, err
	}
	eligible, _, err := w.EligibleAlternativeIDs()
	if err != nil {
		return nil, nil, nil, err
	}
	if len(eligible) < 2 {
		return nil, nil, []AbsoluteReadinessIssue{{
			Code: "method_not_ready", Message: "need at least two eligible alternatives",
		}}, nil
	}
	altNames := map[string]bool{}
	for _, id := range eligible {
		altNames[id] = true
	}

	// attr[crit][alt] = raw string
	byCrit := map[string]map[string]string{}
	for _, a := range attrs {
		if !altNames[a.AlternativeID] {
			continue
		}
		if byCrit[a.CriterionID] == nil {
			byCrit[a.CriterionID] = map[string]string{}
		}
		byCrit[a.CriterionID][a.AlternativeID] = a.Value
	}

	leaves := LeafCriterionIDs(criteria)
	for _, cid := range leaves {
		prefer, err := w.PreferForCriterion(cid)
		if err != nil {
			return nil, nil, nil, err
		}
		rowMap := byCrit[cid]
		completeNumeric := true
		values := map[string]float64{}
		for _, aid := range eligible {
			raw, ok := rowMap[aid]
			if !ok || strings.TrimSpace(raw) == "" {
				completeNumeric = false
				break
			}
			v, err := engine.ParseNumericAttribute(raw)
			if err != nil {
				completeNumeric = false
				break
			}
			values[aid] = v
		}
		if !completeNumeric {
			if len(rowMap) > 0 {
				issues = append(issues, AbsoluteReadinessIssue{
					Code:    "method_incomplete_matrix",
					Message: fmt.Sprintf("criterion %s: decision matrix incomplete (need numeric attributes for all eligible alternatives)", cid),
				})
			}
			continue
		}
		if prefer == "" {
			issues = append(issues, AbsoluteReadinessIssue{
				Code:    "method_missing_prefer",
				Message: fmt.Sprintf("criterion %s: missing prefer (ahp constrain %s --prefer higher|lower)", cid, cid),
			})
			continue
		}
		cols = append(cols, engine.AbsoluteColumnInput{
			CriterionID: cid,
			Prefer:      prefer,
			Values:      values,
		})
	}
	if len(cols) == 0 {
		if len(issues) == 0 {
			issues = append(issues, AbsoluteReadinessIssue{
				Code:    "method_not_ready",
				Message: "no attribute-ready leaf criteria (need numeric attributes + prefer for each leaf)",
			})
		}
		return eligible, nil, issues, nil
	}
	return eligible, cols, issues, nil
}

func issueMessages(issues []AbsoluteReadinessIssue) []string {
	out := make([]string, 0, len(issues))
	for _, is := range issues {
		out = append(out, is.Message)
	}
	return out
}

// Absolute computes sum-norm matrix and optional hybrid scores using Saaty criteria leaf weights.
// Hybrid scores attach only when criteria (and criteria:*) matrices are complete with CR ≤ 0.10.
// Incomplete alt:* pairwise does not block hybrid — those matrices are unused in absolute measurement.
// Does not run full Saaty synthesis (no alt:* solve).
func (w *Workspace) Absolute(includeProposals bool) (*AbsoluteMethodResult, error) {
	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	altIDs, cols, issues, err := w.BuildAbsoluteColumnInputs()
	if err != nil {
		return nil, err
	}
	out := &AbsoluteMethodResult{
		Workspace: w.Root,
		Title:     meta.Title,
		Method:    "absolute",
		Caveat:    absoluteMatrixCaveat,
		Warnings:  issueMessages(issues),
	}
	if len(cols) == 0 {
		return out, fmt.Errorf("absolute: %s", joinIssues(issues))
	}
	m, err := engine.BuildAbsoluteMatrix(altIDs, cols)
	if err != nil {
		return nil, err
	}
	out.AlternativeIDs = altIDs
	out.Matrix = m

	cw, err := w.CriteriaLeafWeights(includeProposals)
	if err != nil {
		return nil, err
	}
	wts := map[string]float64{}
	missingW := false
	for _, col := range cols {
		wt, ok := cw.LeafWeights[col.CriterionID]
		if !ok {
			missingW = true
			break
		}
		wts[col.CriterionID] = wt
	}
	if missingW || len(wts) == 0 {
		out.Warnings = append(out.Warnings, "criteria leaf weights incomplete — matrix only (no hybrid scores); fill criteria pairwise or use ahp gaussian")
		return out, nil
	}
	if ready, reason := criteriaMatricesReady(cw.Matrices); !ready {
		out.Warnings = append(out.Warnings, "hybrid skipped: "+reason+"; matrix only — fix criteria pairwise CR or use ahp gaussian")
		return out, nil
	}
	scores, err := engine.ScoreAbsolute(m, wts)
	if err != nil {
		out.Warnings = append(out.Warnings, err.Error())
		return out, nil
	}
	alts, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	out.Method = "hybrid"
	out.Caveat = hybridAbsoluteCaveat
	out.CriterionWeights = wts
	out.Scores = scores
	out.Ranking = rankFromScores(scores, alts)
	return out, nil
}

// CriteriaLeafWeightsResult is criteria hierarchy weights without alt:* synthesis.
type CriteriaLeafWeightsResult struct {
	LeafWeights map[string]float64
	Matrices    map[string]MatrixPayload // criteria and criteria:* only
}

// CriteriaLeafWeights solves Saaty criteria (and nested criteria:*) matrices only.
func (w *Workspace) CriteriaLeafWeights(includeProposals bool) (*CriteriaLeafWeightsResult, error) {
	criteria, err := w.Criteria()
	if err != nil {
		return nil, err
	}
	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	var used []PairwiseRow
	for _, p := range pairwise {
		if includeProposals || p.Status == "committed" {
			used = append(used, p)
		}
	}
	var root []Criterion
	childByParent := map[string][]Criterion{}
	critNames := map[string]string{}
	for _, c := range criteria {
		critNames[c.ID] = c.Name
		if c.ParentID == "" {
			root = append(root, c)
		} else {
			childByParent[c.ParentID] = append(childByParent[c.ParentID], c)
		}
	}
	leaf, matrices, _ := criteriaHierarchyWeights(root, childByParent, critNames, groupPairs(used))
	return &CriteriaLeafWeightsResult{LeafWeights: leaf, Matrices: matrices}, nil
}

// criteriaMatricesReady checks criteria / criteria:* matrices only (not alt:*).
func criteriaMatricesReady(matrices map[string]MatrixPayload) (bool, string) {
	if len(matrices) == 0 {
		return false, "no criteria matrix"
	}
	found := false
	for key, m := range matrices {
		if strings.HasPrefix(key, "alt:") {
			continue
		}
		found = true
		if !m.Complete {
			return false, fmt.Sprintf("%s matrix incomplete (equal-weight fallback is not hybrid)", key)
		}
		if m.CR != nil && *m.CR > engine.CRAccept {
			return false, fmt.Sprintf("%s CR=%.3f exceeds 0.10", key, *m.CR)
		}
	}
	if !found {
		return false, "no criteria matrix"
	}
	return true, ""
}

// Gaussian runs Santos AHP-Gaussian on attribute-ready leaves.
func (w *Workspace) Gaussian() (*GaussianMethodResult, error) {
	meta, err := w.LoadMeta()
	if err != nil {
		return nil, err
	}
	altIDs, cols, issues, err := w.BuildAbsoluteColumnInputs()
	if err != nil {
		return nil, err
	}
	out := &GaussianMethodResult{
		Workspace: w.Root,
		Title:     meta.Title,
		Method:    "gaussian",
		Caveat:    gaussianCaveat,
		Warnings:  issueMessages(issues),
	}
	if len(cols) == 0 {
		return out, fmt.Errorf("gaussian: %s", joinIssues(issues))
	}
	m, err := engine.BuildAbsoluteMatrix(altIDs, cols)
	if err != nil {
		return nil, err
	}
	g, err := engine.GaussianFromAbsolute(m)
	if err != nil {
		return nil, err
	}
	alts, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	out.Result = g
	out.Ranking = rankFromScores(g.Scores, alts)
	return out, nil
}

func joinIssues(issues []AbsoluteReadinessIssue) string {
	if len(issues) == 0 {
		return "not ready"
	}
	return issues[0].Message
}

func rankFromScores(scores map[string]float64, alts []Alternative) []RankRow {
	names := map[string]string{}
	for _, a := range alts {
		names[a.ID] = a.Name
	}
	ordered := engine.RankScores(scores)
	out := make([]RankRow, 0, len(ordered))
	for i, id := range ordered {
		name := names[id]
		if name == "" {
			name = id
		}
		out = append(out, RankRow{Rank: i + 1, ID: id, Name: name, Weight: scores[id]})
	}
	return out
}

// WriteAbsoluteOutputs writes output/absolute.json.
func (w *Workspace) WriteAbsoluteOutputs(res *AbsoluteMethodResult) (string, error) {
	if err := os.MkdirAll(w.OutputDir(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(w.OutputDir(), "absolute.json")
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// WriteGaussianOutputs writes output/gaussian.json.
func (w *Workspace) WriteGaussianOutputs(res *GaussianMethodResult) (string, error) {
	if err := os.MkdirAll(w.OutputDir(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(w.OutputDir(), "gaussian.json")
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// AttachComparative fills Absolute and/or Gaussian on an existing Saaty ComputeResult
// and writes comparative JSON. Saaty ranking remains canonical.
// method: saaty (no-op) | gaussian | hybrid | absolute | compare
func (w *Workspace) AttachComparative(result *ComputeResult, includeProposals bool, method string) error {
	if result == nil {
		return fmt.Errorf("nil compute result")
	}
	switch method {
	case "", "saaty":
		return nil
	case "gaussian":
		gres, err := w.Gaussian()
		if err != nil {
			return err
		}
		result.Gaussian = gres
		if _, err := w.WriteGaussianOutputs(gres); err != nil {
			return err
		}
		return nil
	case "absolute":
		ares, err := w.Absolute(includeProposals)
		if err != nil {
			return err
		}
		result.Absolute = ares
		if _, err := w.WriteAbsoluteOutputs(ares); err != nil {
			return err
		}
		return nil
	case "hybrid":
		ares, err := w.Absolute(includeProposals)
		if err != nil {
			return err
		}
		if ares.Method != "hybrid" {
			msg := "hybrid requires complete criteria pairwise with CR ≤ 0.10"
			if len(ares.Warnings) > 0 {
				msg = ares.Warnings[len(ares.Warnings)-1]
			}
			return fmt.Errorf("hybrid: %s (got matrix-only; use ahp absolute or ahp gaussian)", msg)
		}
		result.Absolute = ares
		if _, err := w.WriteAbsoluteOutputs(ares); err != nil {
			return err
		}
		return nil
	case "compare":
		if ares, err := w.Absolute(includeProposals); err != nil {
			result.Warnings = append(result.Warnings, "absolute: "+err.Error())
		} else {
			result.Absolute = ares
			if _, err := w.WriteAbsoluteOutputs(ares); err != nil {
				return err
			}
		}
		gres, err := w.Gaussian()
		if err != nil {
			return err
		}
		result.Gaussian = gres
		if _, err := w.WriteGaussianOutputs(gres); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown method %q (saaty|gaussian|hybrid|absolute|compare)", method)
	}
}

// MethodFindings returns doctor findings for absolute/gaussian readiness.
func (w *Workspace) MethodFindings() ([]Finding, error) {
	_, cols, issues, err := w.BuildAbsoluteColumnInputs()
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, is := range issues {
		fix := "ahp set-attribute … && ahp constrain <c> --prefer higher|lower"
		if is.Code == "method_missing_prefer" {
			fix = "ahp constrain <criterion> --prefer higher|lower"
		}
		sev := "warn"
		if len(cols) == 0 {
			sev = "error"
		}
		findings = append(findings, Finding{
			Code: is.Code, Severity: sev, Message: is.Message, Fix: fix,
		})
	}
	if len(cols) > 0 {
		findings = append(findings, Finding{
			Code:     "method_ready_hint",
			Severity: "info",
			Message:  fmt.Sprintf("%d attribute-ready leaf criterion(ia) — comparative path: ahp gaussian / ahp absolute", len(cols)),
			Fix:      "ahp doctor --method && ahp gaussian",
		})
	}
	return findings, nil
}
