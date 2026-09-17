package engine

import (
	"fmt"
	"math"
	"sort"
)

// RankEntry is a compact ranking row used in sensitivity payloads.
type RankEntry struct {
	Rank   int     `json:"rank"`
	ID     string  `json:"id"`
	Name   string  `json:"name,omitempty"`
	Weight float64 `json:"weight"`
}

// PerturbImpact is the ranking after a one-at-a-time weight change.
type PerturbImpact struct {
	CriterionWeight float64     `json:"criterion_weight"`
	LeaderID        string      `json:"leader_id"`
	LeaderChanged   bool        `json:"leader_changed"`
	LeaderDeltaW    float64     `json:"leader_delta_weight"`
	Ranking         []RankEntry `json:"ranking"`
}

// CriterionSensitivity summarizes ±δ and rank-reversal for one leaf criterion.
type CriterionSensitivity struct {
	CriterionID     string         `json:"criterion_id"`
	CriterionName   string         `json:"criterion_name,omitempty"`
	BaseWeight      float64        `json:"base_weight"`
	Plus            PerturbImpact  `json:"plus_delta"`
	Minus           PerturbImpact  `json:"minus_delta"`
	TornadoEffect   float64        `json:"tornado_effect"`
	ReversalDelta   *float64       `json:"reversal_delta,omitempty"`
	ReversalWeight  *float64       `json:"reversal_weight,omitempty"`
	ReversalNote    string         `json:"reversal_note,omitempty"`
}

// SensitivityResult is a one-at-a-time leaf-weight robustness analysis.
type SensitivityResult struct {
	Delta         float64                 `json:"delta"`
	BaseLeaderID  string                  `json:"base_leader_id"`
	BaseLeaderName string                 `json:"base_leader_name,omitempty"`
	BaseRanking   []RankEntry             `json:"base_ranking"`
	ByCriterion   []CriterionSensitivity  `json:"by_criterion"`
	Summary       string                  `json:"summary"`
}

// AdjustLeafWeight sets criterion id to newW (clamped to [0,1]) and
// renormalizes the remaining leaf weights proportionally so they sum to 1.
func AdjustLeafWeight(weights map[string]float64, id string, newW float64) map[string]float64 {
	out := make(map[string]float64, len(weights))
	if len(weights) == 0 {
		return out
	}
	if newW < 0 {
		newW = 0
	}
	if newW > 1 {
		newW = 1
	}
	old := weights[id]
	sumOthers := 0.0
	for k, v := range weights {
		if k == id {
			continue
		}
		sumOthers += v
	}
	remain := 1 - newW
	for k, v := range weights {
		if k == id {
			out[k] = newW
			continue
		}
		if sumOthers <= 1e-15 {
			// Spread evenly if others were ~0.
			n := float64(len(weights) - 1)
			if n <= 0 {
				out[k] = 0
			} else {
				out[k] = remain / n
			}
			continue
		}
		out[k] = v * (remain / sumOthers)
	}
	_ = old
	return out
}

// RankFromWeights builds a sorted ranking from synthesized global weights.
func RankFromWeights(globals map[string]float64, altIDs []string, altNames map[string]string) []RankEntry {
	type kv struct {
		id string
		w  float64
	}
	rows := make([]kv, 0, len(altIDs))
	for _, id := range altIDs {
		rows = append(rows, kv{id, globals[id]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].w > rows[j].w })
	out := make([]RankEntry, 0, len(rows))
	for i, r := range rows {
		name := r.id
		if altNames != nil && altNames[r.id] != "" {
			name = altNames[r.id]
		}
		out = append(out, RankEntry{Rank: i + 1, ID: r.id, Name: name, Weight: r.w})
	}
	return out
}

func impactAt(
	leafWeights map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames map[string]string,
	critID string,
	newW float64,
	baseLeader string,
	baseLeaderW float64,
) PerturbImpact {
	adj := AdjustLeafWeight(leafWeights, critID, newW)
	globals := Synthesize(adj, local, altIDs)
	ranking := RankFromWeights(globals, altIDs, altNames)
	leader := ""
	if len(ranking) > 0 {
		leader = ranking[0].ID
	}
	leaderW := globals[baseLeader]
	return PerturbImpact{
		CriterionWeight: newW,
		LeaderID:        leader,
		LeaderChanged:   leader != "" && leader != baseLeader,
		LeaderDeltaW:    leaderW - baseLeaderW,
		Ranking:         ranking,
	}
}

