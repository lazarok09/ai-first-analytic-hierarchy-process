package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// PreferDirection controls whether larger or smaller numeric attributes win.
type PreferDirection string

const (
	PreferHigher PreferDirection = "higher"
	PreferLower  PreferDirection = "lower"
)

// AttrPairSuggestion is one Saaty judgment derived from objective attribute values.
type AttrPairSuggestion struct {
	Left       string  `json:"left"`
	Right      string  `json:"right"`
	Value      float64 `json:"value"`
	ValueLabel string  `json:"value_label"`
	LeftValue  float64 `json:"left_value"`
	RightValue float64 `json:"right_value"`
	RawRatio   float64 `json:"raw_ratio"`
	Note       string  `json:"note"`
}

// SuggestFromValuesOptions controls ratio→Saaty mapping from numeric attributes.
type SuggestFromValuesOptions struct {
	HigherBetter bool
	// Stretch applies an affine preference-preserving transform before ratios:
	//   higher-better: v' = v − min + 1
	//   lower-better:  v' = max − v + 1
	// so tight bands (e.g. guest scores 8.4 vs 9.5) discriminate on the Saaty scale.
	// Non-positive values always trigger this transform (even when Stretch is false)
	// so amenity counts of 0 are valid.
	Stretch bool
}

// ParseNumericAttribute parses a CSV attribute cell into a float.
func ParseNumericAttribute(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, fmt.Errorf("empty numeric attribute")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("not numeric %q: %w", raw, err)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid numeric %q", raw)
	}
	return v, nil
}

// SuggestPairwiseFromValues maps numeric attribute values to Saaty pairwise
// intensities via ratios (nearest Saaty). Non-positive values are allowed
// (affine-shifted). Prefer SuggestPairwiseFromValuesOpts for --stretch.
func SuggestPairwiseFromValues(ids []string, values map[string]float64, higherBetter bool) ([]AttrPairSuggestion, error) {
	return SuggestPairwiseFromValuesOpts(ids, values, SuggestFromValuesOptions{HigherBetter: higherBetter})
}

// SuggestPairwiseFromValuesOpts is the full attribute→Saaty bridge.
func SuggestPairwiseFromValuesOpts(ids []string, values map[string]float64, opts SuggestFromValuesOptions) ([]AttrPairSuggestion, error) {
	if len(ids) < 2 {
		return nil, fmt.Errorf("need at least two alternatives with numeric attributes")
	}
	for _, id := range ids {
		if _, ok := values[id]; !ok {
			return nil, fmt.Errorf("missing numeric value for %s", id)
		}
	}

	transformed, transformNote, err := prepareAttributeValues(ids, values, opts)
	if err != nil {
		return nil, err
	}

	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)

	var out []AttrPairSuggestion
	for i, left := range ordered {
		for _, right := range ordered[i+1:] {
			lv, rv := transformed[left], transformed[right]
			// After affine transform, larger transformed value is always preferred.
			raw := lv / rv
			if raw > 9 {
				raw = 9
			} else if raw < 1.0/9 {
				raw = 1.0 / 9
			}
			saaty := NearestSaaty(raw)
			dir := "higher-better"
			if !opts.HigherBetter {
				dir = "lower-better"
			}
			note := fmt.Sprintf("from attributes (%s): %s=%.4g vs %s=%.4g → pref ratio %.3g → Saaty %s",
				dir, left, values[left], right, values[right], raw, FormatSaaty(saaty))
			if transformNote != "" {
				note = note + "; " + transformNote
			}
			out = append(out, AttrPairSuggestion{
				Left: left, Right: right,
				Value: saaty, ValueLabel: FormatSaaty(saaty),
				LeftValue: values[left], RightValue: values[right], RawRatio: raw, Note: note,
			})
		}
	}
	return out, nil
}

// prepareAttributeValues optionally affine-shifts values so ratios discriminate
// and non-positive counts are valid. Returns working values + note fragment.
func prepareAttributeValues(ids []string, values map[string]float64, opts SuggestFromValuesOptions) (map[string]float64, string, error) {
	minV, maxV := values[ids[0]], values[ids[0]]
	hasNonPos := false
	for _, id := range ids {
		v := values[id]
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
		if v <= 0 {
			hasNonPos = true
		}
	}

	needShift := opts.Stretch || hasNonPos
	if !needShift {
		// Classic ratio on raw values (lower-better flips via reciprocal preference).
		out := make(map[string]float64, len(ids))
		for _, id := range ids {
			v := values[id]
			if v <= 0 {
				return nil, "", fmt.Errorf("attribute value for %s must be > 0 (got %v)", id, v)
			}
			if opts.HigherBetter {
				out[id] = v
			} else {
				// Encode lower-better as higher preference weight = 1/v.
				out[id] = 1 / v
			}
		}
		return out, "", nil
	}

	out := make(map[string]float64, len(ids))
	for _, id := range ids {
		v := values[id]
		if opts.HigherBetter {
			out[id] = v - minV + 1
		} else {
			out[id] = maxV - v + 1
		}
		if out[id] <= 0 {
			return nil, "", fmt.Errorf("internal: non-positive transformed value for %s", id)
		}
	}
	note := "affine stretch"
	if hasNonPos && !opts.Stretch {
		note = "affine shift for non-positive values"
	} else if hasNonPos && opts.Stretch {
		note = "affine stretch (incl. non-positive)"
	}
	return out, note, nil
}
