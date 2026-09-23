package engine

import (
	"math"
	"testing"
)

func TestBuildAbsoluteMatrixHigherAndLower(t *testing.T) {
	alts := []string{"a", "b", "c"}
	m, err := BuildAbsoluteMatrix(alts, []AbsoluteColumnInput{
		{CriterionID: "score", Prefer: PreferHigher, Values: map[string]float64{"a": 1, "b": 2, "c": 3}},
		{CriterionID: "price", Prefer: PreferLower, Values: map[string]float64{"a": 100, "b": 200, "c": 200}},
	})
	if err != nil {
		t.Fatal(err)
	}
	approx(t, m.NormByCriterion["score"]["a"], 1.0/6)
	approx(t, m.NormByCriterion["score"]["c"], 0.5)
	// price: 1/100, 1/200, 1/200 → sum 0.02; a gets 0.5
	approx(t, m.NormByCriterion["price"]["a"], 0.5)
	approx(t, m.NormByCriterion["price"]["b"], 0.25)
}

func TestBuildAbsoluteMatrixBenefitZeroOK(t *testing.T) {
	m, err := BuildAbsoluteMatrix([]string{"a", "b"}, []AbsoluteColumnInput{
		{CriterionID: "feature", Prefer: PreferHigher, Values: map[string]float64{"a": 0, "b": 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	approx(t, m.NormByCriterion["feature"]["a"], 0)
	approx(t, m.NormByCriterion["feature"]["b"], 1)
}

func TestBuildAbsoluteMatrixCostZeroFails(t *testing.T) {
	_, err := BuildAbsoluteMatrix([]string{"a", "b"}, []AbsoluteColumnInput{
		{CriterionID: "price", Prefer: PreferLower, Values: map[string]float64{"a": 0, "b": 10}},
	})
	if err == nil {
		t.Fatal("expected error for prefer=lower with zero")
	}
}

func TestBuildAbsoluteMatrixBenefitNegativeFails(t *testing.T) {
	_, err := BuildAbsoluteMatrix([]string{"a", "b"}, []AbsoluteColumnInput{
		{CriterionID: "score", Prefer: PreferHigher, Values: map[string]float64{"a": -1, "b": 2}},
	})
	if err == nil {
		t.Fatal("expected error for prefer=higher with negative")
	}
}

func TestGaussianGenericMixed(t *testing.T) {
	m, err := BuildAbsoluteMatrix([]string{"x", "y", "z"}, []AbsoluteColumnInput{
		{CriterionID: "quality", Prefer: PreferHigher, Values: map[string]float64{"x": 8, "y": 9, "z": 7}},
		{CriterionID: "cost", Prefer: PreferLower, Values: map[string]float64{"x": 100, "y": 120, "z": 90}},
	})
	if err != nil {
		t.Fatal(err)
	}
	g, err := GaussianFromAbsolute(m)
	if err != nil {
		t.Fatal(err)
	}
	wSum := 0.0
	for _, w := range g.Weights {
		wSum += w
	}
	approx(t, wSum, 1)
	sSum := 0.0
	for _, s := range g.Scores {
		sSum += s
	}
	approx(t, sSum, 1)
	ranked := RankScores(g.Scores)
	if len(ranked) != 3 {
		t.Fatalf("rank len=%d", len(ranked))
	}
}

// Paper Tables 9 / 15–17 numeric lock (Santos et al. 2021). Fixture only — not a product domain.
func TestGaussianPaperNumericFixture(t *testing.T) {
	alts := []string{"m1", "m2", "m3"}
	cols := []AbsoluteColumnInput{
		{CriterionID: "c1", Prefer: PreferHigher, Values: map[string]float64{"m1": 4000, "m2": 9330, "m3": 10660}},
		{CriterionID: "c2", Prefer: PreferHigher, Values: map[string]float64{"m1": 11, "m2": 26, "m3": 30}},
		{CriterionID: "c3", Prefer: PreferHigher, Values: map[string]float64{"m1": 30, "m2": 25, "m3": 35}},
		{CriterionID: "c4", Prefer: PreferHigher, Values: map[string]float64{"m1": 25, "m2": 25, "m3": 120}},
		{CriterionID: "c5", Prefer: PreferHigher, Values: map[string]float64{"m1": 1, "m2": 2, "m3": 2}},
		{CriterionID: "c6", Prefer: PreferHigher, Values: map[string]float64{"m1": 0, "m2": 1, "m3": 1}},
		{CriterionID: "c7", Prefer: PreferLower, Values: map[string]float64{"m1": 290e6, "m2": 310e6, "m3": 310e6}},
		{CriterionID: "c8", Prefer: PreferLower, Values: map[string]float64{"m1": 592e6, "m2": 633e6, "m3": 633e6}},
		{CriterionID: "c9", Prefer: PreferLower, Values: map[string]float64{"m1": 6, "m2": 8, "m3": 8}},
	}
	m, err := BuildAbsoluteMatrix(alts, cols)
	if err != nil {
		t.Fatal(err)
	}
	// Table 10 spot checks
	approxTol(t, m.NormByCriterion["c1"]["m1"], 0.1667, 5e-4)
	approxTol(t, m.NormByCriterion["c6"]["m1"], 0, 1e-9)
	approxTol(t, m.NormByCriterion["c6"]["m2"], 0.5, 1e-9)
	approxTol(t, m.NormByCriterion["c9"]["m1"], 0.4, 1e-3)

	g, err := GaussianFromAbsolute(m)
	if err != nil {
		t.Fatal(err)
	}
	// Table 17
	approxTol(t, g.Scores["m3"], 0.5144, 5e-3)
	approxTol(t, g.Scores["m2"], 0.3390, 5e-3)
	approxTol(t, g.Scores["m1"], 0.1465, 5e-3)
	if RankScores(g.Scores)[0] != "m3" {
		t.Fatalf("expected m3 first, got %v", RankScores(g.Scores))
	}
}

func TestScoreAbsoluteWithExpertWeights(t *testing.T) {
	m, err := BuildAbsoluteMatrix([]string{"a", "b"}, []AbsoluteColumnInput{
		{CriterionID: "q", Prefer: PreferHigher, Values: map[string]float64{"a": 1, "b": 3}},
		{CriterionID: "p", Prefer: PreferLower, Values: map[string]float64{"a": 10, "b": 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	scores, err := ScoreAbsolute(m, map[string]float64{"q": 0.7, "p": 0.3})
	if err != nil {
		t.Fatal(err)
	}
	// q norms: 0.25, 0.75; p: 1/10 vs 1/20 → 2/3, 1/3
	approx(t, scores["a"], 0.7*0.25+0.3*(2.0/3))
	approx(t, scores["b"], 0.7*0.75+0.3*(1.0/3))
}

func approxTol(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("got %v want %v (tol %v)", got, want, tol)
	}
}
