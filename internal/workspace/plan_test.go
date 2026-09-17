package workspace_test

import (
	"testing"

	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestPlanShowsProposalDeltas(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	must(t, ws.EnsureLayout())
	must(t, ws.SaveMeta(workspace.Meta{Title: "Plan demo"}))
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"}))
	must(t, ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "acme", Name: "Acme"}))
	must(t, ws.UpsertAlternative(workspace.Alternative{ID: "globex", Name: "Globex"}))

	// Committed: cost slightly preferred; alts equal under each.
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "criteria", Left: "cost", Right: "quality", Value: 2, Status: "committed",
	}))
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:cost", Left: "acme", Right: "globex", Value: 1, Status: "committed",
	}))
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 1, Status: "committed",
	}))

	// Proposal would strongly prefer acme under quality (changes ranking weights).
	must(t, ws.UpsertPairwise(workspace.PairwiseRow{
		Matrix: "alt:quality", Left: "acme", Right: "globex", Value: 9, Status: "proposal",
	}))

	plan, err := ws.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Proposals != 1 {
		t.Fatalf("proposals=%d", plan.Proposals)
	}
	if plan.Title != "Plan demo" {
		t.Fatalf("title=%q", plan.Title)
	}
	if len(plan.RankDeltas) == 0 {
		t.Fatal("expected rank deltas")
	}
	// Applying proposal must not mutate CSV status.
	rows, err := ws.Pairwise()
	if err != nil {
		t.Fatal(err)
	}
	foundProposal := false
	for _, r := range rows {
		if r.Matrix == "alt:quality" && r.Status == "proposal" {
			foundProposal = true
		}
	}
	if !foundProposal {
		t.Fatal("plan must not commit proposals")
	}
}
