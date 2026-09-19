package workspace

import (
	"fmt"
	"sort"

	"github.com/lazarok09/ahp-method/internal/engine"
)

// ExplainResult is the workspace-facing explain payload.
type ExplainResult struct {
	Workspace        string               `json:"workspace"`
	Title            string               `json:"title"`
	IncludeProposals bool                 `json:"include_proposals"`
	Complete         bool                 `json:"complete"`
	FilterAlt        string               `json:"filter_alt,omitempty"`
	engine.ExplainResult
}

// Explain returns criterion contribution breakdowns for alternatives.
// If altID is non-empty, only that alternative is included.
func (w *Workspace) Explain(includeProposals bool, altID string) (*ExplainResult, error) {
	result, err := w.Compute(includeProposals)
	if err != nil {
		return nil, err
	}
	leaf, local, altIDs, altNames, critNames := synthesisInputs(result)
	if len(altIDs) == 0 {
		return nil, fmt.Errorf("no alternatives to explain")
	}
	if altID != "" {
		found := false
		for _, id := range altIDs {
			if id == altID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown alternative %q", altID)
		}
		altIDs = []string{altID}
	}
	exp := engine.Explain(leaf, local, altIDs, altNames, critNames)
	return &ExplainResult{
		Workspace: w.Root, Title: result.Title,
		IncludeProposals: includeProposals, Complete: result.Complete,
		FilterAlt: altID, ExplainResult: exp,
	}, nil
}

// SensitivityResult wraps engine sensitivity with workspace metadata.
type SensitivityResult struct {
	Workspace        string `json:"workspace"`
	Title            string `json:"title"`
	IncludeProposals bool   `json:"include_proposals"`
	Complete         bool   `json:"complete"`
	engine.SensitivityResult
}

// Sensitivity runs one-at-a-time leaf-weight perturbation analysis.
func (w *Workspace) Sensitivity(includeProposals bool, delta float64) (*SensitivityResult, error) {
	result, err := w.Compute(includeProposals)
	if err != nil {
		return nil, err
	}
	if !result.Complete {
		return nil, fmt.Errorf("workspace incomplete — fill pairwise (or pass --include-proposals) before sensitivity")
	}
	leaf, local, altIDs, altNames, critNames := synthesisInputs(result)
	sens := engine.Sensitivity(leaf, local, altIDs, altNames, critNames, delta)
	return &SensitivityResult{
		Workspace: w.Root, Title: result.Title,
		IncludeProposals: includeProposals, Complete: result.Complete,
		SensitivityResult: sens,
	}, nil
}

func synthesisInputs(result *ComputeResult) (
	leaf map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames, critNames map[string]string,
) {
	leaf = result.LeafWeights
	if leaf == nil {
		leaf = map[string]float64{}
	}
	local = map[string]map[string]float64{}
	for cid := range leaf {
		key := "alt:" + cid
		if m, ok := result.Matrices[key]; ok {
			local[cid] = m.Weights
		} else {
			local[cid] = map[string]float64{}
		}
	}
	altNames = map[string]string{}
	altIDs = make([]string, 0, len(result.Alternatives))
	for _, a := range result.Alternatives {
		altIDs = append(altIDs, a.ID)
		altNames[a.ID] = a.Name
	}
	critNames = map[string]string{}
	for _, c := range result.Criteria {
		critNames[c.ID] = c.Name
	}
	return leaf, local, altIDs, altNames, critNames
}

// SuggestFromAttributesOptions controls attribute→Saaty proposal generation.
type SuggestFromAttributesOptions struct {
	CriterionID string
	Prefer      engine.PreferDirection // higher | lower
	DryRun      bool
	// Refresh rewrites matching pairs as proposals even when a committed row
	// already exists (demotes committed→proposal). Never writes status=committed.
	Refresh bool
	// Stretch affine-shifts attributes before ratios so tight bands discriminate
	// (e.g. guest scores 8.4 vs 9.5). Non-positive values always shift.
	Stretch bool
}