// Sensitivity runs ±delta one-at-a-time perturbations and estimates the
// smallest positive weight shift on each criterion that flips the leader.
func Sensitivity(
	leafWeights map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames, critNames map[string]string,
	delta float64,
) SensitivityResult {
	if delta <= 0 {
		delta = 0.05
	}
	baseGlobals := Synthesize(leafWeights, local, altIDs)
	baseRanking := RankFromWeights(baseGlobals, altIDs, altNames)
	baseLeader := ""
	baseLeaderName := ""
	baseLeaderW := 0.0
	if len(baseRanking) > 0 {
		baseLeader = baseRanking[0].ID
		baseLeaderName = baseRanking[0].Name
		baseLeaderW = baseRanking[0].Weight
	}

	critOrder := make([]string, 0, len(leafWeights))
	for cid := range leafWeights {
		critOrder = append(critOrder, cid)
	}
	sort.Strings(critOrder)

	byCrit := make([]CriterionSensitivity, 0, len(critOrder))
	flipCount := 0
	for _, cid := range critOrder {
		baseW := leafWeights[cid]
		plusW := math.Min(1, baseW+delta)
		minusW := math.Max(0, baseW-delta)
		plus := impactAt(leafWeights, local, altIDs, altNames, cid, plusW, baseLeader, baseLeaderW)
		minus := impactAt(leafWeights, local, altIDs, altNames, cid, minusW, baseLeader, baseLeaderW)
		tornado := math.Max(math.Abs(plus.LeaderDeltaW), math.Abs(minus.LeaderDeltaW))
		if plus.LeaderChanged || minus.LeaderChanged {
			flipCount++
		}

		name := cid
		if critNames != nil && critNames[cid] != "" {
			name = critNames[cid]
		}
		cs := CriterionSensitivity{
			CriterionID: cid, CriterionName: name, BaseWeight: baseW,
			Plus: plus, Minus: minus, TornadoEffect: tornado,
		}
		revDelta, revW, note := findReversal(leafWeights, local, altIDs, altNames, cid, baseW, baseLeader)
		if revDelta != nil {
			cs.ReversalDelta = revDelta
			cs.ReversalWeight = revW
			cs.ReversalNote = note
		} else if note != "" {
			cs.ReversalNote = note
		}
		byCrit = append(byCrit, cs)
	}

	sort.Slice(byCrit, func(i, j int) bool {
		if byCrit[i].TornadoEffect == byCrit[j].TornadoEffect {
			return byCrit[i].CriterionID < byCrit[j].CriterionID
		}
		return byCrit[i].TornadoEffect > byCrit[j].TornadoEffect
	})

	summary := fmt.Sprintf("leader=%s; ±%.2f flips on %d/%d criteria",
		baseLeaderName, delta, flipCount, len(byCrit))
	if baseLeaderName == "" {
		summary = "no ranking to analyze"
	}

	return SensitivityResult{
		Delta: delta, BaseLeaderID: baseLeader, BaseLeaderName: baseLeaderName,
		BaseRanking: baseRanking, ByCriterion: byCrit, Summary: summary,
	}
}

// findReversal searches for the smallest |Δw| that changes the leader when
// adjusting criterion cid. Prefers the direction toward the challenger.
func findReversal(
	leafWeights map[string]float64,
	local map[string]map[string]float64,
	altIDs []string,
	altNames map[string]string,
	cid string,
	baseW float64,
	baseLeader string,
) (delta *float64, weight *float64, note string) {
	if baseLeader == "" || len(altIDs) < 2 {
		return nil, nil, "need a ranked leader"
	}
	// Coarse scan then refine.
	const steps = 40
	type hit struct {
		w, d float64
	}
	var hits []hit
	for i := 0; i <= steps; i++ {
		w := float64(i) / float64(steps)
		adj := AdjustLeafWeight(leafWeights, cid, w)
		ranking := RankFromWeights(Synthesize(adj, local, altIDs), altIDs, altNames)
		if len(ranking) == 0 || ranking[0].ID == baseLeader {
			continue
		}
		hits = append(hits, hit{w: w, d: math.Abs(w - baseW)})
	}
	if len(hits) == 0 {
		return nil, nil, "no leader flip in [0,1] for this criterion"
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if h.d < best.d-1e-12 || (math.Abs(h.d-best.d) < 1e-12 && math.Abs(h.w-baseW) < math.Abs(best.w-baseW)) {
			best = h
		}
	}
	// Refine around best.w with binary search on the side away from base.
	lo, hi := baseW, best.w
	if lo > hi {
		lo, hi = hi, lo
	}
	// Ensure lo keeps leader, hi flips (expand if needed).
	for iter := 0; iter < 24; iter++ {
		mid := (lo + hi) / 2
		adj := AdjustLeafWeight(leafWeights, cid, mid)
		ranking := RankFromWeights(Synthesize(adj, local, altIDs), altIDs, altNames)
		flipped := len(ranking) > 0 && ranking[0].ID != baseLeader
		if best.w >= baseW {
			// Increasing weight flips.
			if flipped {
				hi = mid
			} else {
				lo = mid
			}
		} else {
			// Decreasing weight flips.
			if flipped {
				lo = mid
			} else {
				hi = mid
			}
		}
	}
	flipW := hi
	if best.w < baseW {
		flipW = lo
	}
	d := flipW - baseW
	return &d, &flipW, fmt.Sprintf("leader flips when %s weight → %.3f (Δ=%+.3f)", cid, flipW, d)
}
