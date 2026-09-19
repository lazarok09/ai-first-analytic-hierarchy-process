package engine

import (
	"math"
	"testing"
)

func TestConsistent3x3(t *testing.T) {
	ids := []string{"cost", "quality", "risk"}
	pairs := map[PairKey]float64{
		{Left: "cost", Right: "quality"}: 2,
		{Left: "cost", Right: "risk"}:    6,
		{Left: "quality", Right: "risk"}: 3,
	}
	r := SolvePairwise(ids, pairs)
	if !r.Complete {
		t.Fatal("expected complete")
	}
	if r.CR > 1e-6 {
		t.Fatalf("CR=%v", r.CR)
	}
	approx(t, r.Weights["cost"], 0.6)
	approx(t, r.Weights["quality"], 0.3)
	approx(t, r.Weights["risk"], 0.1)
}

func TestIncomplete(t *testing.T) {
	r := SolvePairwise([]string{"a", "b", "c"}, map[PairKey]float64{{Left: "a", Right: "b"}: 3})
	if r.Complete {
		t.Fatal("expected incomplete")
	}
	if len(r.Missing) != 2 {
		t.Fatalf("missing=%v", r.Missing)
	}
	approx(t, r.Weights["a"], 1.0/3)
}

func TestSynthesis(t *testing.T) {
	scores := Synthesize(
		map[string]float64{"cost": 0.6, "quality": 0.4},
		map[string]map[string]float64{
			"cost":    {"x": 0.7, "y": 0.3},
			"quality": {"x": 0.2, "y": 0.8},
		},
		[]string{"x", "y"},
	)
	approx(t, scores["x"], 0.6*0.7+0.4*0.2)
	approx(t, scores["y"], 0.6*0.3+0.4*0.8)
}

func TestInconsistentRepairs(t *testing.T) {
	ids := []string{"a", "b", "c"}
	pairs := map[PairKey]float64{
		{Left: "a", Right: "b"}: 5,
		{Left: "b", Right: "c"}: 5,
		{Left: "a", Right: "c"}: 1,
	}
	r := SolvePairwise(ids, pairs)
	if r.CR <= 0.10 {
		t.Fatalf("expected inconsistent CR=%v", r.CR)
	}
	if len(r.Repairs) == 0 {
		t.Fatal("expected repairs")
	}
}

func TestParseSaaty(t *testing.T) {
	v, err := ParseSaaty("1/5")
	if err != nil {
		t.Fatal(err)
	}
	approx(t, v, 0.2)
	if FormatSaaty(0.2) != "1/5" {
		t.Fatalf("format=%s", FormatSaaty(0.2))
	}
}

func TestExplainContributions(t *testing.T) {
	leaf := map[string]float64{"cost": 0.6, "quality": 0.4}
	local := map[string]map[string]float64{
		"cost":    {"x": 0.7, "y": 0.3},
		"quality": {"x": 0.2, "y": 0.8},
	}
	exp := Explain(leaf, local, []string{"x", "y"},
		map[string]string{"x": "X", "y": "Y"},
		map[string]string{"cost": "Cost", "quality": "Quality"})
	if len(exp.Alternatives) != 2 {
		t.Fatalf("alts=%d", len(exp.Alternatives))
	}
	x := exp.Alternatives[0]
	if x.ID != "x" || x.Rank != 1 {
		t.Fatalf("expected x rank1 got %+v", x)
	}
	approx(t, x.GlobalWeight, 0.6*0.7+0.4*0.2)
	if len(x.Contributions) != 2 {
		t.Fatalf("contributions=%d", len(x.Contributions))
	}
	approx(t, x.Contributions[0].Contribution, 0.6*0.7) // cost dominates for x
	approx(t, x.Contributions[0].Contribution+x.Contributions[1].Contribution, x.GlobalWeight)
}

func TestAdjustLeafWeightRenormalizes(t *testing.T) {
	w := map[string]float64{"a": 0.5, "b": 0.3, "c": 0.2}
	adj := AdjustLeafWeight(w, "a", 0.7)
	approx(t, adj["a"], 0.7)
	sum := adj["a"] + adj["b"] + adj["c"]
	approx(t, sum, 1)
	approx(t, adj["b"]/adj["c"], 0.3/0.2)
}

