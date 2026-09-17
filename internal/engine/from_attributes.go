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
// intensities via ratios (nearest Saaty). Values must be > 0.
// higherBetter=true means larger values are preferred; false means smaller.
func SuggestPairwiseFromValues(ids []string, values map[string]float64, higherBetter bool) ([]AttrPairSuggestion, error) {
	if len(ids) < 2 {
		return nil, fmt.Errorf("need at least two alternatives with numeric attributes")
	}
	for _, id := range ids {
		v, ok := values[id]
		if !ok {
			return nil, fmt.Errorf("missing numeric value for %s", id)
		}
		if v <= 0 {
			return nil, fmt.Errorf("attribute value for %s must be > 0 (got %v)", id, v)
		}
	}
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)

	var out []AttrPairSuggestion
	for i, left := range ordered {
		for _, right := range ordered[i+1:] {
			lv, rv := values[left], values[right]
			var raw float64
			if higherBetter {
				raw = lv / rv
			} else {
				raw = rv / lv
			}
			// Cap extreme ratios before Saaty snap so 100× price gaps → 9, not garbage.
			if raw > 9 {
				raw = 9
			} else if raw < 1.0/9 {
				raw = 1.0 / 9
			}
			saaty := NearestSaaty(raw)
			dir := "higher-better"
			if !higherBetter {
				dir = "lower-better"
			}
			note := fmt.Sprintf("from attributes (%s): %s=%.4g vs %s=%.4g → ratio %.3g → Saaty %s",
				dir, left, lv, right, rv, lv/rv, FormatSaaty(saaty))
			if !higherBetter {
				note = fmt.Sprintf("from attributes (%s): %s=%.4g vs %s=%.4g → pref ratio %.3g → Saaty %s",
					dir, left, lv, right, rv, raw, FormatSaaty(saaty))
			}
			out = append(out, AttrPairSuggestion{
				Left: left, Right: right,
				Value: saaty, ValueLabel: FormatSaaty(saaty),
				LeftValue: lv, RightValue: rv, RawRatio: raw, Note: note,
			})
		}
	}
	return out, nil
}
