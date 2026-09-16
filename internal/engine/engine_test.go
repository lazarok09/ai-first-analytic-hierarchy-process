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

func approx(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-5*math.Max(1, math.Abs(want)) {
		t.Fatalf("got %v want %v", got, want)
	}
}
