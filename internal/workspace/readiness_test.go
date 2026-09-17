package workspace_test

import (
	"testing"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
)

func TestEmptyWorkspaceNotComplete(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete {
		t.Fatal("expected complete=false on empty init workspace")
	}
	if result.Consistent {
		t.Fatal("expected consistent=false on empty init workspace")
	}
	s, err := ws.Status(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Complete {
		t.Fatal("status complete should be false")
	}
	if s.ExitCode != cliout.ExitIncomplete {
		t.Fatalf("exit_code=%d want %d", s.ExitCode, cliout.ExitIncomplete)
	}
	if s.Next.Kind != "structure" {
		t.Fatalf("next=%+v", s.Next)
	}
}

func TestEqualWeightRankingLabeled(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	mustReady(t, ws)
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "b", Name: "B"})

	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete {
		t.Fatal("expected incomplete")
	}
	if result.RankingMode != workspace.RankingModeEqualFallback {
		t.Fatalf("ranking_mode=%q", result.RankingMode)
	}
	if len(result.Ranking) != 2 {
		t.Fatalf("ranking len=%d", len(result.Ranking))
	}
	s, err := ws.Status(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.RankingMode != workspace.RankingModeEqualFallback {
		t.Fatalf("status ranking_mode=%q", s.RankingMode)
	}
}

func TestProposalsFillGapsNextIsPlan(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	mustReady(t, ws)
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "b", Name: "B"})

	pairs := []workspace.PairwiseRow{
		{Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: "proposal"},
		{Matrix: "alt:cost", Left: "a", Right: "b", Value: 2, Status: "proposal"},
		{Matrix: "alt:quality", Left: "a", Right: "b", Value: 4, Status: "proposal"},
	}
	for _, p := range pairs {
		if err := ws.UpsertPairwise(p); err != nil {
			t.Fatal(err)
		}
	}

	s, err := ws.Status(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Complete {
		t.Fatal("committed view should still be incomplete")
	}
	if !s.ProposalsFillGaps {
		t.Fatalf("expected proposals_fill_gaps; covered=%d uncovered=%d missing=%d",
			len(s.CoveredByProposals), len(s.Uncovered), len(s.MissingCommitted))
	}
	if len(s.Uncovered) != 0 {
		t.Fatalf("uncovered=%v", s.Uncovered)
	}
	if len(s.CoveredByProposals) != 3 {
		t.Fatalf("covered=%d", len(s.CoveredByProposals))
	}
	if s.Next.Kind != "proposals" || s.Next.Command != "ahp plan" {
		t.Fatalf("next=%+v", s.Next)
	}
	if s.ExitCode != cliout.ExitProposals {
		t.Fatalf("exit_code=%d want %d", s.ExitCode, cliout.ExitProposals)
	}
}

func TestCompleteAHPRankingMode(t *testing.T) {
	dir := t.TempDir()
	ws := workspace.Open(dir)
	mustReady(t, ws)
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "cost", Name: "Cost"})
	_ = ws.UpsertCriterion(workspace.Criterion{ID: "quality", Name: "Quality"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "a", Name: "A"})
	_ = ws.UpsertAlternative(workspace.Alternative{ID: "b", Name: "B"})
	for _, p := range []workspace.PairwiseRow{
		{Matrix: "criteria", Left: "cost", Right: "quality", Value: 3, Status: "committed"},
		{Matrix: "alt:cost", Left: "a", Right: "b", Value: 2, Status: "committed"},
		{Matrix: "alt:quality", Left: "a", Right: "b", Value: 4, Status: "committed"},
	} {
		if err := ws.UpsertPairwise(p); err != nil {
			t.Fatal(err)
		}
	}
	result, err := ws.Compute(false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.RankingMode != workspace.RankingModeAHP {
		t.Fatalf("complete=%v mode=%q", result.Complete, result.RankingMode)
	}
}

func mustReady(t *testing.T, ws *workspace.Workspace) {
	t.Helper()
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
}
