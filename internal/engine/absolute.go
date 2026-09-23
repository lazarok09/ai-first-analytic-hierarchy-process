package engine

import (
	"fmt"
	"math"
	"sort"
)

// AbsoluteColumn is one criterion's raw and sum-normalized scores across alternatives.
type AbsoluteColumn struct {
	CriterionID string             `json:"criterion_id"`
	Prefer      PreferDirection    `json:"prefer"`
	Raw         map[string]float64 `json:"raw"`
	Norm        map[string]float64 `json:"norm"`
}

// AbsoluteMatrix is the Santos-style decision matrix after cost inversion + sum-norm.
type AbsoluteMatrix struct {
	AlternativeIDs []string         `json:"alternative_ids"`
	Columns        []AbsoluteColumn `json:"columns"`
	// NormByCriterion[criterionID][altID] = n_{a,c}
	NormByCriterion map[string]map[string]float64 `json:"norm_by_criterion"`
}

// GaussianFactor is μ, σ, f=σ/μ and renormalized weight for one criterion.
type GaussianFactor struct {
	CriterionID string  `json:"criterion_id"`
	Mean        float64 `json:"mean"`
	SD          float64 `json:"sd"`
	Factor      float64 `json:"factor"`
	Weight      float64 `json:"weight"`
}

// GaussianResult is absolute matrix + Gaussian criterion weights + ranking scores.
type GaussianResult struct {
	Matrix  AbsoluteMatrix   `json:"matrix"`
	Factors []GaussianFactor `json:"factors"`
	Weights map[string]float64 `json:"weights"`
	Scores  map[string]float64 `json:"scores"`
}

// AbsoluteColumnInput is raw attribute values for one criterion.
type AbsoluteColumnInput struct {
	CriterionID string
	Prefer      PreferDirection
	Values      map[string]float64 // altID → raw value
}

// BuildAbsoluteMatrix builds a sum-normalized decision matrix.
// prefer=lower inverts via 1/v (errors on v<=0). prefer=higher allows zeros.
func BuildAbsoluteMatrix(altIDs []string, cols []AbsoluteColumnInput) (*AbsoluteMatrix, error) {
	if len(altIDs) < 2 {
		return nil, fmt.Errorf("need at least two alternatives")
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("need at least one attribute-ready criterion")
	}
	seenAlt := map[string]bool{}
	for _, id := range altIDs {
		if id == "" {
			return nil, fmt.Errorf("empty alternative id")
		}
		if seenAlt[id] {
			return nil, fmt.Errorf("duplicate alternative id %q", id)
		}
		seenAlt[id] = true
	}

	out := &AbsoluteMatrix{
		AlternativeIDs:  append([]string(nil), altIDs...),
		NormByCriterion: map[string]map[string]float64{},
	}
	seenCrit := map[string]bool{}
	for _, col := range cols {
		if col.CriterionID == "" {
			return nil, fmt.Errorf("empty criterion id")
		}
		if seenCrit[col.CriterionID] {
			return nil, fmt.Errorf("duplicate criterion id %q", col.CriterionID)
		}
		seenCrit[col.CriterionID] = true
		if col.Prefer != PreferHigher && col.Prefer != PreferLower {
			return nil, fmt.Errorf("criterion %s: prefer must be higher or lower", col.CriterionID)
		}
		raw := map[string]float64{}
		transformed := make([]float64, len(altIDs))
		for i, aid := range altIDs {
			v, ok := col.Values[aid]
			if !ok {
				return nil, fmt.Errorf("criterion %s: missing value for alternative %s", col.CriterionID, aid)
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("criterion %s: invalid value for %s", col.CriterionID, aid)
			}
			raw[aid] = v
			tv, err := transformAbsoluteValue(v, col.Prefer, col.CriterionID, aid)
			if err != nil {
				return nil, err
			}
			transformed[i] = tv
		}
		sum := 0.0
		for _, tv := range transformed {
			sum += tv
		}
		if sum <= 0 {
			return nil, fmt.Errorf("criterion %s: column sum is zero after transform", col.CriterionID)
		}
		norm := map[string]float64{}
		for i, aid := range altIDs {
			norm[aid] = transformed[i] / sum
		}
		out.Columns = append(out.Columns, AbsoluteColumn{
			CriterionID: col.CriterionID,
			Prefer:      col.Prefer,
			Raw:         raw,
			Norm:        norm,
		})
		out.NormByCriterion[col.CriterionID] = norm
	}
	return out, nil
}

