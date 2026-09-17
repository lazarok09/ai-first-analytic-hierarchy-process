package workspace

import (
	"fmt"
	"sort"
)

// PlanResult is a dry-run of committing all proposal judgments (no CSV writes).
type PlanResult struct {
	Workspace          string           `json:"workspace"`
	Title              string           `json:"title"`
	Proposals          int              `json:"proposals"`
	CompleteBefore     bool             `json:"complete_before"`
	CompleteAfter      bool             `json:"complete_after"`
	ConsistentBefore   bool             `json:"consistent_before"`
	ConsistentAfter    bool             `json:"consistent_after"`
	RankingBefore      []RankRow        `json:"ranking_before"`
	RankingAfter       []RankRow        `json:"ranking_after"`
	RankDeltas         []RankDelta      `json:"rank_deltas"`
	MatrixDeltas       []MatrixCRDelta  `json:"matrix_deltas"`
	ProposalRows       []PairwiseRow    `json:"proposal_rows,omitempty"`
	WouldChangeRanking bool             `json:"would_change_ranking"`
	WouldChangeCR      bool             `json:"would_change_cr"`
	Summary            string           `json:"summary"`
}

// RankDelta compares one alternative before vs after applying proposals.
type RankDelta struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	RankBefore   int     `json:"rank_before"`
	RankAfter    int     `json:"rank_after"`
	WeightBefore float64 `json:"weight_before"`
	WeightAfter  float64 `json:"weight_after"`
	DeltaWeight  float64 `json:"delta_weight"`
	DeltaRank    int     `json:"delta_rank"` // negative = moved up
}

// MatrixCRDelta compares one matrix's completeness and CR.
type MatrixCRDelta struct {
	Matrix         string   `json:"matrix"`
	CompleteBefore bool     `json:"complete_before"`
	CompleteAfter  bool     `json:"complete_after"`
	CRBefore       *float64 `json:"cr_before,omitempty"`
	CRAfter        *float64 `json:"cr_after,omitempty"`
	DeltaCR        *float64 `json:"delta_cr,omitempty"`
}

// Plan recomputes as if proposals were committed and reports ranking/CR deltas.
// It never writes files or flips status columns.
func (w *Workspace) Plan() (*PlanResult, error) {
	before, err := w.Compute(false)
	if err != nil {
		return nil, err
	}
	after, err := w.Compute(true)
	if err != nil {
		return nil, err
	}

	pairwise, err := w.Pairwise()
	if err != nil {
		return nil, err
	}
	var proposals []PairwiseRow
	for _, p := range pairwise {
		if p.Status == "proposal" {
			proposals = append(proposals, p)
		}
	}

	rankDeltas := diffRanks(before.Ranking, after.Ranking)
	matrixDeltas := diffMatrices(before.Matrices, after.Matrices)

	wouldRank := false
	for _, d := range rankDeltas {
		if d.DeltaRank != 0 || absFloat(d.DeltaWeight) > 1e-9 {
			wouldRank = true
			break
		}
	}
	wouldCR := false
	for _, d := range matrixDeltas {
		if d.CompleteBefore != d.CompleteAfter {
			wouldCR = true
			break
		}
		if d.DeltaCR != nil && absFloat(*d.DeltaCR) > 1e-9 {
			wouldCR = true
			break
		}
	}

	summary := fmt.Sprintf("%d proposal(s): ranking change=%v, CR/completeness change=%v",
		len(proposals), wouldRank, wouldCR)

	return &PlanResult{
		Workspace:          w.Root,
		Title:              before.Title,
		Proposals:          len(proposals),
		CompleteBefore:     before.Complete,
		CompleteAfter:      after.Complete,
		ConsistentBefore:   before.Consistent,
		ConsistentAfter:    after.Consistent,
		RankingBefore:      before.Ranking,
		RankingAfter:       after.Ranking,
		RankDeltas:         rankDeltas,
		MatrixDeltas:       matrixDeltas,
		ProposalRows:       proposals,
		WouldChangeRanking: wouldRank,
		WouldChangeCR:      wouldCR,
		Summary:            summary,
	}, nil
}

func diffRanks(before, after []RankRow) []RankDelta {
	byID := map[string]RankRow{}
	for _, r := range before {
		byID[r.ID] = r
	}
	seen := map[string]bool{}
	var out []RankDelta
	for _, a := range after {
		seen[a.ID] = true
		b, ok := byID[a.ID]
		if !ok {
			out = append(out, RankDelta{
				ID: a.ID, Name: a.Name,
				RankBefore: 0, RankAfter: a.Rank,
				WeightBefore: 0, WeightAfter: a.Weight,
				DeltaWeight: a.Weight, DeltaRank: -a.Rank,
			})
			continue
		}
		out = append(out, RankDelta{
			ID: a.ID, Name: a.Name,
			RankBefore: b.Rank, RankAfter: a.Rank,
			WeightBefore: b.Weight, WeightAfter: a.Weight,
			DeltaWeight: a.Weight - b.Weight,
			DeltaRank:   a.Rank - b.Rank,
		})
	}
	for _, b := range before {
		if seen[b.ID] {
			continue
		}
		out = append(out, RankDelta{
			ID: b.ID, Name: b.Name,
			RankBefore: b.Rank, RankAfter: 0,
			WeightBefore: b.Weight, WeightAfter: 0,
			DeltaWeight: -b.Weight, DeltaRank: b.Rank,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RankAfter == 0 {
			return false
		}
		if out[j].RankAfter == 0 {
			return true
		}
		return out[i].RankAfter < out[j].RankAfter
	})
	return out
}

func diffMatrices(before, after map[string]MatrixPayload) []MatrixCRDelta {
	keys := map[string]struct{}{}
	for k := range before {
		keys[k] = struct{}{}
	}
	for k := range after {
		keys[k] = struct{}{}
	}
	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	sort.Strings(list)

	out := make([]MatrixCRDelta, 0, len(list))
	for _, k := range list {
		b, bok := before[k]
		a, aok := after[k]
		d := MatrixCRDelta{Matrix: k}
		if bok {
			d.CompleteBefore = b.Complete
			d.CRBefore = b.CR
		}
		if aok {
			d.CompleteAfter = a.Complete
			d.CRAfter = a.CR
		}
		if d.CRBefore != nil && d.CRAfter != nil {
			delta := *d.CRAfter - *d.CRBefore
			d.DeltaCR = &delta
		}
		out = append(out, d)
	}
	return out
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
