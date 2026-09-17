package engine

import "sort"

// Contribution is one criterion's share of an alternative's global weight.
type Contribution struct {
	CriterionID     string  `json:"criterion_id"`
	CriterionName   string  `json:"criterion_name,omitempty"`
	CriterionWeight float64 `json:"criterion_weight"`
	LocalWeight     float64 `json:"local_weight"`
	Contribution    float64 `json:"contribution"`
	Share           float64 `json:"share"`
}

// AltExplain breaks down one alternative's global score by leaf criterion.
type AltExplain struct {
	ID            string         `json:"id"`
	Name          string         `json:"name,omitempty"`
	Rank          int            `json:"rank"`
	GlobalWeight  float64        `json:"global_weight"`
	Contributions []Contribution `json:"contributions"`
}

// ExplainResult is the decision-quality breakdown for one or more alternatives.
type ExplainResult struct {
	Alternatives []AltExplain `json:"alternatives"`
}

// Explain computes local×criterion contributions for each alternative.
// leafWeights keys are leaf criterion ids; local[crit][alt] are local priorities.
// critNames / altNames are optional display maps.
func Explain(
	leafWeights map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames, critNames map[string]string,
) ExplainResult {
	globals := Synthesize(leafWeights, local, altIDs)
	type kv struct {
		id string
		w  float64
	}
	ranked := make([]kv, 0, len(altIDs))
	for _, id := range altIDs {
		ranked = append(ranked, kv{id, globals[id]})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].w > ranked[j].w })
	rankOf := map[string]int{}
	for i, r := range ranked {
		rankOf[r.id] = i + 1
	}

	critOrder := make([]string, 0, len(leafWeights))
	for cid := range leafWeights {
		critOrder = append(critOrder, cid)
	}
	sort.Strings(critOrder)

	out := ExplainResult{Alternatives: make([]AltExplain, 0, len(altIDs))}
	for _, alt := range altIDs {
		g := globals[alt]
		conts := make([]Contribution, 0, len(critOrder))
		for _, cid := range critOrder {
			cw := leafWeights[cid]
			lw := 0.0
			if loc := local[cid]; loc != nil {
				lw = loc[alt]
			}
			c := cw * lw
			share := 0.0
			if g > 0 {
				share = c / g
			}
			name := cid
			if critNames != nil && critNames[cid] != "" {
				name = critNames[cid]
			}
			conts = append(conts, Contribution{
				CriterionID: cid, CriterionName: name,
				CriterionWeight: cw, LocalWeight: lw,
				Contribution: c, Share: share,
			})
		}
		sort.Slice(conts, func(i, j int) bool {
			if conts[i].Contribution == conts[j].Contribution {
				return conts[i].CriterionID < conts[j].CriterionID
			}
			return conts[i].Contribution > conts[j].Contribution
		})
		aname := alt
		if altNames != nil && altNames[alt] != "" {
			aname = altNames[alt]
		}
		out.Alternatives = append(out.Alternatives, AltExplain{
			ID: alt, Name: aname, Rank: rankOf[alt],
			GlobalWeight: g, Contributions: conts,
		})
	}
	sort.Slice(out.Alternatives, func(i, j int) bool {
		return out.Alternatives[i].Rank < out.Alternatives[j].Rank
	})
	return out
}