func TestSensitivityTornadoAndFlip(t *testing.T) {
	// isolation-heavy like the headphone dogfood: flip when isolation softens.
	leaf := map[string]float64{"isolation": 0.5, "value": 0.3, "comfort": 0.2}
	local := map[string]map[string]float64{
		"isolation": {"closed": 0.7, "open": 0.3},
		"value":     {"closed": 0.3, "open": 0.7},
		"comfort":   {"closed": 0.4, "open": 0.6},
	}
	alts := []string{"closed", "open"}
	names := map[string]string{"closed": "Closed", "open": "Open"}
	sens := Sensitivity(leaf, local, alts, names, nil, 0.20)
	if sens.BaseLeaderID != "closed" {
		t.Fatalf("leader=%s", sens.BaseLeaderID)
	}
	if len(sens.ByCriterion) != 3 {
		t.Fatalf("criteria=%d", len(sens.ByCriterion))
	}
	// Largest tornado should involve isolation or value (discrimination).
	if sens.ByCriterion[0].TornadoEffect < sens.ByCriterion[2].TornadoEffect {
		t.Fatalf("tornado not sorted: %+v", sens.ByCriterion)
	}
	var iso *CriterionSensitivity
	for i := range sens.ByCriterion {
		if sens.ByCriterion[i].CriterionID == "isolation" {
			iso = &sens.ByCriterion[i]
			break
		}
	}
	if iso == nil {
		t.Fatal("missing isolation")
	}
	if iso.ReversalDelta == nil {
		t.Fatalf("expected isolation reversal, note=%s", iso.ReversalNote)
	}
}

func TestSuggestPairwiseFromValuesLowerBetter(t *testing.T) {
	ids := []string{"cheap", "mid", "pricey"}
	vals := map[string]float64{"cheap": 100, "mid": 200, "pricey": 400}
	sug, err := SuggestPairwiseFromValues(ids, vals, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sug) != 3 {
		t.Fatalf("pairs=%d", len(sug))
	}
	by := map[string]AttrPairSuggestion{}
	for _, s := range sug {
		by[s.Left+"|"+s.Right] = s
	}
	// cheap vs pricey: 400/100=4 → Saaty 4 (cheap preferred)
	cp := by["cheap|pricey"]
	if cp.Value != 4 {
		t.Fatalf("cheap|pricey=%v label=%s", cp.Value, cp.ValueLabel)
	}
	// mid vs pricey: 400/200=2 → 2
	if by["mid|pricey"].Value != 2 {
		t.Fatalf("mid|pricey=%v", by["mid|pricey"].Value)
	}
}

func TestSuggestPairwiseAllowsZeroAmenityCounts(t *testing.T) {
	// Hotel dogfood: amenities 0 vs 1 must not error.
	ids := []string{"none", "pool"}
	vals := map[string]float64{"none": 0, "pool": 1}
	sug, err := SuggestPairwiseFromValues(ids, vals, true)
	if err != nil {
		t.Fatalf("zero amenity should be allowed: %v", err)
	}
	if len(sug) != 1 {
		t.Fatalf("pairs=%d", len(sug))
	}
	// Ordered alphabetically: none|pool. After shift none=1, pool=2 → pref ratio 0.5
	// (pool better → Saaty 1/2 when left is worse).
	if sug[0].Left != "none" || sug[0].Right != "pool" {
		t.Fatalf("unexpected order %s|%s", sug[0].Left, sug[0].Right)
	}
	if sug[0].Value != 0.5 {
		t.Fatalf("none|pool=%v want 1/2 (note=%s)", sug[0].Value, sug[0].Note)
	}
}

func TestSuggestPairwiseStretchDiscriminatesTightScores(t *testing.T) {
	ids := []string{"ok", "great"}
	vals := map[string]float64{"ok": 8.4, "great": 9.5}
	flat, err := SuggestPairwiseFromValues(ids, vals, true)
	if err != nil {
		t.Fatal(err)
	}
	if flat[0].Value != 1 {
		t.Fatalf("without stretch expected Saaty 1, got %v", flat[0].Value)
	}
	spread, err := SuggestPairwiseFromValuesOpts(ids, vals, SuggestFromValuesOptions{
		HigherBetter: true, Stretch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// stretch: ok=1, great=2.1 → ratio≈2.1 → Saaty 2
	if spread[0].Value != 2 {
		t.Fatalf("with stretch expected Saaty 2, got %v note=%s", spread[0].Value, spread[0].Note)
	}
}

func approx(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-5*math.Max(1, math.Abs(want)) {
		t.Fatalf("got %v want %v", got, want)
	}
}
