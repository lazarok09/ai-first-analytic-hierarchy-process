package workspace

import (
	"path/filepath"
	"testing"

	"github.com/lazarok09/ahp-method/internal/engine"
)

func TestExplainAndSensitivityVendor(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "vendor-selection")
	ws := Open(root)
	if _, err := ws.LoadMeta(); err != nil {
		t.Skip("vendor example missing")
	}
	exp, err := ws.Explain(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Alternatives) == 0 {
		t.Fatal("expected alternatives")
	}
	sum := 0.0
	for _, c := range exp.Alternatives[0].Contributions {
		sum += c.Contribution
	}
	if absF(sum-exp.Alternatives[0].GlobalWeight) > 1e-6 {
		t.Fatalf("contributions %.6f != global %.6f", sum, exp.Alternatives[0].GlobalWeight)
	}

	sens, err := ws.Sensitivity(false, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	if sens.BaseLeaderID == "" {
		t.Fatal("expected leader")
	}
	if len(sens.ByCriterion) == 0 {
		t.Fatal("expected criteria")
	}
}

func TestSuggestFromAttributesDryRun(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "t"}); err != nil {
		t.Fatal(err)
	}
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(Alternative{ID: "b", Name: "B"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "a", CriterionID: "value", Value: "100", Unit: "BRL"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "b", CriterionID: "value", Value: "300", Unit: "BRL"})

	res, err := ws.SuggestFromAttributes(SuggestFromAttributesOptions{
		CriterionID: "value", Prefer: engine.PreferLower, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 || len(res.Suggestions) != 1 {
		t.Fatalf("got %+v", res)
	}
	if res.Suggestions[0].Value != 3 {
		t.Fatalf("expected Saaty 3 for 300/100, got %v", res.Suggestions[0].Value)
	}
	pairs, _ := ws.Pairwise()
	if len(pairs) != 0 {
		t.Fatal("dry-run must not write")
	}
}

func TestSuggestFromAttributesRefreshDemotesCommitted(t *testing.T) {
	dir := t.TempDir()
	ws := Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.SaveMeta(Meta{Title: "t"}); err != nil {
		t.Fatal(err)
	}
	_ = ws.UpsertCriterion(Criterion{ID: "value", Name: "Value"})
	_ = ws.UpsertAlternative(Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(Alternative{ID: "b", Name: "B"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "a", CriterionID: "value", Value: "100", Unit: "BRL"})
	_ = ws.UpsertAttribute(AttributeRow{AlternativeID: "b", CriterionID: "value", Value: "300", Unit: "BRL"})
	_ = ws.UpsertPairwise(PairwiseRow{
		Matrix: "alt:value", Left: "a", Right: "b", Value: 1, Status: "committed", Note: "stale",
	})

	skipped, err := ws.SuggestFromAttributes(SuggestFromAttributesOptions{
		CriterionID: "value", Prefer: engine.PreferLower,
	})
	if err != nil {
		t.Fatal(err)
	}
	if skipped.Written != 0 || skipped.Skipped != 1 {
		t.Fatalf("default must skip committed, got %+v", skipped)
	}

	refreshed, err := ws.SuggestFromAttributes(SuggestFromAttributesOptions{
		CriterionID: "value", Prefer: engine.PreferLower, Refresh: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Written != 1 || refreshed.Skipped != 0 || refreshed.Refreshed != 1 {
		t.Fatalf("refresh should demote committed→proposal, got %+v", refreshed)
	}
	pairs, err := ws.Pairwise()
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Status != "proposal" || pairs[0].Value != 3 {
		t.Fatalf("expected proposal Saaty 3, got %+v", pairs[0])
	}
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