// SuggestFromAttributesResult lists proposed pairs derived from attributes.
type SuggestFromAttributesResult struct {
	Workspace   string                      `json:"workspace"`
	CriterionID string                      `json:"criterion_id"`
	Matrix      string                      `json:"matrix"`
	Prefer      string                      `json:"prefer"`
	DryRun      bool                        `json:"dry_run"`
	Refresh     bool                        `json:"refresh"`
	Stretch     bool                        `json:"stretch"`
	Written     int                         `json:"written"`
	Skipped     int                         `json:"skipped_committed"`
	Refreshed   int                         `json:"refreshed"` // demoted committed→proposal (subset of Written)
	Suggestions []engine.AttrPairSuggestion `json:"suggestions"`
	Summary     string                      `json:"summary"`
}

// SuggestFromAttributes converts numeric attributes under a criterion into
// Saaty pairwise proposals on matrix alt:<criterion>. Never writes committed.
func (w *Workspace) SuggestFromAttributes(opts SuggestFromAttributesOptions) (*SuggestFromAttributesResult, error) {
	if opts.CriterionID == "" {
		return nil, fmt.Errorf("criterion is required")
	}
	prefer := opts.Prefer
	if prefer == "" {
		prefer = engine.PreferHigher
	}
	if prefer != engine.PreferHigher && prefer != engine.PreferLower {
		return nil, fmt.Errorf("prefer must be higher or lower")
	}

	criteria, err := w.Criteria()
	if err != nil {
		return nil, err
	}
	foundCrit := false
	for _, c := range criteria {
		if c.ID == opts.CriterionID {
			foundCrit = true
			break
		}
	}
	if !foundCrit {
		return nil, fmt.Errorf("unknown criterion %q", opts.CriterionID)
	}

	alts, err := w.Alternatives()
	if err != nil {
		return nil, err
	}
	attrs, err := w.Attributes()
	if err != nil {
		return nil, err
	}

	values := map[string]float64{}
	var ids []string
	for _, a := range alts {
		var raw string
		for _, at := range attrs {
			if at.AlternativeID == a.ID && at.CriterionID == opts.CriterionID {
				raw = at.Value
				break
			}
		}
		if raw == "" {
			continue
		}
		v, err := engine.ParseNumericAttribute(raw)
		if err != nil {
			return nil, fmt.Errorf("attribute %s/%s: %w", a.ID, opts.CriterionID, err)
		}
		values[a.ID] = v
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)

	sug, err := engine.SuggestPairwiseFromValuesOpts(ids, values, engine.SuggestFromValuesOptions{
		HigherBetter: prefer == engine.PreferHigher,
		Stretch:      opts.Stretch,
	})
	if err != nil {
		return nil, err
	}

	matrix := "alt:" + opts.CriterionID
	out := &SuggestFromAttributesResult{
		Workspace: w.Root, CriterionID: opts.CriterionID, Matrix: matrix,
		Prefer: string(prefer), DryRun: opts.DryRun, Refresh: opts.Refresh, Stretch: opts.Stretch,
		Suggestions: sug,
	}

	existing, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	committedKeys := map[string]bool{}
	for _, p := range existing {
		if p.Matrix == matrix && p.Status == "committed" {
			committedKeys[pairSet(matrix, p.Left, p.Right)] = true
		}
	}

	written := 0
	skipped := 0
	refreshed := 0
	for _, s := range sug {
		key := pairSet(matrix, s.Left, s.Right)
		wasCommitted := committedKeys[key]
		if wasCommitted && !opts.Refresh {
			skipped++
			continue
		}
		if opts.DryRun {
			written++
			if wasCommitted {
				refreshed++
			}
			continue
		}
		row := PairwiseRow{
			Matrix: matrix, Left: s.Left, Right: s.Right,
			Value: s.Value, Status: "proposal", Note: s.Note,
		}
		if err := w.UpsertPairwise(row); err != nil {
			return nil, err
		}
		written++
		if wasCommitted {
			refreshed++
		}
	}
	out.Written = written
	out.Skipped = skipped
	out.Refreshed = refreshed
	action := "wrote"
	if opts.DryRun {
		action = "would write"
	}
	out.Summary = fmt.Sprintf("%s %d proposal(s) on %s (%s-better%s); skipped %d committed; refreshed %d",
		action, written, matrix, prefer, stretchNote(opts.Stretch), skipped, refreshed)
	return out, nil
}

func stretchNote(stretch bool) string {
	if stretch {
		return ", stretch"
	}
	return ""
}