func transformAbsoluteValue(v float64, prefer PreferDirection, critID, altID string) (float64, error) {
	switch prefer {
	case PreferHigher:
		if v < 0 {
			return 0, fmt.Errorf("criterion %s: prefer=higher rejects negative value for %s (got %g)", critID, altID, v)
		}
		return v, nil
	case PreferLower:
		if v <= 0 {
			return 0, fmt.Errorf("criterion %s: prefer=lower requires positive value for %s (got %g)", critID, altID, v)
		}
		return 1.0 / v, nil
	default:
		return 0, fmt.Errorf("criterion %s: invalid prefer %q", critID, prefer)
	}
}

// ScoreAbsolute scores alternatives with given criterion weights (must cover all columns).
// Weights are renormalized over the matrix columns that are present.
func ScoreAbsolute(m *AbsoluteMatrix, weights map[string]float64) (map[string]float64, error) {
	if m == nil {
		return nil, fmt.Errorf("nil absolute matrix")
	}
	if len(m.Columns) == 0 {
		return nil, fmt.Errorf("empty absolute matrix")
	}
	w := map[string]float64{}
	sum := 0.0
	for _, col := range m.Columns {
		wt, ok := weights[col.CriterionID]
		if !ok {
			return nil, fmt.Errorf("missing weight for criterion %s", col.CriterionID)
		}
		if wt < 0 || math.IsNaN(wt) || math.IsInf(wt, 0) {
			return nil, fmt.Errorf("invalid weight for criterion %s", col.CriterionID)
		}
		w[col.CriterionID] = wt
		sum += wt
	}
	if sum <= 0 {
		return nil, fmt.Errorf("criterion weights sum to zero")
	}
	for id := range w {
		w[id] /= sum
	}
	scores := map[string]float64{}
	for _, aid := range m.AlternativeIDs {
		s := 0.0
		for _, col := range m.Columns {
			s += w[col.CriterionID] * col.Norm[aid]
		}
		scores[aid] = s
	}
	return scores, nil
}

// GaussianFromAbsolute computes Santos AHP-Gaussian factors (sample SD) and scores.
func GaussianFromAbsolute(m *AbsoluteMatrix) (*GaussianResult, error) {
	if m == nil {
		return nil, fmt.Errorf("nil absolute matrix")
	}
	n := float64(len(m.AlternativeIDs))
	if n < 2 {
		return nil, fmt.Errorf("need at least two alternatives")
	}
	factors := make([]GaussianFactor, 0, len(m.Columns))
	rawFactors := make([]float64, 0, len(m.Columns))
	for _, col := range m.Columns {
		vals := make([]float64, 0, len(m.AlternativeIDs))
		sum := 0.0
		for _, aid := range m.AlternativeIDs {
			v := col.Norm[aid]
			vals = append(vals, v)
			sum += v
		}
		mean := sum / n
		if mean <= 0 {
			return nil, fmt.Errorf("criterion %s: mean of normalized column is zero", col.CriterionID)
		}
		sd := sampleSD(vals, mean)
		f := sd / mean
		factors = append(factors, GaussianFactor{
			CriterionID: col.CriterionID,
			Mean:        mean,
			SD:          sd,
			Factor:      f,
		})
		rawFactors = append(rawFactors, f)
	}
	fSum := 0.0
	for _, f := range rawFactors {
		fSum += f
	}
	if fSum <= 0 {
		return nil, fmt.Errorf("gaussian factors sum to zero (no dispersion across criteria)")
	}
	weights := map[string]float64{}
	for i := range factors {
		factors[i].Weight = factors[i].Factor / fSum
		weights[factors[i].CriterionID] = factors[i].Weight
	}
	scores, err := ScoreAbsolute(m, weights)
	if err != nil {
		return nil, err
	}
	return &GaussianResult{
		Matrix:  *m,
		Factors: factors,
		Weights: weights,
		Scores:  scores,
	}, nil
}

func sampleSD(vals []float64, mean float64) float64 {
	n := len(vals)
	if n < 2 {
		return 0
	}
	ss := 0.0
	for _, v := range vals {
		d := v - mean
		ss += d * d
	}
	return math.Sqrt(ss / float64(n-1))
}

// RankScores turns score map into sorted id list (desc).
func RankScores(scores map[string]float64) []string {
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] == scores[ids[j]] {
			return ids[i] < ids[j]
		}
		return scores[ids[i]] > scores[ids[j]]
	})
	return ids
}
